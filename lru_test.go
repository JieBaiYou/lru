package lru

import (
	"fmt"
	"sync"
	"testing"
)

// 测试创建LRU缓存
func TestNewLru(t *testing.T) {
	t.Log("🔍 测试: 创建新的LRU缓存")
	lru := NewLru[string, int](5)
	if lru == nil {
		t.Fatal("❌ 创建LRU缓存失败")
	}
	if lru.size != 5 {
		t.Errorf("❌ 缓存容量不匹配: 期望 5, 实际 %d", lru.size)
	}
	if lru.list == nil {
		t.Error("❌ 链表未初始化")
	}
	if lru.items == nil {
		t.Error("❌ 哈希表未初始化")
	}
	t.Log("✅ LRU缓存创建成功，容量为:", lru.size)
}

// 测试Set方法
func TestLruSet(t *testing.T) {
	t.Log("🔍 测试: Set方法和LRU淘汰机制")
	lru := NewLru[string, int](3)
	t.Log("📌 创建容量为3的LRU缓存")

	// 添加元素
	t.Log("➕ 添加元素: a=1, b=2, c=3")
	lru.Set("a", 1)
	lru.Set("b", 2)
	lru.Set("c", 3)

	// 打印当前缓存状态
	t.Log("🔄 当前缓存状态:")
	printCacheStatus(t, lru)

	// 验证元素是否正确添加
	if lru.Len() != 3 {
		t.Errorf("❌ 缓存长度不匹配: 期望 3, 实际 %d", lru.Len())
	} else {
		t.Log("✅ 缓存长度正确: 3")
	}

	// 验证淘汰机制
	t.Log("➕ 添加第四个元素 d=4，应淘汰最旧元素 a")
	lru.Set("d", 4)

	t.Log("🔄 添加d后的缓存状态:")
	printCacheStatus(t, lru)

	// 检查最旧的元素是否被淘汰
	if v, ok := lru.Get("a"); ok {
		t.Errorf("❌ 元素'a'应该被淘汰，但仍能获取到: %v", v)
	} else {
		t.Log("✅ 元素'a'已被正确淘汰")
	}

	// 检查其他元素是否还在
	if _, ok := lru.Get("b"); !ok {
		t.Error("❌ 元素'b'应该在缓存中，但未找到")
	} else {
		t.Log("✅ 元素'b'仍在缓存中")
	}
	if _, ok := lru.Get("c"); !ok {
		t.Error("❌ 元素'c'应该在缓存中，但未找到")
	} else {
		t.Log("✅ 元素'c'仍在缓存中")
	}
	if _, ok := lru.Get("d"); !ok {
		t.Error("❌ 元素'd'应该在缓存中，但未找到")
	} else {
		t.Log("✅ 元素'd'已添加到缓存中")
	}

	// 验证更新已存在元素
	t.Log("🔄 更新已存在元素: b=20")
	lru.Set("b", 20)

	t.Log("🔄 更新b后的缓存状态:")
	printCacheStatus(t, lru)

	if v, ok := lru.Get("b"); !ok || v != 20 {
		t.Errorf("❌ 元素'b'更新失败: 期望 20, 实际 %v, 存在状态 %v", v, ok)
	} else {
		t.Log("✅ 元素'b'已更新为20")
	}
}

