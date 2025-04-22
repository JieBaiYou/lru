# LRU 缓存库

[![Go Reference](https://pkg.go.dev/badge/github.com/JieBaiYou/lru.svg)](https://pkg.go.dev/github.com/JieBaiYou/lru)
[![Go Report Card](https://goreportcard.com/badge/github.com/JieBaiYou/lru)](https://goreportcard.com/report/github.com/JieBaiYou/lru)
[![Coverage Status](https://coveralls.io/repos/github/JieBaiYou/lru/badge.svg?branch=master)](https://coveralls.io/github/JieBaiYou/lru?branch=master)

一个高性能、线程安全的 LRU（最近最少使用）缓存实现，基于 Go 1.18+ 泛型。支持任意类型的键值对存储，并提供丰富的缓存操作方法。

## 特性

- 基于 Go 1.18+ 泛型，支持任意类型的键值对
- 完全线程安全，支持并发读写操作
- 高性能实现，读/写/删除操作时间复杂度为 O(1)
- 丰富的 API，支持多种访问和修改方式
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
    "github.com/JieBaiYou/lru"
)

func main() {
    // 创建容量为3的缓存
    cache := lru.NewLru[string, int](3)

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
    cache.Range(func(k, v any) bool {
        fmt.Printf("%v: %v\n", k, v)
        return true
    })
}
```

## API 文档

### 创建缓存

```go
// 创建一个指定大小的 LRU 缓存
cache := lru.NewLru[KeyType, ValueType](size)
```

### 基本操作

#### Set 方法

```go
// 向缓存中添加或更新元素，并将其移动到最近使用位置
cache.Set(key, value)
```

#### Get 方法

```go
// 从缓存中获取元素，并将其移动到最近使用位置
// 返回值和是否存在的标志
value, ok := cache.Get(key)
```

#### Put 方法

```go
// 向缓存中添加或更新元素，但不改变其位置
cache.Put(key, value)
```

#### Take 方法

```go
// 从缓存中获取元素，但不改变其位置
// 返回值和是否存在的标志
value, ok := cache.Take(key)
```

#### Del 方法

```go
// 从缓存中删除元素，返回是否删除成功
deleted := cache.Del(key)
```

#### Len 方法

```go
// 获取缓存中的元素数量
count := cache.Len()
```

#### Range 方法

```go
// 遍历缓存中的所有元素
// 如果回调函数返回 false，停止遍历
cache.Range(func(k, v any) bool {
    // 处理键值对
    // 返回 true 继续遍历，返回 false 停止遍历
    return true
})
```

#### Flush 方法

```go
// 清空缓存中的所有元素
cache.Flush()
```

## 性能

基准测试结果（在 Intel(R) Xeon(R) Gold 6133 CPU @ 2.50GHz 上运行）：

```
BenchmarkLruSet-16           2781999   440.1 ns/op    118 B/op    2 allocs/op
BenchmarkLruGetHit-16        5621221   252.1 ns/op      0 B/op    0 allocs/op
BenchmarkLruGetMiss-16      30678438    39.0 ns/op      0 B/op    0 allocs/op
```

## 使用示例

### 基本缓存操作

```go
// 创建缓存
cache := lru.NewLru[string, int](100)

// 添加元素
cache.Set("key1", 100)
cache.Set("key2", 200)

// 获取元素
if val, ok := cache.Get("key1"); ok {
    // 使用 val
}

// 删除元素
cache.Del("key2")

// 获取但不更新元素位置
if val, ok := cache.Take("key1"); ok {
    // 使用 val，但不会影响该元素在 LRU 中的位置
}

// 更新元素但不改变其位置
cache.Put("key1", 150)

// 获取元素数量
count := cache.Len()

// 清空缓存
cache.Flush()
```

### 使用自定义类型

```go
type User struct {
    ID   int
    Name string
    Age  int
}

// 创建存储自定义类型的缓存
cache := lru.NewLru[int, User](100)

cache.Set(1, User{ID: 1, Name: "Alice", Age: 30})
cache.Set(2, User{ID: 2, Name: "Bob", Age: 25})

if user, ok := cache.Get(1); ok {
    fmt.Printf("用户: %s, 年龄: %d\n", user.Name, user.Age)
}
```

### 遍历缓存

```go
cache := lru.NewLru[string, int](10)
// 添加一些元素...

// 遍历所有元素
cache.Range(func(k, v any) bool {
    key := k.(string)
    value := v.(int)
    fmt.Printf("%s: %d\n", key, value)
    return true
})

// 提前终止遍历
count := 0
cache.Range(func(k, v any) bool {
    count++
    return count < 5 // 只遍历前5个元素
})
```

## 注意事项

- 如果设置的缓存容量为 0，任何添加操作都会被拒绝
- 当缓存已满时，添加新元素会自动淘汰最久未使用的元素
- 所有操作都是线程安全的，可以在并发环境中安全使用

## 贡献

欢迎提交问题和功能请求，也欢迎提交 Pull Requests。

## 许可证

MIT 许可证
