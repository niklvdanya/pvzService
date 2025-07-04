package workerpool

import (
	"context"
	"sync"
)

type Pool struct {
	jobs       chan Job
	cancelRoot context.CancelFunc
	wg         sync.WaitGroup
	mu         sync.Mutex
	workers    int
}

func New(workerCnt, queueSize int) *Pool {
	ctx, cancel := context.WithCancel(context.Background())
	p := &Pool{
		jobs:       make(chan Job, queueSize),
		cancelRoot: cancel,
		workers:    workerCnt,
	}
	p.resize(ctx, workerCnt)
	return p
}

func (p *Pool) Submit(j Job) {
	p.jobs <- j
}

func (p *Pool) Close() {
	p.cancelRoot()
	p.wg.Wait()
	close(p.jobs)
}

func (p *Pool) Resize(n int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if n == p.workers || n <= 0 {
		return
	}
	diff := n - p.workers
	ctx := context.Background()
	if diff > 0 {
		p.resize(ctx, diff)
	} else {
		for i := 0; i < -diff; i++ {
			p.Submit(Job{Ctx: ctx, Run: func(context.Context) (any, error) { return nil, context.Canceled }, Resp: make(chan Response, 1)})
		}
	}
	p.workers = n
}

func (p *Pool) resize(ctx context.Context, add int) {
	for i := 0; i < add; i++ {
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			for {
				select {
				case j := <-p.jobs:
					v, err := j.Run(j.Ctx)
					select {
					case j.Resp <- Response{v, err}:
					case <-j.Ctx.Done():
					}
				case <-ctx.Done():
					return
				}
			}
		}()
	}
}
