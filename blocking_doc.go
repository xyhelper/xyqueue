package xyqueue

// Advanced: Blocking and Timeout Patterns
//
// xyqueue exposes a non-blocking, concurrency-safe FIFO API. If you need
// blocking semantics (e.g., producers wake consumers when items arrive) or
// context-aware timeouts, layer a thin wrapper using sync.Cond and context.
//
// Design notes:
//   - Broadcast only when an Enqueue actually adds a new value (in de-dup
//     mode, repeated values may be ignored and should not spuriously wake
//     waiters).
//   - Use the standard "wait in a loop" pattern to handle spurious wakeups.
//   - Prefer context for cancellation and deadlines to avoid goroutine leaks.
//
// Minimal outline:
//
//  type BQ struct {
//      mu sync.Mutex
//      cv *sync.Cond
//      q  *xyqueue.Queue[string]
//  }
//
//  func newBQ(dedup bool) *BQ {
//      b := &BQ{q: xyqueue.New[string](dedup)}
//      b.cv = sync.NewCond(&b.mu)
//      return b
//  }
//
//  func (b *BQ) Put(v string) {
//      b.mu.Lock()
//      added := b.q.Enqueue(v)
//      b.mu.Unlock()
//      if added { b.cv.Broadcast() }
//  }
//
//  func (b *BQ) Take(ctx context.Context) (string, bool) {
//      b.mu.Lock()
//      defer b.mu.Unlock()
//      for b.q.Len() == 0 {
//          done := make(chan struct{})
//          go func() { b.cv.Wait(); close(done) }()
//          select {
//          case <-ctx.Done():
//              return "", false
//          case <-done:
//          }
//      }
//      return b.q.Dequeue()
//  }
//
// For systems where a condition variable is unnecessary, a lightweight polling
// loop with a deadline is another option:
//
//  deadline := time.After(200 * time.Millisecond)
//  for {
//      if v, ok := q.Dequeue(); ok { /* use v */ break }
//      select {
//      case <-deadline:
//          // timeout handling
//          return
//      default:
//          time.Sleep(1 * time.Millisecond)
//      }
//  }

