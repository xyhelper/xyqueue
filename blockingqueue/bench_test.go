package blockingqueue

import (
    "context"
    "testing"
    "time"
)

// Benchmark pairs of Put/Take with a single consumer.
func BenchmarkPutTake(b *testing.B) {
    bq := New[int](false)
    ctx := context.Background()
    done := make(chan struct{})
    // Consumer
    go func() {
        for i := 0; i < b.N; i++ {
            _, _ = bq.Take(ctx)
        }
        close(done)
    }()
    b.ReportAllocs()
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        bq.Put(i)
    }
    <-done
}

// Benchmark TryTake in a polling-like scenario.
func BenchmarkTryTake(b *testing.B) {
    bq := New[int](false)
    // Pre-fill
    for i := 0; i < b.N; i++ { bq.Put(i) }
    b.ReportAllocs()
    b.ResetTimer()
    taken := 0
    for taken < b.N {
        if _, ok := bq.TryTake(); ok {
            taken++
        } else {
            time.Sleep(time.Microsecond)
        }
    }
}

