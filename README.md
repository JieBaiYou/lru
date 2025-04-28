# Go LRU 缓存

[![Go Reference](https://pkg.go.dev/badge/github.com/JieBaiYou/lru.svg)](https://pkg.go.dev/github.com/JieBaiYou/lru)
[![Go Report Card](https://goreportcard.com/badge/github.com/JieBaiYou/lru)](https://goreportcard.com/report/github.com/JieBaiYou/lru)
[![Coverage Status](https://coveralls.io/repos/github/JieBaiYou/lru/badge.svg?branch=master)](https://coveralls.io/github/JieBaiYou/lru?branch=master)

一个高性能、线程安全的 LRU（最近最少使用）缓存实现，基于 Go 1.18+ 泛型。支持任意类型的键值对存储，提供过期时间设置和自动清理功能。

## 特性

- 基于 Go 1.18+ 泛型，支持任意类型的键值对
- 完全线程安全，支持并发读写操作
- 支持为缓存项设置过期时间（TTL）
- "零配置"的自动后台清理过期项
- 高性能实现，读/写/删除操作时间复杂度为 O(1)
- 灵活的 API 设计，支持链式调用
- 100% 测试覆盖率，稳定可靠
- 零外部依赖，仅使用标准库

## 安装

```bash
go get github.com/JieBaiYou/lru
```

要求 Go 1.18 或更高版本（支持泛型）。

## 快速开始

以下是基本用法示例：

```go
package main

import (
    "fmt"
    "time"
    "github.com/JieBaiYou/lru"
)

func main() {
    // 创建容量为3的缓存
    cache := lru.New[string, int](3)

    // 添加元素
    cache.Set("one", 1)
    cache.Set("two", 2)
    cache.Set("three", 3)

    // 读取元素
    if val, ok := cache.Get("one"); ok {
        fmt.Println("值:", val) // 输出: 值: 1
    }

    // 添加第四个元素（会淘汰最久未使用的元素）
    cache.Set("four", 4)

    // "two" 应该已被淘汰
    if _, ok := cache.Get("two"); !ok {
        fmt.Println("元素 'two' 已被淘汰")
    }

    // 遍历所有元素
    fmt.Println("缓存内容:")
    cache.Range(func(k string, v int) bool {
        fmt.Printf("%v: %v\n", k, v)
        return true
    })
}
```

## 核心 API

### 创建缓存

```go
// 创建指定大小的缓存
cache := lru.New[KeyType, ValueType](size)

// 创建并配置默认过期时间
cache := lru.New[string, int](100).TTL(time.Minute)

// 创建并开启后台清理
cache := lru.New[string, int](100).TTL(time.Minute).Cleaner(time.Minute)
```

### 基本操作

```go
// 添加或更新缓存项
cache.Set(key, value)

// 添加并设置过期时间
cache.Set(key, value).Expire(5 * time.Minute)

// 获取缓存项
value, exists := cache.Get(key)

// 获取但不更新LRU顺序
value, exists := cache.Peek(key)

// 删除缓存项
deleted := cache.Delete(key)

// 清空整个缓存
cache.Clear()

// 获取缓存大小和容量
size := cache.Size()
capacity := cache.Capacity()

// 调整缓存容量
err := cache.SetCapacity(200)

// 获取所有缓存键
keys := cache.Keys()

// 遍历所有缓存项
cache.Range(func(key KeyType, value ValueType) bool {
    // 处理key/value
    // 返回false停止遍历
    return true
})
```

### 过期时间与清理

```go
// 设置全局默认过期时间
cache.TTL(time.Hour)

// 为特定项设置过期时间
cache.Set(key, value).Expire(10 * time.Minute)

// 设置为永不过期
cache.Set(key, value).Expire(0)

// 手动清理所有过期项
purged := cache.Purge()

// 启动自动清理（指定清理间隔）
cache.Cleaner(5 * time.Minute)

// 停止使用缓存时清理goroutine
cache.Close()
```

## 高级使用示例

### 带过期时间的缓存

```go
package main

import (
    "fmt"
    "time"
    "github.com/JieBaiYou/lru"
)

func main() {
    // 创建缓存并设置1分钟的默认过期时间
    cache := lru.New[string, string](100).TTL(time.Minute)

    // 使用默认过期时间添加项
    cache.Set("key1", "默认1分钟过期")

    // 使用自定义过期时间添加项
    cache.Set("key2", "5秒后过期").Expire(5 * time.Second)

    // 添加永不过期的项
    cache.Set("key3", "永不过期").Expire(0)

    fmt.Println("初始状态:")
    printCacheContents(cache)

    // 等待6秒让key2过期
    time.Sleep(6 * time.Second)

    fmt.Println("\n6秒后:")
    printCacheContents(cache)

    // 手动清理过期项
    purged := cache.Purge()
    fmt.Printf("\n手动清理了 %d 个过期项\n", purged)

    fmt.Println("\n清理后:")
    printCacheContents(cache)
}

func printCacheContents(cache *lru.Cache[string, string]) {
    cache.Range(func(k string, v string) bool {
        fmt.Printf("键: %s, 值: %s\n", k, v)
        return true
    })
}
```

### 自动清理过期项

```go
package main

import (
    "fmt"
    "time"
    "github.com/JieBaiYou/lru"
)

func main() {
    // 创建缓存，启用自动清理
    cache := lru.New[string, int](100).
        TTL(time.Minute).           // 默认过期时间1分钟
        Cleaner(5 * time.Second)    // 每5秒清理一次

    // 使用defer确保资源释放
    defer cache.Close()

    // 添加一些会很快过期的项
    cache.Set("a", 1).Expire(3 * time.Second)
    cache.Set("b", 2).Expire(4 * time.Second)
    cache.Set("c", 3).Expire(10 * time.Second)
    cache.Set("d", 4) // 使用默认过期时间

    fmt.Println("初始状态:")
    fmt.Printf("缓存大小: %d\n", cache.Size())

    // 等待6秒，让自动清理器运行
    time.Sleep(6 * time.Second)

    fmt.Println("\n6秒后 (a和b应该被自动清理):")
    fmt.Printf("缓存大小: %d\n", cache.Size())
    printKeys(cache)

    // 再等待5秒
    time.Sleep(5 * time.Second)

    fmt.Println("\n11秒后 (c应该被自动清理):")
    fmt.Printf("缓存大小: %d\n", cache.Size())
    printKeys(cache)
}

func printKeys(cache *lru.Cache[string, int]) {
    keys := cache.Keys()
    fmt.Printf("当前键: %v\n", keys)
}
```

### 泛型类型示例

```go
package main

import (
    "fmt"
    "github.com/JieBaiYou/lru"
)

// 自定义键类型
type UserID int

// 自定义值类型
type UserProfile struct {
    Name    string
    Age     int
    IsAdmin bool
}

func main() {
    // 创建使用自定义类型的缓存
    cache := lru.New[UserID, UserProfile](100)

    // 添加用户数据
    cache.Set(1001, UserProfile{Name: "张三", Age: 30, IsAdmin: false})
    cache.Set(1002, UserProfile{Name: "李四", Age: 25, IsAdmin: true})
    cache.Set(1003, UserProfile{Name: "王五", Age: 35, IsAdmin: false})

    // 获取并使用用户数据
    if profile, ok := cache.Get(1002); ok {
        fmt.Printf("用户: %s, 年龄: %d, 管理员: %t\n",
            profile.Name, profile.Age, profile.IsAdmin)
    }

    // 只获取管理员用户
    adminCount := 0
    cache.Range(func(id UserID, profile UserProfile) bool {
        if profile.IsAdmin {
            adminCount++
            fmt.Printf("管理员 #%d: %s (ID: %d)\n", adminCount, profile.Name, id)
        }
        return true
    })
}
```

### 并发安全使用示例

```go
package main

import (
    "fmt"
    "sync"
    "time"
    "github.com/JieBaiYou/lru"
)

func main() {
    // 创建缓存
    cache := lru.New[string, int](1000)

    var wg sync.WaitGroup

    // 并发写入
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            for j := 0; j < 100; j++ {
                key := fmt.Sprintf("key-%d-%d", id, j)
                cache.Set(key, id*1000+j)
                // 模拟一些处理时间
                time.Sleep(time.Microsecond)
            }
        }(i)
    }

    // 并发读取
    for i := 0; i < 5; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            for j := 0; j < 200; j++ {
                // 随机访问不同写入者创建的键
                writeID := j % 10
                itemID := j % 100
                key := fmt.Sprintf("key-%d-%d", writeID, itemID)

                cache.Get(key)
                // 模拟一些处理时间
                time.Sleep(time.Microsecond)
            }
        }(i)
    }

    // 等待所有协程完成
    wg.Wait()

    // 查看最终缓存状态
    fmt.Printf("最终缓存大小: %d/%d\n", cache.Size(), cache.Capacity())
}
```

## 性能

基准测试结果（在 Intel Core i7-9750H CPU @ 2.60GHz 上运行）：

```
BenchmarkSet-12           2000000    644 ns/op    119 B/op    2 allocs/op
BenchmarkGetHit-12        5000000    236 ns/op      0 B/op    0 allocs/op
BenchmarkGetMiss-12      30000000     40 ns/op      0 B/op    0 allocs/op
```

## 最佳实践

1. **选择合适的缓存大小**：设置一个合理的缓存大小对性能至关重要。过大的缓存会占用更多内存，而过小的缓存会导致频繁淘汰。

2. **使用过期时间**：为缓存项设置过期时间可以防止数据过时。对不同类型的数据使用不同的过期策略。

3. **启用自动清理**：对大型缓存，启用自动清理可以及时释放内存。清理间隔应考虑缓存大小和过期频率。

4. **正确释放资源**：不再使用缓存时调用`Close()`停止自动清理，防止资源泄露。推荐使用`defer cache.Close()`确保资源释放。即使忘记调用 Close，缓存在被垃圾回收时也会尝试释放资源。

## 贡献

欢迎提交问题和功能请求，也欢迎提交 Pull Requests。

## 许可证

MIT 许可证