// 测试Get方法
func TestLruGet(t *testing.T) {
	t.Log("🔍 测试: Get方法和元素访问更新机制")
	lru := NewLru[string, int](3)
	t.Log("📌 创建容量为3的LRU缓存")

	t.Log("➕ 添加元素: a=1, b=2")
	lru.Set("a", 1)
	lru.Set("b", 2)

	t.Log("🔄 初始缓存状态:")
	printCacheStatus(t, lru)

	// 获取存在的元素
	t.Log("🔍 获取元素'a'")
	if v, ok := lru.Get("a"); !ok || v != 1 {
		t.Errorf("❌ 获取元素'a'失败: 期望 1, 实际 %v, 存在状态 %v", v, ok)
	} else {
		t.Log("✅ 成功获取元素'a' =", v)
	}

	t.Log("🔄 获取a后的缓存状态(应该移到最前):")
	printCacheStatus(t, lru)

	// 获取不存在的元素
	t.Log("🔍 尝试获取不存在的元素'c'")
	if v, ok := lru.Get("c"); ok {
		t.Errorf("❌ 不应获取到不存在的元素'c': %v", v)
	} else {
		t.Log("✅ 正确返回元素'c'不存在")
	}

	// 验证LRU机制 - 添加第三个元素
	t.Log("➕ 添加第三个元素 c=3")
	lru.Set("c", 3)
	t.Log("🔄 添加c后的缓存状态:")
	printCacheStatus(t, lru)

	// 访问最旧的元素"a"，使其变为最新
	t.Log("🔍 再次获取元素'a'，使其成为最近使用的元素")
	lru.Get("a")
	t.Log("🔄 再次获取a后的缓存状态(a应该移到最前):")
	printCacheStatus(t, lru)

	// 添加第四个元素，应该淘汰"b"而不是"a"
	t.Log("➕ 添加第四个元素 d=4，应淘汰元素'b'而非'a'")
	lru.Set("d", 4)
	t.Log("🔄 添加d后的缓存状态:")
	printCacheStatus(t, lru)

	// 检查"a"是否还在
	if _, ok := lru.Get("a"); !ok {
		t.Error("❌ 元素'a'应该仍在缓存中，但未找到")
	} else {
		t.Log("✅ 元素'a'仍在缓存中")
	}

	// 检查"b"是否被淘汰
	if _, ok := lru.Get("b"); ok {
		t.Error("❌ 元素'b'应该被淘汰，但仍能获取到")
	} else {
		t.Log("✅ 元素'b'已被正确淘汰")
	}
}

// 测试Put方法
func TestLruPut(t *testing.T) {
	t.Log("🔍 测试: Put方法(不更新位置的设置)")
	lru := NewLru[string, int](3)
	t.Log("📌 创建容量为3的LRU缓存")

	// 添加元素
	t.Log("➕ 通过Put添加元素: a=1, b=2, c=3")
	lru.Put("a", 1)
	lru.Put("b", 2)
	lru.Put("c", 3)

	t.Log("🔄 初始缓存状态:")
	printCacheStatus(t, lru)

	// 验证元素是否正确添加
	if lru.Len() != 3 {
		t.Errorf("❌ 缓存长度不匹配: 期望 3, 实际 %d", lru.Len())
	} else {
		t.Log("✅ 缓存长度正确: 3")
	}

	// 验证淘汰机制
	t.Log("➕ 添加第四个元素 d=4，应淘汰最旧元素'a'")
	lru.Put("d", 4)
	t.Log("🔄 添加d后的缓存状态:")
	printCacheStatus(t, lru)

	// 检查最旧的元素是否被淘汰
	if v, ok := lru.Get("a"); ok {
		t.Errorf("❌ 元素'a'应该被淘汰，但仍能获取到: %v", v)
	} else {
		t.Log("✅ 元素'a'已被正确淘汰")
	}

	// 更新元素但不改变位置
	t.Log("🔍 先通过Get访问元素'b'，使其变为最新")
	lru.Get("b")
	t.Log("🔄 获取b后的缓存状态(b应该移到最前):")
	printCacheStatus(t, lru)

	t.Log("🔍 再通过Get访问元素'c'，使其变为最新")
	lru.Get("c")
	t.Log("🔄 获取c后的缓存状态(c应该移到最前):")
	printCacheStatus(t, lru)

	t.Log("🔄 通过Put更新元素'b'的值为20，但不改变其位置")
	lru.Put("b", 20)
	t.Log("🔄 Put更新b后的缓存状态(b位置不变，仅值更新):")
	printCacheStatus(t, lru)

	// 添加新元素，应该淘汰"d"而不是"b"
	t.Log("➕ 添加第五个元素 e=5，应淘汰最旧元素'd'而非'b'")
	lru.Put("e", 5)
	t.Log("🔄 添加e后的缓存状态:")
	printCacheStatus(t, lru)

	// 检查"d"是否被淘汰
	if _, ok := lru.Get("d"); ok {
		t.Error("❌ 元素'd'应该被淘汰，但仍能获取到")
	} else {
		t.Log("✅ 元素'd'已被正确淘汰")
	}

	// 检查"b"是否还在，且值已更新
	if v, ok := lru.Get("b"); !ok || v != 20 {
		t.Errorf("❌ 元素'b'应该在缓存中且值为20，但得到: %v, 存在状态 %v", v, ok)
	} else {
		t.Log("✅ 元素'b'仍在缓存中且值已更新为20")
	}
}

