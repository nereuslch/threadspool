package threadspool

type JobStatus string

var (
	StatusJobNoAccept JobStatus = "no job accept"
	StatusJobRunning  JobStatus = "accept by thread pool"
	StatusJobDone     JobStatus = "finish the job"
	StatusJobCancel   JobStatus = "client cancel the job"
)

type JobStatusQueryFunc func() JobStatus

func (j JobStatusQueryFunc) Status() JobStatus {
	return j()
}

type JobStatusQueryer interface{ Status() JobStatus }

type Job interface {
	Process() error
	Cancel()
}

type jobCounter struct {
	add func()
	des func()
}
