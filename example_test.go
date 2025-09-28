package xyqueue

import (
    "context"
    "fmt"
    "sync"
    "time"
)

// Example showing basic FIFO without de-duplication.
func Example_basic() {
    q := New[int](false)
    q.Enqueue(1)
    q.Enqueue(2)
    q.Enqueue(3)
    for !q.IsEmpty() {
        v, _ := q.Dequeue()
        fmt.Println(v)
    }
    // Output:
    // 1
    // 2
    // 3
}

// Example showing FIFO with de-duplication enabled.
func Example_dedup() {
    q := New[string](true)
    q.Enqueue("a")
    q.Enqueue("a") // ignored
    q.Enqueue("b")
    for !q.IsEmpty() {
        v, _ := q.Dequeue()
        fmt.Println(v)
    }
    // Output:
    // a
    // b
}

// Example for EnqueueMany and ToSlice with de-duplication.
func Example_enqueueMany() {
    q := New[int](true)
    n := q.EnqueueMany(1, 1, 2, 3, 3)
    fmt.Println(n)
    fmt.Println(q.ToSlice())
    // Output:
    // 3
    // [1 2 3]
}

// Example for Peek.
func Example_peek() {
    q := New[string](false)
    q.Enqueue("x")
    q.Enqueue("y")
    v, _ := q.Peek()
    fmt.Println(v)
    // Output:
    // x
}

// Example for Contains and Remove.
func Example_contains_remove() {
    q := New[int](true)
    q.EnqueueMany(10, 20, 30)
    fmt.Println(q.Contains(20))
    fmt.Println(q.Remove(20))
    fmt.Println(q.Contains(20))
    // Output:
    // true
    // true
    // false
}

// Example for Clear and Len/IsEmpty.
func Example_clear_toSlice() {
    q := New[int](false)
    q.EnqueueMany(1, 2)
    fmt.Println(q.ToSlice())
    q.Clear()
    fmt.Println(q.Len(), q.IsEmpty())
    // Output:
    // [1 2]
    // 0 true
}

// Example for NewWithCapacity.
func Example_newWithCapacity() {
    q := NewWithCapacity[string](true, 128)
    q.Enqueue("a")
    q.Enqueue("a") // ignored in dedup mode
    q.Enqueue("b")
    fmt.Println(q.ToSlice())
    // Output:
    // [a b]
}

// Example showing that after Dequeue, a previously present value can be
// added again when de-duplication is enabled.
func Example_dedup_readd_after_dequeue() {
    q := New[string](true)
    q.Enqueue("x")
    q.Dequeue() // remove "x"
    ok := q.Enqueue("x")
    fmt.Println(ok)
    // Output:
    // true
}

// Example demonstrating a minimal blocking wrapper using sync.Cond and context.
// This is an optional pattern layered on top of the non-blocking Queue.
func Example_blockingWrapper_cond() {
    type BQ struct {
        mu sync.Mutex
        cv *sync.Cond
        q  *Queue[string]
    }
    newBQ := func(dedup bool) *BQ {
        b := &BQ{q: New[string](dedup)}
        b.cv = sync.NewCond(&b.mu)
        return b
    }
    put := func(b *BQ, v string) {
        b.mu.Lock()
        added := b.q.Enqueue(v)
        b.mu.Unlock()
        if added {
            b.cv.Broadcast()
        }
    }
    takeCtx := func(b *BQ, ctx context.Context) (string, bool) {
        b.mu.Lock()
        defer b.mu.Unlock()
        for b.q.Len() == 0 {
            done := make(chan struct{})
            go func() { b.cv.Wait(); close(done) }()
            select {
            case <-ctx.Done():
                return "", false
            case <-done:
            }
        }
        v, ok := b.q.Dequeue()
        return v, ok
    }

    bq := newBQ(true)
    ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
    defer cancel()
    go func() {
        time.Sleep(10 * time.Millisecond)
        put(bq, "hello")
    }()
    v, ok := takeCtx(bq, ctx)
    fmt.Println(v, ok)
    // Output:
    // hello true
}

// Example using a comparable struct type.
func Example_structType() {
    type user struct {
        ID   int
        Name string
    }
    q := New[user](true)
    q.Enqueue(user{ID: 1, Name: "a"})
    q.Enqueue(user{ID: 1, Name: "a"}) // ignored (duplicate)
    q.Enqueue(user{ID: 2, Name: "b"})
    fmt.Println(len(q.ToSlice()))
    // Output:
    // 2
}
