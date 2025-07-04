package workerpool

type Pool struct {
	jobs chan Job
}

func New(n, qsize int) *Pool {
	p := &Pool{jobs: make(chan Job, qsize)}
	for i := 0; i < n; i++ {
		go p.worker()
	}
	return p
}

func (p *Pool) Submit(j Job) {
	p.jobs <- j
}

func (p *Pool) worker() {
	for j := range p.jobs {
		v, err := j.Run(j.Ctx)
		select {
		case j.Resp <- Response{v, err}:
		case <-j.Ctx.Done():
		}
	}
}
