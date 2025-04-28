package lru

import (
	"container/list"
	"fmt"
	"runtime"
	"sync"
	"time"
)

// DefaultCacheSize 是缓存大小的默认值
const DefaultCacheSize = 10

// Cache 是线程安全的LRU缓存，支持过期时间和自动清理
type Cache[K comparable, V any] struct {
	mu              sync.RWMutex        // 读写互斥锁，保证并发安全
	items           map[K]*list.Element // 存储键到链表节点的映射，用于O(1)时间复杂度查找
	list            *list.List          // 双向链表，用于维护LRU顺序
	size            int                 // 缓存的最大容量
	ttl             time.Duration       // 缓存项的默认过期时间
	cleanerStopCh   chan struct{}       // 用于停止清理协程的信号通道
	cleanerInterval time.Duration       // 自动清理的时间间隔
}

// entry 表示缓存中的条目
type entry[K comparable, V any] struct {
	key      K         // 缓存项的键
	value    V         // 缓存项的值
	expireAt time.Time // 缓存项的过期时间点，零值表示永不过期
}

// entryOption 提供单个缓存项的链式操作
type entryOption[K comparable, V any] struct {
	key   K            // 操作的缓存项键
	cache *Cache[K, V] // 指向所属缓存的引用
}

// New 创建指定大小的缓存
// 参数 size: 缓存的最大容量，当容量满时会淘汰最久未使用的项
// 如果 size <= 0，则使用默认容量DefaultCacheSize
func New[K comparable, V any](size int) *Cache[K, V] {
	if size <= 0 {
		size = DefaultCacheSize // 使用默认缓存大小
	}
	return &Cache[K, V]{
		size:          size,
		items:         make(map[K]*list.Element),
		list:          list.New(),
		cleanerStopCh: make(chan struct{}),
	}
}

// TTL 设置默认过期时间
// 参数 duration: 所有新缓存项的默认生存时间
// 返回缓存实例本身，支持链式调用
func (c *Cache[K, V]) TTL(duration time.Duration) *Cache[K, V] {
	c.mu.Lock()
	c.ttl = duration
	c.mu.Unlock()
	return c
}

// Cleaner 设置自动清理过期项的时间间隔
// 参数 interval: 清理过期项的时间间隔
// 返回缓存实例本身，支持链式调用
// 注意: 使用此方法后，不再使用缓存时应调用Close方法停止清理goroutine，
// 否则可能导致资源泄漏。推荐使用defer cache.Close()
func (c *Cache[K, V]) Cleaner(interval time.Duration) *Cache[K, V] {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.startCleaner(interval)

	// 设置finalizer，防止用户忘记调用Close方法
	runtime.SetFinalizer(c, func(c *Cache[K, V]) {
		// 避免在finalizer中持有对象引用，创建一个本地副本
		stopCh := c.cleanerStopCh
		if stopCh != nil {
			close(stopCh)
		}
	})

	return c
}

// startCleaner 启动自动清理器
// 参数 interval: 清理的时间间隔
// 会先停止现有的清理器（如果有），然后启动新的清理协程
func (c *Cache[K, V]) startCleaner(interval time.Duration) {
	// 停止现有的清理器
	if c.cleanerStopCh != nil {
		close(c.cleanerStopCh)
	}

	c.cleanerInterval = interval
	c.cleanerStopCh = make(chan struct{})

	go c.cleanerLoop()
}

// cleanerLoop 定时清理过期元素
// 内部使用，作为协程运行，会定期调用Purge方法清理过期项
// 支持panic恢复，确保清理协程不会意外终止
func (c *Cache[K, V]) cleanerLoop() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("缓存清理协程崩溃: %v\n", r)
			// 可选：重启清理器
			go c.cleanerLoop()
		}
	}()

	ticker := time.NewTicker(c.cleanerInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.Purge()

		case <-c.cleanerStopCh:
			return
		}
	}
}

// Close 停止自动清理
// 如果清理器正在运行，则停止它
// 当不再使用缓存时，应当调用此方法释放资源
// 建议使用defer语句确保资源被释放: defer cache.Close()
func (c *Cache[K, V]) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cleanerStopCh != nil {
		close(c.cleanerStopCh)
		c.cleanerStopCh = nil
	}

	// 取消finalizer
	runtime.SetFinalizer(c, nil)
}

// Purge 清理所有过期项，返回清理的项数
// 线程安全，会遍历缓存中的所有项并删除已过期的
// 返回值: 清理的项数
func (c *Cache[K, V]) Purge() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	count := 0

	for e := c.list.Front(); e != nil; {
		next := e.Next()
		item := e.Value.(entry[K, V])
		if !item.expireAt.IsZero() && now.After(item.expireAt) {
			c.removeElement(e)
			count++
		}
		e = next
	}

	return count
}

