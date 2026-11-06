package worker

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestBoundedConcurrencyAndBackpressure(t *testing.T) {
	p := New(2, 1)
	defer p.Close()
	release := make(chan struct{})
	started := make(chan struct{}, 2)
	var active, max atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, e := p.Do(context.Background(), func() (any, error) {
				n := active.Add(1)
				for {
					old := max.Load()
					if n <= old || max.CompareAndSwap(old, n) {
						break
					}
				}
				started <- struct{}{}
				<-release
				active.Add(-1)
				return nil, nil
			})
			if e != nil {
				t.Error(e)
			}
		}()
		<-started
	}
	// Both workers are occupied. Queue exactly one job, then verify overload rejection.
	queued := make(chan Result, 1)
	go func() {
		v, e := p.Do(context.Background(), func() (any, error) { return "queued", nil })
		queued <- Result{v, e}
	}()
	deadline := time.Now().Add(time.Second)
	for len(p.jobs) != 1 {
		if time.Now().After(deadline) {
			t.Fatal("job not queued")
		}
		time.Sleep(time.Millisecond)
	}
	if _, e := p.Do(context.Background(), func() (any, error) { t.Error("overloaded job ran"); return nil, nil }); !errors.Is(e, ErrBusy) {
		t.Fatalf("expected busy: %v", e)
	}
	close(release)
	wg.Wait()
	r := <-queued
	if r.Err != nil || r.Value != "queued" {
		t.Fatalf("queued result: %+v", r)
	}
	if max.Load() != 2 {
		t.Fatalf("concurrency: %d", max.Load())
	}
}
func TestCancellationAndClose(t *testing.T) {
	p := New(1, 1)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := p.Do(ctx, func() (any, error) { t.Error("cancelled job ran"); return nil, nil }); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
	p.Close()
	if _, e := p.Do(context.Background(), func() (any, error) { t.Error("closed pool job ran"); return nil, nil }); !errors.Is(e, ErrClosed) {
		t.Fatal(e)
	}
}
