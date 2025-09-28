package blockingqueue

import (
    "context"
    "fmt"
    "time"
)

func Example_basic() {
    bq := New[string](true)
    go func() {
        // Producer
        _ = bq.Put("a")
        _ = bq.Put("a") // ignored (dedup)
        _ = bq.Put("b")
    }()

    // Consumer with timeout safety
    ctx, cancel := context.WithTimeout(context.Background(), time.Second)
    defer cancel()
    v1, _ := bq.Take(ctx)
    v2, _ := bq.Take(ctx)
    fmt.Println(v1, v2)
    // Output:
    // a b
}

func Example_errorHandling() {
    bq := New[int](false)

    // Context timeout leads to an error from Take.
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
    defer cancel()
    _, err := bq.Take(ctx)
    fmt.Println(IsContextError(err))
    fmt.Println(err == ErrDeadlineExceeded || err == ErrCanceled)

    // Put returns whether the value was actually added (dedup awareness).
    bq = New[int](true)
    fmt.Println(bq.Put(1)) // true
    fmt.Println(bq.Put(1)) // false (ignored by dedup)

    // TryTake is non-blocking and reports via ok.
    if v, ok := bq.TryTake(); ok {
        fmt.Println(v, ok) // first value present
    } else {
        fmt.Println("empty", ok)
    }
    if v, ok := bq.TryTake(); ok {
        fmt.Println(v, ok)
    } else {
        fmt.Println("empty", ok) // now empty
    }
    // Output:
    // true
    // true
    // true
    // false
    // 1 true
    // empty false
}
