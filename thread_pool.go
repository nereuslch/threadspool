package threadspool

import (
	"sync"
	"errors"
	"sync/atomic"
)

var (
	ErrPoolRetChanNotRun = errors.New("the pool is not running")
)

type Pool struct {
	queueSize int64
	retC      chan workRequest
	works     []*workWrapper
	sync.RWMutex
}

func (p *Pool) WorkSize() int {
	p.RLock()
	defer p.RUnlock()
	return len(p.works)
}

func (p *Pool) QueueLength() int64 {
	return atomic.LoadInt64(&p.queueSize)
}

func New(n int) *Pool {
	p := &Pool{
		retC: make(chan workRequest),
	}
	p.SetWorkSize(n)
	return p
}

func (p *Pool) SetWorkSize(n int) {
	p.Lock()
	defer p.Unlock()

	for idx := len(p.works); idx < n; idx ++ {
		p.works = append(p.works, newWorkWrapper(p.retC))
	}

	for idx := len(p.works); idx > n; idx -- {
		p.works[idx].exist()
	}

	p.works = p.works[:n]
}

func (p *Pool) Push(j Job) (func(), JobStatusQueryer, <-chan error) {
	atomic.AddInt64(&p.queueSize, 1)
	work, open := <-p.retC
	if !open {
		panic(ErrPoolRetChanNotRun)
	}

	work.jobC <- struct {
		Job
		jobCounter
	}{j, jobCounter{
		add: func() { atomic.AddInt64(&p.queueSize, 1) },
		des: func() { atomic.AddInt64(&p.queueSize, -1) },
	}}
	return work.cancelFunc, work.queryer, work.retC
}
