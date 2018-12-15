package threadspool

import (
	"sync"
	"sync/atomic"
)

type workWrapper struct {
	job        Job
	status     atomic.Value
	reqC       chan<- workRequest
	stopC      chan struct{}
	stoppedC   chan struct{}
	cancelJobC chan struct{}
	sync.RWMutex
}

type workRequest struct {
	jobC chan<- struct {
		Job
		jobCounter
	}
	retC       <-chan error
	cancelFunc func()
	queryer    JobStatusQueryer
}

func newWorkWrapper(req chan<- workRequest) *workWrapper {
	wp := &workWrapper{
		reqC:       req,
		stopC:      make(chan struct{}),
		stoppedC:   make(chan struct{}),
		cancelJobC: make(chan struct{}),
	}

	wp.status.Store(StatusJobNoAccept)
	go wp.run()
	return wp
}

func (w *workWrapper) run() {
	jobChan, retChan := make(chan struct {
		Job
		jobCounter
	}), make(chan error)
	defer func() {
		close(w.stopC)
		close(w.stoppedC)
	}()

	for {
		select {
		case w.reqC <- workRequest{
			jobChan,
			retChan,
			w.cancel,
			JobStatusQueryFunc(w.jobStatus),
		}:
			select {
			case job := <-jobChan:
				job.add()
				w.status.Store(StatusJobRunning)
				w.setJob(job.Job.(Job))
				select {
				case retChan <- w.getJob().Process():
					w.status.Store(StatusJobDone)
				case <-w.cancelJobC:
					w.status.Store(StatusJobCancel)
					w.cancelJobC = make(chan struct{})
				}
				job.des()
			case <-w.cancelJobC:
				w.status.Store(StatusJobCancel)
				w.cancelJobC = make(chan struct{})
			}
		case <-w.stopC:
			w.cancel()
			return
		}

		w.setJob(nil)
	}
}

func (w *workWrapper) jobStatus() JobStatus {
	return w.status.Load().(JobStatus)
}

func (w *workWrapper) exist() {
	close(w.stopC)
	<-w.stoppedC
}

func (w *workWrapper) getJob() Job {
	w.RLock()
	defer w.RUnlock()
	return w.job
}

func (w *workWrapper) setJob(j Job) {
	w.Lock()
	defer w.Unlock()
	w.job = j
}

func (w *workWrapper) cancel() {
	w.RLock()
	defer w.RUnlock()
	if w.job == nil {
		return
	}

	close(w.cancelJobC)
	w.job.Cancel()
}