// 测试Take方法
func TestLruTake(t *testing.T) {
	t.Log("🔍 测试: Take方法(不更新位置的获取)")
	lru := NewLru[string, int](3)
	t.Log("📌 创建容量为3的LRU缓存")

	t.Log("➕ 添加元素: a=1, b=2")
	lru.Set("a", 1)
	lru.Set("b", 2)

	t.Log("🔄 初始缓存状态:")
	printCacheStatus(t, lru)

	// 获取存在的元素但不更新位置
	t.Log("🔍 通过Take获取元素'a'，不应改变其位置")
	if v, ok := lru.Take("a"); !ok || v != 1 {
		t.Errorf("❌ Take获取元素'a'失败: 期望 1, 实际 %v, 存在状态 %v", v, ok)
	} else {
		t.Log("✅ 成功通过Take获取元素'a' =", v)
	}

	t.Log("🔄 Take获取a后的缓存状态(a位置不应改变):")
	printCacheStatus(t, lru)

	// 获取不存在的元素
	t.Log("🔍 尝试通过Take获取不存在的元素'c'")
	if v, ok := lru.Take("c"); ok {
		t.Errorf("❌ 不应获取到不存在的元素'c': %v", v)
	} else {
		t.Log("✅ 正确返回元素'c'不存在")
	}

	// 验证Take不改变元素位置 - 添加第三个元素
	t.Log("➕ 添加第三个元素 c=3")
	lru.Set("c", 3)
	t.Log("🔄 添加c后的缓存状态:")
	printCacheStatus(t, lru)

	// 访问但不更新最旧的元素"a"
	t.Log("🔍 再次通过Take获取元素'a'，不应改变其位置")
	lru.Take("a")
	t.Log("🔄 Take获取a后的缓存状态(a位置不应改变):")
	printCacheStatus(t, lru)

	// 添加第四个元素，应该淘汰"a"而不是"b"
	t.Log("➕ 添加第四个元素 d=4，应淘汰最旧元素'a'而非'b'")
	lru.Set("d", 4)
	t.Log("🔄 添加d后的缓存状态:")
	printCacheStatus(t, lru)

	// 检查"a"是否被淘汰
	if _, ok := lru.Get("a"); ok {
		t.Error("❌ 元素'a'应该被淘汰，但仍能获取到")
	} else {
		t.Log("✅ 元素'a'已被正确淘汰")
	}

	// 检查"b"是否还在
	if _, ok := lru.Get("b"); !ok {
		t.Error("❌ 元素'b'应该仍在缓存中，但未找到")
	} else {
		t.Log("✅ 元素'b'仍在缓存中")
	}
}

// 测试Del方法
func TestLruDel(t *testing.T) {
	t.Log("🔍 测试: Del方法(删除元素)")
	lru := NewLru[string, int](3)
	t.Log("📌 创建容量为3的LRU缓存")

	t.Log("➕ 添加元素: a=1, b=2")
	lru.Set("a", 1)
	lru.Set("b", 2)

	t.Log("🔄 初始缓存状态:")
	printCacheStatus(t, lru)

	// 删除存在的元素
	t.Log("🗑️ 删除元素'a'")
	if deleted := lru.Del("a"); !deleted {
		t.Error("❌ 删除元素'a'失败")
	} else {
		t.Log("✅ 成功删除元素'a'")
	}

	t.Log("🔄 删除a后的缓存状态:")
	printCacheStatus(t, lru)

	// 验证元素是否已删除
	if _, ok := lru.Get("a"); ok {
		t.Error("❌ 元素'a'应该已被删除，但仍能获取到")
	} else {
		t.Log("✅ 元素'a'已被正确删除")
	}

	// 删除不存在的元素
	t.Log("🗑️ 尝试删除不存在的元素'c'")
	if deleted := lru.Del("c"); deleted {
		t.Error("❌ 删除不存在的元素'c'应返回false，但返回true")
	} else {
		t.Log("✅ 正确返回不存在元素'c'的删除结果")
	}

	// 验证长度
	if lru.Len() != 1 {
		t.Errorf("❌ 缓存长度不匹配: 期望 1, 实际 %d", lru.Len())
	} else {
		t.Log("✅ 删除后缓存长度正确: 1")
	}
}

