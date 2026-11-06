// Package worker bounds concurrent mutations and queued work. Jobs never own HTTP state.
package worker

import (
	"context"
	"errors"
	"sync"
)

var ErrBusy = errors.New("server busy; please retry")
var ErrClosed = errors.New("worker pool is closed")

type Result struct {
	Value any
	Err   error
}
type job struct {
	ctx    context.Context
	run    func() (any, error)
	result chan Result
}
type Pool struct {
	jobs   chan job
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func New(size, queue int) *Pool {
	if size < 1 || queue < 1 {
		panic("worker and queue sizes must be positive")
	}
	ctx, cancel := context.WithCancel(context.Background())
	p := &Pool{jobs: make(chan job, queue), cancel: cancel, ctx: ctx}
	for i := 0; i < size; i++ {
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case j := <-p.jobs:
					if j.ctx.Err() != nil {
						j.result <- Result{Err: j.ctx.Err()}
						continue
					}
					value, err := j.run()
					j.result <- Result{value, err}
				}
			}
		}()
	}
	return p
}
func (p *Pool) Do(ctx context.Context, run func() (any, error)) (any, error) {
	result := make(chan Result, 1)
	select {
	case <-p.ctx.Done():
		return nil, ErrClosed
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	select {
	case p.jobs <- job{ctx, run, result}:
	default:
		return nil, ErrBusy
	}
	select {
	case r := <-result:
		return r.Value, r.Err
	case <-p.ctx.Done():
		return nil, ErrClosed
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Close is called after the HTTP server has drained active requests.
func (p *Pool) Close() { p.cancel(); p.wg.Wait() }
