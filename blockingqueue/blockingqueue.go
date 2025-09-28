package blockingqueue

import (
    "context"
    "errors"
    "sync"

    base "github.com/xyhelper/xyqueue"
)

// Queue is a blocking, concurrency-safe FIFO built on xyqueue with optional
// de-duplication. When de-duplication is enabled, Put skips values already
// present; after removal the value can be added again.
//
// All methods are safe for concurrent use by multiple goroutines.
type Queue[T comparable] struct {
    mu sync.Mutex
    cv *sync.Cond
    q  *base.Queue[T]
}

// New creates a new blocking queue.
func New[T comparable](dedup bool) *Queue[T] {
    b := &Queue[T]{q: base.New[T](dedup)}
    b.cv = sync.NewCond(&b.mu)
    return b
}

// NewWithCapacity creates a new blocking queue with initial capacity.
func NewWithCapacity[T comparable](dedup bool, capacity int) *Queue[T] {
    b := &Queue[T]{q: base.NewWithCapacity[T](dedup, capacity)}
    b.cv = sync.NewCond(&b.mu)
    return b
}

// Put appends v to the tail. Returns true if the value was added, or false
// when de-duplication is enabled and v is already present. Wakes waiters only
// when an element is actually added.
func (b *Queue[T]) Put(v T) bool {
    b.mu.Lock()
    added := b.q.Enqueue(v)
    if added {
        b.cv.Broadcast()
    }
    b.mu.Unlock()
    return added
}

// PutMany enqueues items and returns the count actually added.
// Broadcasts once if any element is added.
func (b *Queue[T]) PutMany(items ...T) int {
    b.mu.Lock()
    n := b.q.EnqueueMany(items...)
    if n > 0 {
        b.cv.Broadcast()
    }
    b.mu.Unlock()
    return n
}

// TryTake removes and returns the head value without blocking.
// ok is false if the queue is empty.
func (b *Queue[T]) TryTake() (v T, ok bool) {
    b.mu.Lock()
    v, ok = b.q.Dequeue()
    b.mu.Unlock()
    return
}

// Take blocks until an element is available or ctx is done. On success returns
// (value, nil). On cancellation returns the zero value and ctx.Err().
func (b *Queue[T]) Take(ctx context.Context) (T, error) {
    if ctx == nil {
        ctx = context.Background()
    }
    b.mu.Lock()
    // Fast path
    if v, ok := b.q.Dequeue(); ok {
        b.mu.Unlock()
        return v, nil
    }
    // Wait with context cancellation. We spawn a short-lived watcher that
    // broadcasts on cancellation to wake Wait.
    for {
        done := make(chan struct{})
        go func() {
            select {
            case <-ctx.Done():
                b.mu.Lock()
                b.cv.Broadcast()
                b.mu.Unlock()
            case <-done:
            }
        }()

        b.cv.Wait() // releases and re-acquires b.mu
        close(done)

        if v, ok := b.q.Dequeue(); ok {
            b.mu.Unlock()
            return v, nil
        }
        if err := ctx.Err(); err != nil {
            b.mu.Unlock()
            var zero T
            return zero, err
        }
    }
}

// Peek returns the head value without removing it. ok is false when empty.
func (b *Queue[T]) Peek() (v T, ok bool) {
    b.mu.Lock()
    v, ok = b.q.Peek()
    b.mu.Unlock()
    return
}

// Len returns the number of elements currently queued.
func (b *Queue[T]) Len() int {
    b.mu.Lock()
    n := b.q.Len()
    b.mu.Unlock()
    return n
}

// IsEmpty reports whether the queue is empty.
func (b *Queue[T]) IsEmpty() bool { return b.Len() == 0 }

// Contains reports whether v is currently present in the queue.
func (b *Queue[T]) Contains(v T) bool {
    b.mu.Lock()
    ok := b.q.Contains(v)
    b.mu.Unlock()
    return ok
}

// Remove deletes the first occurrence of v from the queue if present.
// Returns true if removed.
func (b *Queue[T]) Remove(v T) bool {
    b.mu.Lock()
    removed := b.q.Remove(v)
    b.mu.Unlock()
    return removed
}

// Clear removes all elements from the queue.
func (b *Queue[T]) Clear() {
    b.mu.Lock()
    b.q.Clear()
    b.mu.Unlock()
}

// ErrCanceled is returned by Take when the context is canceled.
var ErrCanceled = context.Canceled

// ErrDeadlineExceeded is returned by Take when the context deadline expires.
var ErrDeadlineExceeded = context.DeadlineExceeded

// IsContextError reports whether err equals context.Canceled or context.DeadlineExceeded.
func IsContextError(err error) bool {
    return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

