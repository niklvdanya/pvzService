package workerpool

import (
	"context"
	"sync"
	"sync/atomic"
)

type Pool struct {
	jobs    chan Job
	kill    chan struct{} // сигналы для метода Resize
	rootCtx context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	workers int32
}

func New(workerCnt, queueSize int) *Pool {
	if workerCnt <= 0 {
		workerCnt = 1
	}
	if queueSize <= 0 {
		queueSize = workerCnt * 4
	}

	ctx, cancel := context.WithCancel(context.Background())
	p := &Pool{
		jobs:    make(chan Job, queueSize),
		kill:    make(chan struct{}, workerCnt*2),
		rootCtx: ctx,
		cancel:  cancel,
	}
	atomic.StoreInt32(&p.workers, int32(workerCnt))
	p.spawn(workerCnt)
	return p
}

func (p *Pool) spawn(n int) {
	for i := 0; i < n; i++ {
		p.wg.Add(1)
		go p.worker()
	}
}

func (p *Pool) worker() {
	defer p.wg.Done()
	for {
		select {
		case <-p.kill:
			return
		case job, ok := <-p.jobs:
			if !ok {
				return
			}
			// проверяем отмену контеста до выполнения
			if errCtx := job.Ctx.Err(); errCtx != nil {
				select {
				case job.Resp <- Response{Err: errCtx}:
				default:
				}
				continue
			}

			v, err := job.Run(job.Ctx)
			select {
			case job.Resp <- Response{Value: v, Err: err}:
			case <-job.Ctx.Done():
			}
		case <-p.rootCtx.Done():
			return
		}
	}
}

func (p *Pool) Submit(j Job) {
	select {
	case p.jobs <- j:
	default:
		// случай для заполненной очереди: выполняем задачу синхронно, чтобы не задерживать вызов rpc‑хендлера
		v, err := j.Run(j.Ctx)
		select {
		case j.Resp <- Response{Value: v, Err: err}:
		case <-j.Ctx.Done():
		}
	}
}

func (p *Pool) Resize(n int) {
	if n < 0 {
		return
	}
	cur := int(atomic.LoadInt32(&p.workers))
	if n == cur {
		return
	}

	if n > cur {
		p.spawn(n - cur)
	} else {
		diff := cur - n
		// первые curr - n воркеров, которые попадут в case <-p.kill: будут завершены
		for i := 0; i < diff; i++ {
			p.kill <- struct{}{}
		}
	}
	atomic.StoreInt32(&p.workers, int32(n))
}

func (p *Pool) Close() {
	p.cancel()
	close(p.jobs)
	for i := int32(0); i < atomic.LoadInt32(&p.workers); i++ {
		p.kill <- struct{}{}
	}
	p.wg.Wait()
	close(p.kill)
}