// 测试Len方法
func TestLruLen(t *testing.T) {
	t.Log("🔍 测试: Len方法(获取缓存长度)")
	lru := NewLru[string, int](3)
	t.Log("📌 创建容量为3的LRU缓存")

	// 空缓存
	if lru.Len() != 0 {
		t.Errorf("❌ 空缓存长度不为0: %d", lru.Len())
	} else {
		t.Log("✅ 空缓存长度正确: 0")
	}

	// 添加元素
	t.Log("➕ 添加元素: a=1, b=2")
	lru.Set("a", 1)
	lru.Set("b", 2)

	t.Log("🔄 添加两个元素后的缓存状态:")
	printCacheStatus(t, lru)

	// 验证长度
	if lru.Len() != 2 {
		t.Errorf("❌ 缓存长度不匹配: 期望 2, 实际 %d", lru.Len())
	} else {
		t.Log("✅ 添加两个元素后缓存长度正确: 2")
	}

	// 删除元素
	t.Log("🗑️ 删除元素'a'")
	lru.Del("a")

	t.Log("🔄 删除a后的缓存状态:")
	printCacheStatus(t, lru)

	// 验证长度
	if lru.Len() != 1 {
		t.Errorf("❌ 缓存长度不匹配: 期望 1, 实际 %d", lru.Len())
	} else {
		t.Log("✅ 删除一个元素后缓存长度正确: 1")
	}
}

// 测试Range方法
func TestLruRange(t *testing.T) {
	t.Log("🔍 测试: Range方法(遍历缓存)")
	lru := NewLru[string, int](3)
	t.Log("📌 创建容量为3的LRU缓存")

	t.Log("➕ 添加元素: a=1, b=2, c=3")
	lru.Set("a", 1)
	lru.Set("b", 2)
	lru.Set("c", 3)

	t.Log("🔄 缓存状态:")
	printCacheStatus(t, lru)

	// 收集遍历结果
	t.Log("🔍 开始遍历缓存元素")
	keys := make([]string, 0)
	values := make([]int, 0)

	lru.Range(func(k, v any) bool {
		key := k.(string)
		value := v.(int)
		t.Logf("   遍历元素: %s = %d", key, value)
		keys = append(keys, key)
		values = append(values, value)
		return true
	})

	// 验证元素个数
	if len(keys) != 3 || len(values) != 3 {
		t.Errorf("❌ 遍历元素数量不正确: 期望 3, 实际获得 %d 个键和 %d 个值", len(keys), len(values))
	} else {
		t.Log("✅ 遍历元素数量正确: 3")
	}

	// 测试提前终止遍历
	t.Log("🔍 测试提前终止遍历（只遍历前两个元素）")
	count := 0
	lru.Range(func(k, v any) bool {
		key := k.(string)
		value := v.(int)
		count++
		t.Logf("   遍历元素 %d: %s = %d", count, key, value)
		return count < 2 // 只遍历两个元素
	})

	if count != 2 {
		t.Errorf("❌ 提前终止遍历失败: 期望遍历 2 个元素, 实际遍历了 %d 个", count)
	} else {
		t.Log("✅ 提前终止遍历正确: 只遍历了2个元素")
	}
}

