# XYQUEUE

一个用 Go 实现的泛型队列库，遵循先进先出（FIFO），可选“去重”模式（同一元素在队列中仅保留一次），并发安全，适合在任务调度、消息缓冲、去重队列等场景中使用。

## 目录
- [特性](#特性)
- [安装](#安装)
- [快速开始](#快速开始)
- [API 概览](#api-概览)
- [线程安全与去重说明](#线程安全与去重说明)
- [复杂度简述](#复杂度简述)
- [运行测试](#运行测试)
- [更多示例](#更多示例)
- [高级用法：阻塞/超时的包装模式](#高级用法阻塞超时的包装模式)
- [基准测试](#基准测试)
- [示例：并发入队去重](#示例并发入队去重)
- [阻塞队列（blockingqueue 子包）](#阻塞队列blockingqueue-子包)
  - [错误处理示例](#错误处理示例)

## 特性
- 泛型支持：`Queue[T comparable]`，适用于任意可比较类型。
- 可选去重：开启后，已在队列中的元素不会被重复入队；元素被取出后可再次入队。
- 并发安全：内部采用 `sync.Mutex` 保护，常见操作线程安全。
- 简洁 API：`Enqueue/Dequeue/Peek/Len/Contains/Remove/Clear/ToSlice`。

## 安装
```bash
go get github.com/xyhelper/xyqueue@latest
```

要求 Go 1.18+（使用泛型）。

## 快速开始
```go
package main

import (
    "fmt"
    "github.com/xyhelper/xyqueue"
)

func main() {
    // 开启去重
    q := xyqueue.New[string](true)
    q.Enqueue("a")
    q.Enqueue("a") // 被忽略（已存在）
    q.Enqueue("b")

    if v, ok := q.Peek(); ok {
        fmt.Println("peek:", v) // peek: a
    }

    for !q.IsEmpty() {
        v, _ := q.Dequeue()
        fmt.Println(v)
    }
}
```

## API 概览
- `New(dedup bool)` / `NewWithCapacity(dedup bool, n int)`：创建队列；`dedup=true` 开启去重。
- `Enqueue(v T) bool`：入队；去重开启时，若已存在则返回 `false`。
- `EnqueueMany(items ...T) int`：批量入队；返回成功入队的数量。
- `Dequeue() (T, bool)`：出队；空队列返回 `ok=false`。
- `Peek() (T, bool)`：查看队头不移除。
- `Len() int` / `IsEmpty() bool`：长度与空判定。
- `Contains(v T) bool`：判断是否在队列中（去重模式 O(1)，否则 O(n)）。
- `Remove(v T) bool`：移除首个匹配元素（O(n)）。
- `Clear()` / `ToSlice() []T`：清空 / 复制为切片。

## 线程安全与去重说明
- 全部公开方法均加锁，适合多协程环境。可搭配 `go test -race` 检查竞态。
- 去重仅保证“队列中不出现重复元素”；当元素被 `Dequeue`/`Remove` 移除后，可再次入队。

## 复杂度简述
- `Enqueue/Dequeue/Peek/Len`：摊还 O(1)。
- `Contains`：去重模式 O(1)，否则 O(n)。
- `Remove`：O(n)。

## 运行测试
```bash
go fmt ./...
go vet ./...
go test -race -v ./...
```

## 更多示例
入队批量与去重：
```go
q := xyqueue.New[int](true)
added := q.EnqueueMany(1, 1, 2, 3, 3) // 仅 1,2,3 被加入
fmt.Println(added)       // 3
fmt.Println(q.ToSlice()) // [1 2 3]
```

查看队头（不移除）：
```go
q := xyqueue.New[string](false)
q.Enqueue("x"); q.Enqueue("y")
v, _ := q.Peek() // x
```

判断与移除元素：
```go
q := xyqueue.New[int](true)
q.EnqueueMany(10, 20, 30)
fmt.Println(q.Contains(20)) // true
fmt.Println(q.Remove(20))   // true
fmt.Println(q.Contains(20)) // false
```

清空与复制为切片：
```go
q := xyqueue.New[int](false)
q.EnqueueMany(1, 2)
fmt.Println(q.ToSlice()) // [1 2]
q.Clear()
fmt.Println(q.Len(), q.IsEmpty()) // 0 true
```

指定初始容量：
```go
q := xyqueue.NewWithCapacity[string](true, 128)
q.Enqueue("a"); q.Enqueue("a") // 被忽略
q.Enqueue("b")
fmt.Println(q.ToSlice()) // [a b]
```

### 高级用法：阻塞/超时的包装模式
基础队列为非阻塞设计。需要阻塞等待或超时取消时，可在外层用 `sync.Cond` + `context` 封装（示例简化版）：
```go
type BQ[T comparable] struct {
    mu sync.Mutex
    cv *sync.Cond
    q  *xyqueue.Queue[T]
}

func newBQ[T comparable](dedup bool) *BQ[T] {
    b := &BQ[T]{q: xyqueue.New[T](dedup)}
    b.cv = sync.NewCond(&b.mu)
    return b
}

func (b *BQ[T]) Put(v T) {
    b.mu.Lock()
    added := b.q.Enqueue(v)
    b.mu.Unlock()
    if added { b.cv.Broadcast() }
}

func (b *BQ[T]) Take(ctx context.Context) (T, bool) {
    var zero T
    b.mu.Lock()
    defer b.mu.Unlock()
    for b.q.Len() == 0 {
        done := make(chan struct{})
        go func() { b.cv.Wait(); close(done) }()
        select {
        case <-ctx.Done():
            return zero, false
        case <-done:
        }
    }
    return b.q.Dequeue()
}
```

或采用“轮询 + 超时”的简化方案：
```go
deadline := time.After(200 * time.Millisecond)
for {
    if v, ok := q.Dequeue(); ok { /* use v */ break }
    select {
    case <-deadline:
        // 超时处理
        return
    default:
        time.Sleep(1 * time.Millisecond)
    }
}
```

## 基准测试
运行基准测试（含内存分配统计）：
```bash
go test -bench=. -benchmem
```
包含的基准：
- `BenchmarkEnqueue`：纯入队吞吐。
- `BenchmarkEnqueueDequeue`：入队/出队交替，控制队列规模。
- `BenchmarkEnqueue_DedupHits`：高命中率的去重入队。
- `BenchmarkContains_Dedup` vs `BenchmarkContains_NonDedup`：`Contains` 在去重（O(1)）与非去重（O(n)）模式下的对比。

## 示例：并发入队去重
```go
var wg sync.WaitGroup
q := xyqueue.New[int](true)
for w := 0; w < 4; w++ {
    wg.Add(1)
    go func() {
        defer wg.Done()
        for i := 0; i < 100; i++ {
            q.Enqueue(i) // 去重保证唯一
        }
    }()
}
wg.Wait()
fmt.Println(q.Len()) // 100
```

## 阻塞队列（blockingqueue 子包）
当需要阻塞式消费或超时控制时，使用 `blockingqueue` 子包（基于 `sync.Cond` + `context` 封装）：

安装/导入：
```bash
go get github.com/xyhelper/xyqueue@latest
```
```go
import (
    "context"
    "time"
    bq "github.com/xyhelper/xyqueue/blockingqueue"
)

func demo() error {
    q := bq.New[string](true) // 去重 + 阻塞
    go func() { q.Put("a"); q.Put("a"); q.Put("b") }()
    ctx, cancel := context.WithTimeout(context.Background(), time.Second)
    defer cancel()
    v1, err := q.Take(ctx) // a
    if err != nil { return err }
    v2, err := q.Take(ctx) // b
    if err != nil { return err }
    _, _ = v1, v2
    return nil
}
```

关键 API：
- `New/ NewWithCapacity`：创建阻塞队列（支持去重）。
- `Put/ PutMany`：入队；仅在实际新增时唤醒等待者。
- `Take(ctx)`：阻塞取元素，ctx 取消/超时返回错误。
- `TryTake`：非阻塞取元素。
- 其余：`Peek/Len/IsEmpty/Contains/Remove/Clear`。

### 错误处理示例
```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
defer cancel()
_, err := q.Take(ctx)
// 统一判断是否为上下文错误（取消或超时）
if bq.IsContextError(err) {
    // err 可能是 bq.ErrCanceled 或 bq.ErrDeadlineExceeded
}

// 去重影响 Put 的返回值：
ok1 := q.Put("a") // true（成功入队）
ok2 := q.Put("a") // false（去重命中，未入队）

// 非阻塞获取：
if v, ok := q.TryTake(); ok {
    _ = v // got value
} else {
    // 队列为空
}
```