// Set 添加或更新缓存项，返回链式调用句柄
// 参数 key: 缓存项的键
// 参数 value: 缓存项的值
// 返回值: 指向该缓存项的句柄，可用于进一步设置过期时间
// 如果添加新项导致缓存超出容量，会删除最久未使用的项
func (c *Cache[K, V]) Set(key K, value V) *entryOption[K, V] {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 计算新的过期时间
	var expireAt time.Time
	if c.ttl > 0 {
		expireAt = time.Now().Add(c.ttl)
	}

	if e, ok := c.items[key]; ok {
		// 更新项 - 延长过期时间(除非原项永不过期)
		item := e.Value.(entry[K, V])
		if !item.expireAt.IsZero() { // 仅当原项有过期时间时更新
			e.Value = entry[K, V]{key, value, expireAt}
		} else {
			e.Value = entry[K, V]{key, value, time.Time{}} // 保持永不过期
		}
		c.list.MoveToFront(e)
	} else {
		// 新增项 - 使用计算的过期时间
		e := c.list.PushFront(entry[K, V]{key, value, expireAt})
		c.items[key] = e

		if c.list.Len() > c.size {
			c.removeOldest()
		}
	}

	return &entryOption[K, V]{key: key, cache: c}
}

// Expire 为单个缓存项设置过期时间
// 参数 duration: 过期时间，如果为0或负值则表示永不过期
// 返回值: 指向该缓存项的句柄，支持链式调用
func (h *entryOption[K, V]) Expire(duration time.Duration) *entryOption[K, V] {
	h.cache.mu.Lock()
	defer h.cache.mu.Unlock()

	if e, ok := h.cache.items[h.key]; ok {
		item := e.Value.(entry[K, V])
		expireAt := time.Time{}
		if duration > 0 {
			expireAt = time.Now().Add(duration)
		}
		e.Value = entry[K, V]{item.key, item.value, expireAt}
	}

	return h
}

// Get 获取缓存项的值，如果不存在或已过期则返回零值和false
// 参数 key: 要获取的缓存项键
// 返回值: 缓存项的值和是否存在/有效的标志
// 注意: 成功获取会将该项移到最近使用位置
func (c *Cache[K, V]) Get(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.get(key, true)
}

// get 内部获取方法，控制是否更新位置
// 参数 key: 要获取的缓存项键
// 参数 updatePos: 是否更新项在链表中的位置（移到最前）
// 返回值: 缓存项的值和是否存在/有效的标志
func (c *Cache[K, V]) get(key K, updatePos bool) (V, bool) {
	if e, ok := c.items[key]; ok {
		item := e.Value.(entry[K, V])
		// 检查是否过期
		if item.expireAt.IsZero() || time.Now().Before(item.expireAt) {
			if updatePos {
				c.list.MoveToFront(e)
			}
			return item.value, true
		}
		// 已过期，删除
		c.removeElement(e)
	}
	var zero V
	return zero, false
}

// Peek 获取值但不更新位置
// 参数 key: 要获取的缓存项键
// 返回值: 缓存项的值和是否存在/有效的标志
// 与Get不同，不会影响项的LRU顺序
func (c *Cache[K, V]) Peek(key K) (V, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.get(key, false)
}

// Delete 删除缓存项
// 参数 key: 要删除的缓存项键
// 返回值: 是否找到并删除了该项
func (c *Cache[K, V]) Delete(key K) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if e, ok := c.items[key]; ok {
		c.removeElement(e)
		return true
	}
	return false
}

// Size 返回当前缓存中的项数
// 返回值: 当前缓存中有效项的数量
func (c *Cache[K, V]) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.list.Len()
}

// Capacity 返回缓存容量
// 返回值: 缓存的最大容量
func (c *Cache[K, V]) Capacity() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.size
}

// SetCapacity 调整缓存容量
// 参数 size: 新的缓存容量
// 如果参数无效（小于等于0），会使用默认容量DefaultCacheSize
// 如果新容量小于当前项数，会删除最久未使用的项直到符合新容量
func (c *Cache[K, V]) SetCapacity(size int) {
	if size <= 0 {
		size = DefaultCacheSize
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.size = size
	// 如果当前大小超过新容量，移除多余项
	for c.list.Len() > c.size {
		c.removeOldest()
	}
}

// Keys 返回所有未过期的键
// 返回值: 包含所有未过期键的切片，按照最近使用顺序排列
func (c *Cache[K, V]) Keys() []K {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := make([]K, 0, c.list.Len())
	now := time.Now()

	for e := c.list.Front(); e != nil; e = e.Next() {
		item := e.Value.(entry[K, V])
		if item.expireAt.IsZero() || now.Before(item.expireAt) {
			keys = append(keys, item.key)
		}
	}

	return keys
}

// Range 遍历所有未过期的缓存项
// 参数 fn: 对每个有效缓存项调用的函数，返回false可停止遍历
// 遍历过程是按照最近使用顺序进行的
func (c *Cache[K, V]) Range(fn func(K, V) bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	now := time.Now()
	for e := c.list.Front(); e != nil; e = e.Next() {
		item := e.Value.(entry[K, V])
		if item.expireAt.IsZero() || now.Before(item.expireAt) {
			if !fn(item.key, item.value) {
				break
			}
		}
	}
}

// Clear 清空缓存
// 删除缓存中的所有项
func (c *Cache[K, V]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.list.Init()
	c.items = make(map[K]*list.Element)
}

// removeOldest 删除最久未使用的项
// 内部方法，从链表尾部删除元素
// 调用前必须持有锁
func (c *Cache[K, V]) removeOldest() {
	if e := c.list.Back(); e != nil {
		c.removeElement(e)
	}
}

// removeElement 从缓存中删除元素
// 参数 e: 要删除的链表元素
// 内部方法，从链表和映射中删除指定元素
// 调用前必须持有锁
func (c *Cache[K, V]) removeElement(e *list.Element) {
	c.list.Remove(e)
	item := e.Value.(entry[K, V])
	delete(c.items, item.key)
}