// 测试Flush方法
func TestLruFlush(t *testing.T) {
	t.Log("🔍 测试: Flush方法(清空缓存)")
	lru := NewLru[string, int](3)
	t.Log("📌 创建容量为3的LRU缓存")

	t.Log("➕ 添加元素: a=1, b=2, c=3")
	lru.Set("a", 1)
	lru.Set("b", 2)
	lru.Set("c", 3)

	t.Log("🔄 初始缓存状态:")
	printCacheStatus(t, lru)

	// 清空缓存
	t.Log("🧹 清空缓存")
	lru.Flush()

	t.Log("🔄 清空后的缓存状态:")
	printCacheStatus(t, lru)

	// 验证长度
	if lru.Len() != 0 {
		t.Errorf("❌ 清空后缓存长度不为0: %d", lru.Len())
	} else {
		t.Log("✅ 清空后缓存长度正确: 0")
	}

	// 验证元素是否已删除
	if _, ok := lru.Get("a"); ok {
		t.Error("❌ 元素'a'清空后仍能获取到")
	} else {
		t.Log("✅ 元素'a'已被正确清空")
	}
	if _, ok := lru.Get("b"); ok {
		t.Error("❌ 元素'b'清空后仍能获取到")
	} else {
		t.Log("✅ 元素'b'已被正确清空")
	}
	if _, ok := lru.Get("c"); ok {
		t.Error("❌ 元素'c'清空后仍能获取到")
	} else {
		t.Log("✅ 元素'c'已被正确清空")
	}

	// 清空后添加新元素
	t.Log("➕ 清空后添加新元素: x=100")
	lru.Set("x", 100)

	t.Log("🔄 添加新元素后的缓存状态:")
	printCacheStatus(t, lru)

	// 验证新元素是否正确添加
	if v, ok := lru.Get("x"); !ok || v != 100 {
		t.Errorf("❌ 清空后添加新元素失败: 期望 100, 实际 %v, 存在状态 %v", v, ok)
	} else {
		t.Log("✅ 清空后成功添加新元素 x=100")
	}
}

// 测试边界条件 - 容量为0
func TestLruZeroCapacity(t *testing.T) {
	t.Log("🔍 测试: 边界条件 - 容量为0")
	lru := NewLru[string, int](0)
	t.Log("📌 创建容量为0的LRU缓存")

	// 尝试添加元素
	t.Log("➕ 尝试添加元素: a=1")
	lru.Set("a", 1)

	t.Log("🔄 添加元素后的缓存状态:")
	printCacheStatus(t, lru)

	// 验证元素是否被添加
	if _, ok := lru.Get("a"); ok {
		t.Error("❌ 容量为0的缓存不应该添加元素，但元素'a'被添加了")
	} else {
		t.Log("✅ 容量为0的缓存正确地拒绝了元素'a'")
	}

	// 验证长度
	if lru.Len() != 0 {
		t.Errorf("❌ 容量为0的缓存长度不为0: %d", lru.Len())
	} else {
		t.Log("✅ 容量为0的缓存长度正确: 0")
	}
}

// 测试边界条件 - 容量为1
func TestLruCapacityOne(t *testing.T) {
	t.Log("🔍 测试: 边界条件 - 容量为1")
	lru := NewLru[string, int](1)
	t.Log("📌 创建容量为1的LRU缓存")

	// 添加元素
	t.Log("➕ 添加元素: a=1")
	lru.Set("a", 1)

	t.Log("🔄 添加a后的缓存状态:")
	printCacheStatus(t, lru)

	// 验证元素是否被添加
	if v, ok := lru.Get("a"); !ok || v != 1 {
		t.Errorf("❌ 添加元素'a'失败: 期望 1, 实际 %v, 存在状态 %v", v, ok)
	} else {
		t.Log("✅ 元素'a'成功添加到容量为1的缓存")
	}

	// 添加另一个元素
	t.Log("➕ 添加第二个元素: b=2")
	lru.Set("b", 2)

	t.Log("🔄 添加b后的缓存状态:")
	printCacheStatus(t, lru)

	// 验证第一个元素是否被淘汰
	if _, ok := lru.Get("a"); ok {
		t.Error("❌ 元素'a'应该被淘汰，但仍能获取到")
	} else {
		t.Log("✅ 元素'a'已被正确淘汰")
	}

	// 验证第二个元素是否被添加
	if v, ok := lru.Get("b"); !ok || v != 2 {
		t.Errorf("❌ 添加元素'b'失败: 期望 2, 实际 %v, 存在状态 %v", v, ok)
	} else {
		t.Log("✅ 元素'b'成功添加并替换了'a'")
	}
}

