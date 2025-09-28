// Package xyqueue provides a generic FIFO queue with optional de-duplication.
//
// The queue is concurrency-safe: all exported methods use internal locking and
// may be called from multiple goroutines. Construct a queue with New or
// NewWithCapacity. When de-duplication is enabled, Enqueue skips values that
// are already present; once a value is removed (via Dequeue or Remove), it may
// be enqueued again.
package xyqueue

