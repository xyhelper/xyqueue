package xyqueue

import (
    "math/rand"
    "testing"
)

func BenchmarkEnqueue(b *testing.B) {
    q := New[int](false)
    b.ReportAllocs()
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        q.Enqueue(i)
    }
}

func BenchmarkEnqueueDequeue(b *testing.B) {
    q := New[int](false)
    b.ReportAllocs()
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        q.Enqueue(i)
        if i%2 == 1 { // keep size bounded
            q.Dequeue()
        }
    }
}

func BenchmarkEnqueue_DedupHits(b *testing.B) {
    q := New[int](true)
    // Preload with a small range to force many duplicate hits.
    for i := 0; i < 1024; i++ {
        q.Enqueue(i)
    }
    rnd := rand.New(rand.NewSource(1))
    b.ReportAllocs()
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        q.Enqueue(rnd.Intn(1024)) // mostly ignored due to dedup
    }
}

func BenchmarkContains_Dedup(b *testing.B) {
    q := New[int](true)
    for i := 0; i < 100_000; i++ {
        q.Enqueue(i)
    }
    b.ReportAllocs()
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = q.Contains(i % 100_000)
    }
}

func BenchmarkContains_NonDedup(b *testing.B) {
    q := New[int](false)
    for i := 0; i < 50_000; i++ {
        q.Enqueue(i)
    }
    b.ReportAllocs()
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = q.Contains(i % 50_000)
    }
}