// 测试并发安全性
func TestLruConcurrency(t *testing.T) {
	t.Log("🔍 测试: 并发安全性")
	lru := NewLru[int, int](1000)
	t.Log("📌 创建容量为1000的LRU缓存")
	var wg sync.WaitGroup
	concurrent := 10

	// 并发写入
	t.Log("⚡ 启动并发写入 (10个协程，每个100个键)")
	for i := 0; i < concurrent; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				key := id*100 + j
				lru.Set(key, key)
			}
		}(i)
	}

	// 并发读取和写入
	t.Log("⚡ 启动并发读取和写入 (10个协程，每个100个键)")
	for i := 0; i < concurrent; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				key := id*100 + j
				lru.Get(key)
				lru.Put(key, key*2)
				lru.Take(key)
			}
		}(i)
	}

	// 并发删除
	t.Log("⚡ 启动并发删除 (10个协程，每个删除50个键)")
	for i := 0; i < concurrent; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 50; j++ { // 只删除一半的键
				key := id*100 + j
				lru.Del(key)
			}
		}(i)
	}

	wg.Wait()
	t.Log("✅ 所有并发操作完成")

	// 验证预期的大致元素数量（由于并发，实际数量可能有所不同）
	count := lru.Len()
	t.Logf("🔢 最终缓存大小: %d", count)
	if count > 1000 {
		t.Errorf("❌ 缓存超出容量限制: %d > 1000", count)
	} else {
		t.Log("✅ 缓存大小在容量限制内")
	}
}

// 测试不同类型
func TestLruDifferentTypes(t *testing.T) {
	t.Log("🔍 测试: 支持不同数据类型")

	// string -> int
	t.Log("📌 测试 string 键 -> int 值")
	lru1 := NewLru[string, int](3)
	lru1.Set("a", 1)
	if v, ok := lru1.Get("a"); !ok || v != 1 {
		t.Error("❌ string->int 类型测试失败")
	} else {
		t.Log("✅ string->int 类型测试通过")
	}

	// int -> string
	t.Log("📌 测试 int 键 -> string 值")
	lru2 := NewLru[int, string](3)
	lru2.Set(1, "a")
	if v, ok := lru2.Get(1); !ok || v != "a" {
		t.Error("❌ int->string 类型测试失败")
	} else {
		t.Log("✅ int->string 类型测试通过")
	}

	// 自定义结构体类型
	t.Log("📌 测试自定义结构体类型")
	type person struct {
		name string
		age  int
	}

	lru3 := NewLru[string, person](3)
	alice := person{name: "Alice", age: 30}
	lru3.Set("alice", alice)
	if v, ok := lru3.Get("alice"); !ok || v.name != "Alice" || v.age != 30 {
		t.Errorf("❌ 自定义结构体类型测试失败: 期望 %+v, 实际 %+v", alice, v)
	} else {
		t.Log("✅ 自定义结构体类型测试通过")
	}
}

// 基准测试 - Set操作
func BenchmarkLruSet(b *testing.B) {
	lru := NewLru[int, int](b.N)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lru.Set(i, i)
	}
}

// 基准测试 - Get操作（缓存命中）
func BenchmarkLruGetHit(b *testing.B) {
	lru := NewLru[int, int](b.N)
	for i := 0; i < b.N; i++ {
		lru.Set(i, i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lru.Get(i)
	}
}

// 基准测试 - Get操作（缓存未命中）
func BenchmarkLruGetMiss(b *testing.B) {
	lru := NewLru[int, int](b.N)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lru.Get(i)
	}
}

// 演示完整用例的示例
func ExampleLru() {
	// 创建一个容量为3的LRU缓存
	cache := NewLru[string, int](3)

	// 添加元素
	cache.Set("one", 1)
	cache.Set("two", 2)
	cache.Set("three", 3)

	// 再次访问"one"使其成为最近使用的元素
	if val, ok := cache.Get("one"); ok {
		fmt.Println("获取到:", val)
	}

	// 添加第四个元素，"two"将被淘汰（最近最少使用）
	cache.Set("four", 4)

	// 检查"two"是否被淘汰
	if _, ok := cache.Get("two"); !ok {
		fmt.Println("键 'two' 已被淘汰")
	}

	// 查看缓存中的所有元素
	fmt.Println("缓存内容:")
	cache.Range(func(k, v any) bool {
		fmt.Printf("%v: %v\n", k, v)
		return true
	})

	// Output:
	// 获取到: 1
	// 键 'two' 已被淘汰
	// 缓存内容:
	// four: 4
	// one: 1
	// three: 3
}

// 帮助函数：打印缓存当前状态
func printCacheStatus[K comparable, V any](t *testing.T, lru *Lru[K, V]) {
	var count int
	lru.Range(func(k, v any) bool {
		count++
		t.Logf("   %d. 键: %v, 值: %v", count, k, v)
		return true
	})
	if count == 0 {
		t.Log("   (缓存为空)")
	}
}
