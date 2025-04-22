package lru

import (
	"container/list"
	"sync"
)

// Lru 是一个线程安全的 LRU 缓存
// 缓存有一个固定的大小, 当缓存已满时, 最近最少使用的元素将被删除为新元素腾出空间
type Lru[K comparable, V any] struct {
	lock  sync.RWMutex        // 读写锁, 用于并发读写操作的保护
	items map[K]*list.Element // 哈希表, 用于存储及快速查找元素
	list  *list.List          // 双向链表, 用于按照元素的访问时间排序
	size  int                 // 缓存的大小, 最多可以存储的元素个数
}

// item 是一个键值对, 用于存储缓存中的元素
type item[K comparable, V any] struct {
	k K // 元素的键
	v V // 元素的值
}

// NewLru 创建一个指定大小的 LRU 缓存
func NewLru[K comparable, V any](size int) *Lru[K, V] {
	return &Lru[K, V]{
		size:  size,
		items: make(map[K]*list.Element),
		list:  list.New(),
	}
}

// Set 向缓存中添加一个元素
// 如果元素已经存在于缓存中, 则将其移动到链表的头部, 表示最近使用过
// 如果缓存已满, 则删除最近最少使用的元素, 再添加新的元素
func (lru *Lru[K, V]) Set(k K, v V) {
	lru.lock.Lock()
	defer lru.lock.Unlock()

	// 读取已有的节点
	if e, ok := lru.items[k]; ok {
		e.Value = item[K, V]{k, v}
		lru.list.MoveToFront(e)
		return
	}

	// 创建新的节点
	e := lru.list.PushFront(item[K, V]{k, v})
	lru.items[k] = e

	// 如果缓存已满, 则删除最近最少使用的元素
	if lru.list.Len() > lru.size {
		b := lru.list.Back()
		if b != nil {
			lru.list.Remove(b)
			kv := b.Value.(item[K, V])
			delete(lru.items, kv.k)
		}
	}
}

// Get 从缓存中获取指定键对应的值
// 如果键存在于缓存中, 则将对应元素移动到链表的头部并返回 对应值 和 true
// 如果键不存在于缓存中, 则返回 nil 和 false
func (lru *Lru[K, V]) Get(k K) (V, bool) {
	lru.lock.Lock()
	defer lru.lock.Unlock()

	// 读取已有的节点
	e, ok := lru.items[k]
	if ok {
		lru.list.MoveToFront(e)
		return e.Value.(item[K, V]).v, ok
	}

	// 如果键不存在于缓存中, 则返回 nil, false
	// return reflect.Zero(reflect.TypeOf((*V)(nil)).Elem()).Interface().(V), ok
	// 创建了类型 V 的零值, 避免了使用反射, 在性能上更优
	var z V
	return z, ok
}

// Put 更新缓存中元素的值, 但不更新元素的位置
// 如果元素已经存在于缓存中, 则更新元素的值, 但不更新元素的位置
// 如果元素不存在于缓存中并且缓存已满, 则删除最近最少使用的元素再添加新元素
// 如果元素不存在于缓存中并且缓存未满, 则添加新的元素
func (lru *Lru[K, V]) Put(k K, v V) {
	lru.lock.Lock()
	defer lru.lock.Unlock()

	// 如果元素已经存在于缓存中, 则更新元素的值, 但不更新元素的位置
	if e, ok := lru.items[k]; ok {
		// 更新元素的值，但不改变其在链表中的位置
		e.Value = item[K, V]{k, v}
		return
	}

	// 创建新的节点
	e := lru.list.PushFront(item[K, V]{k, v})
	lru.items[k] = e

	// 如果缓存已满, 则删除最近最少使用的元素
	if lru.list.Len() > lru.size {
		b := lru.list.Back()
		if b != nil {
			lru.list.Remove(b)
			kv := b.Value.(item[K, V])
			delete(lru.items, kv.k)
		}
	}
}

// Take 获取指定键对应的值, 但不会修改元素的位置
// 如果键存在于缓存中, 则返回 true 和对应的值
// 如果键不存在于缓存中, 则返回 false 和 nil
func (lru *Lru[K, V]) Take(k K) (V, bool) {
	lru.lock.RLock()
	defer lru.lock.RUnlock()

	// 读取已有的节点
	e, ok := lru.items[k]
	if ok {
		return e.Value.(item[K, V]).v, ok
	}

	// 如果键不存在于缓存中, 则返回 nil, false
	// return reflect.Zero(reflect.TypeOf((*V)(nil)).Elem()).Interface().(V), ok
	// 创建了类型 V 的零值, 避免了使用反射, 在性能上更优
	var z V
	return z, ok
}

// Del 从缓存中删除指定键对应的元素
// 如果键存在于缓存中, 则删除对应元素并返回 true
// 如果键不存在于缓存中, 则返回 false
func (lru *Lru[K, V]) Del(k K) bool {
	lru.lock.Lock()
	defer lru.lock.Unlock()

	// 读取已有的节点
	if e, ok := lru.items[k]; ok {
		lru.list.Remove(e)
		delete(lru.items, k)
		return true
	}

	return false
}

// Len 返回缓存中元素的个数
func (lru *Lru[K, V]) Len() int {
	lru.lock.RLock()
	defer lru.lock.RUnlock()
	return lru.list.Len()
}

// Range 遍历缓存中的所有元素
// 遍历顺序为从链表头部到尾部, 即最近访问的元素在前
// 如果返回 false, 则停止遍历
func (lru *Lru[K, V]) Range(f func(k, v any) bool) {
	lru.lock.RLock()
	defer lru.lock.RUnlock()

	for e := lru.list.Front(); e != nil; e = e.Next() {
		item := e.Value.(item[K, V])
		if !f(item.k, item.v) {
			break
		}
	}
}

// Flush 删除所有元素
// 该方法不会释放缓存的内存, 仅仅是将缓存中的元素全部删除
// 如果需要释放缓存的内存, 则需要将缓存的指针设置为 nil
// 例如: lru = nil
func (lru *Lru[K, V]) Flush() {
	lru.lock.Lock()
	defer lru.lock.Unlock()

	// 显式地删除 items 中的键值对, 否则某些类型的键值对仍然会占用内存, 比如map
	for k := range lru.items {
		delete(lru.items, k)
	}

	lru.list.Init()
}
