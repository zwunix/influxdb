package buffer

import (
	"context"
	"sync"
)

type Pool struct {
	New func() interface{}

	buffers chan interface{}
	count   int
	max     int
	mu      sync.RWMutex
}

func NewPool(max int) *Pool {
	return &Pool{
		buffers: make(chan interface{}, max),
		max:     max,
	}
}

func (p *Pool) Get(ctx context.Context) (interface{}, bool) {
	select {
	case buf := <-p.buffers:
		return buf, true
	case <-ctx.Done():
		return nil, false
	default:
		p.mu.Lock()
		if p.count < p.max {
			if p.New == nil {
				p.mu.Unlock()
				return nil, false
			}
			p.count++
			p.mu.Unlock()
			return p.New(), true
		}
		p.mu.Unlock()

		select {
		case buf := <-p.buffers:
			return buf, true
		case <-ctx.Done():
			return nil, false
		}
	}
}

func (p *Pool) Put(x interface{}) {
	p.buffers <- x
}

func (p *Pool) Ch() <-chan interface{} {
	return p.buffers
}
