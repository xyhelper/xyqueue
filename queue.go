package xyqueue

import (
	"sync"
)

// Queue is a generic, concurrency-safe FIFO queue with optional de-duplication.
// When de-duplication is enabled, Enqueue ignores values already present in the
// queue. After a value is removed (via Dequeue/Remove), it can be enqueued
// again. The zero value is not ready for use; construct via New or
// NewWithCapacity.
type Queue[T comparable] struct {
	mu    sync.Mutex
	data  []T
	set   map[T]struct{} // only used when dedup is true
	dedup bool
}

// New creates a new queue.
//
// When dedup is true, repeated Enqueue of the same value while it is present in
// the queue is ignored. All exported methods are safe for concurrent use.
func New[T comparable](dedup bool) *Queue[T] {
	q := &Queue[T]{
		data:  make([]T, 0),
		dedup: dedup,
	}
	if dedup {
		q.set = make(map[T]struct{})
	}
	return q
}

// NewWithCapacity creates a new queue with the given initial capacity.
// Capacity preallocates internal storage; behavior is otherwise identical to
// New. When dedup is true, the presence set is also allocated.
func NewWithCapacity[T comparable](dedup bool, capacity int) *Queue[T] {
	if capacity < 0 {
		capacity = 0
	}
	q := &Queue[T]{
		data:  make([]T, 0, capacity),
		dedup: dedup,
	}
	if dedup {
		q.set = make(map[T]struct{}, capacity)
	}
	return q
}

// Enqueue appends v to the tail.
//
// Returns true if the value was added, or false when de-duplication is enabled
// and v is already present. Amortized complexity: O(1).
func (q *Queue[T]) Enqueue(v T) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.dedup {
		if _, exists := q.set[v]; exists {
			return false
		}
		q.set[v] = struct{}{}
	}
	q.data = append(q.data, v)
	return true
}

// EnqueueMany enqueues items and returns the count actually added.
//
// When de-duplication is enabled, values already present are skipped and order
// of first occurrences is preserved. Amortized complexity: O(k) for k items.
func (q *Queue[T]) EnqueueMany(items ...T) int {
	added := 0
	q.mu.Lock()
	defer q.mu.Unlock()
	for _, v := range items {
		if q.dedup {
			if _, exists := q.set[v]; exists {
				continue
			}
			q.set[v] = struct{}{}
		}
		q.data = append(q.data, v)
		added++
	}
	return added
}

// Dequeue removes and returns the head value.
//
// The second result is false when the queue is empty. Amortized complexity: O(1).
func (q *Queue[T]) Dequeue() (T, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	var zero T
	if len(q.data) == 0 {
		return zero, false
	}
	v := q.data[0]
	// Avoid O(n) element moves by reslicing; let GC reclaim older head when needed.
	q.data = q.data[1:]
	if q.dedup {
		delete(q.set, v)
	}
	return v, true
}

// Peek returns the head value without removing it.
// The second result is false when the queue is empty. Complexity: O(1).
func (q *Queue[T]) Peek() (T, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	var zero T
	if len(q.data) == 0 {
		return zero, false
	}
	return q.data[0], true
}

// Len returns the number of elements currently queued.
// Complexity: O(1). Safe for concurrent use.
func (q *Queue[T]) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.data)
}

// IsEmpty reports whether the queue is empty.
// Complexity: O(1). Equivalent to Len() == 0.
func (q *Queue[T]) IsEmpty() bool {
	return q.Len() == 0
}

// Contains reports whether v is currently present in the queue.
// Complexity: O(1) when de-duplication is enabled; otherwise O(n).
func (q *Queue[T]) Contains(v T) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.dedup {
		_, ok := q.set[v]
		return ok
	}
	for _, x := range q.data {
		if x == v {
			return true
		}
	}
	return false
}

// Remove deletes the first occurrence of v from the queue if present.
// Returns true if removed. Complexity: O(n).
func (q *Queue[T]) Remove(v T) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	for i, x := range q.data {
		if x == v {
			// remove q.data[i]
			copy(q.data[i:], q.data[i+1:])
			q.data = q.data[:len(q.data)-1]
			if q.dedup {
				delete(q.set, v)
			}
			return true
		}
	}
	return false
}

// Clear removes all elements from the queue.
// Complexity: O(1) for the data slice; clearing the presence set (when
// de-duplication is enabled) is O(n) in the number of elements.
func (q *Queue[T]) Clear() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.data = q.data[:0]
	if q.dedup {
		clear(q.set)
	}
}

// ToSlice returns a copy of the queue's contents in FIFO order.
// Complexity: O(n). The returned slice is independent of the queue.
func (q *Queue[T]) ToSlice() []T {
	q.mu.Lock()
	defer q.mu.Unlock()
	out := make([]T, len(q.data))
	copy(out, q.data)
	return out
}
