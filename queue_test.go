package xyqueue

import (
	"runtime"
	"sort"
	"sync"
	"testing"
)

func TestFIFO(t *testing.T) {
	q := New[int](false)
	if !q.IsEmpty() {
		t.Fatal("new queue should be empty")
	}
	q.Enqueue(1)
	q.Enqueue(2)
	q.Enqueue(3)

	if q.Len() != 3 {
		t.Fatalf("len = %d want 3", q.Len())
	}
	if v, ok := q.Peek(); !ok || v != 1 {
		t.Fatalf("peek = %v,%v want 1,true", v, ok)
	}
	for i := 1; i <= 3; i++ {
		v, ok := q.Dequeue()
		if !ok || v != i {
			t.Fatalf("dequeue = %v,%v want %d,true", v, ok, i)
		}
	}
	if _, ok := q.Dequeue(); ok {
		t.Fatal("expected empty after dequeues")
	}
}

func TestDedup(t *testing.T) {
	q := New[string](true)
	added := q.EnqueueMany("a", "b", "a", "c", "b")
	if added != 3 {
		t.Fatalf("added = %d want 3", added)
	}
	if !q.Contains("a") || !q.Contains("b") || !q.Contains("c") {
		t.Fatal("expected all unique elements present")
	}
	got := []string{}
	for !q.IsEmpty() {
		v, _ := q.Dequeue()
		got = append(got, v)
	}
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("len(got)=%d want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order mismatch at %d: got %q want %q", i, got[i], want[i])
		}
	}
	// After removal, we can enqueue again
	if !q.Enqueue("a") {
		t.Fatal("expected enqueue after dequeue to succeed")
	}
}

func TestRemoveAndContains(t *testing.T) {
	q := New[int](true)
	q.EnqueueMany(10, 20, 30)
	if !q.Contains(20) {
		t.Fatal("expected contains 20")
	}
	if !q.Remove(20) {
		t.Fatal("expected remove 20 true")
	}
	if q.Contains(20) {
		t.Fatal("expected 20 removed")
	}
	// Remaining order 10,30
	v, _ := q.Dequeue()
	if v != 10 {
		t.Fatalf("want 10 got %d", v)
	}
	v, _ = q.Dequeue()
	if v != 30 {
		t.Fatalf("want 30 got %d", v)
	}
}

func TestConcurrentDedup(t *testing.T) {
	q := New[int](true)
	values := []int{}
	for i := 0; i < 100; i++ {
		values = append(values, i)
	}
	// Enqueue the same set from multiple goroutines.
	var wg sync.WaitGroup
	workers := runtime.GOMAXPROCS(0)
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, v := range values {
				q.Enqueue(v)
			}
		}()
	}
	wg.Wait()

	// Dequeue all and check uniqueness
	got := q.ToSlice()
	sort.Ints(got)
	if len(got) != len(values) {
		t.Fatalf("len=%d want %d (unique)", len(got), len(values))
	}
	for i := 0; i < len(values); i++ {
		if got[i] != i {
			t.Fatalf("missing or duplicate value: got[%d]=%d", i, got[i])
		}
	}
}
