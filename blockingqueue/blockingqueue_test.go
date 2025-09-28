package blockingqueue

import (
    "context"
    "runtime"
    "sync"
    "testing"
    "time"
)

func TestTakeBlocksAndWakes(t *testing.T) {
    bq := New[string](true)
    done := make(chan struct{})
    go func() {
        defer close(done)
        ctx, cancel := context.WithTimeout(context.Background(), time.Second)
        defer cancel()
        v, err := bq.Take(ctx)
        if err != nil || v != "x" {
            t.Errorf("take got (%q,%v)", v, err)
        }
    }()
    time.Sleep(10 * time.Millisecond)
    if !bq.Put("x") {
        t.Fatal("expected put to add element")
    }
    <-done
}

func TestTakeContextCancel(t *testing.T) {
    bq := New[int](false)
    ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
    defer cancel()
    _, err := bq.Take(ctx)
    if err == nil {
        t.Fatal("expected cancellation error")
    }
}

func TestPutManyWakes(t *testing.T) {
    bq := New[int](false)
    var wg sync.WaitGroup
    got := make(chan int, 3)
    wg.Add(1)
    go func() {
        defer wg.Done()
        ctx, cancel := context.WithTimeout(context.Background(), time.Second)
        defer cancel()
        for i := 0; i < 3; i++ {
            v, err := bq.Take(ctx)
            if err != nil {
                t.Errorf("unexpected err: %v", err)
                return
            }
            got <- v
        }
    }()
    time.Sleep(5 * time.Millisecond)
    n := bq.PutMany(1, 2, 3)
    if n != 3 { t.Fatalf("putmany=%d want 3", n) }
    wg.Wait()
    close(got)
    sum := 0
    for v := range got { sum += v }
    if sum != 6 { t.Fatalf("sum=%d want 6", sum) }
}

func TestHighConcurrency(t *testing.T) {
    bq := New[int](true)
    workers := runtime.GOMAXPROCS(0) * 2
    total := 500
    var wg sync.WaitGroup
    // Consumers
    for i := 0; i < workers; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for {
                ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
                v, err := bq.Take(ctx)
                cancel()
                if err != nil { return }
                _ = v
            }
        }()
    }
    // Producers
    for i := 0; i < total; i++ {
        bq.Put(i)
    }
    // Drain with deadline
    time.Sleep(50 * time.Millisecond)
    wg.Wait()
}

