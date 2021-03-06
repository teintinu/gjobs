package gjobs

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
)

var VerbosityEvent func(args ...interface{})

type GJob struct {
	deps    []*GJob
	result  *GJobResult
	waiter  chan GJobResult
	state   int32
	waiting int32
	mutex   sync.Mutex
	fn      func() (interface{}, error)
}

type GJobState int

const (
	JobStateNotStarted         = 0
	JobStateRunning            = 1
	JobStateFinishedButWaiting = 2
	JobStateFinished           = 3
)

type GJobResult struct {
	value interface{}
	err   error
}

func NewJob(deps []*GJob, fn func() (interface{}, error)) *GJob {
	return &GJob{
		deps:    deps,
		result:  nil,
		waiter:  make(chan GJobResult),
		state:   JobStateNotStarted,
		waiting: 1,
		mutex:   sync.Mutex{},
		fn:      fn,
	}
}

func (job *GJob) ExecInBackground() {

	if atomic.CompareAndSwapInt32(&job.state, JobStateNotStarted, JobStateRunning) {
		if VerbosityEvent != nil {
			VerbosityEvent("GJob.ExecInBackground", job)
		}
		go func() {
			defer func() {
				if VerbosityEvent != nil {
					VerbosityEvent("GJob.ExecInBackground.finally", job)
				}
				if err := recover(); err != nil {
					if VerbosityEvent != nil {
						VerbosityEvent("GJob.ExecInBackground.recover()", job)
					}
					job.mutex.Lock()
					job.result = &GJobResult{
						value: nil,
						err:   errors.New(fmt.Sprint(err)),
					}
					atomic.CompareAndSwapInt32(&job.state, JobStateRunning, JobStateFinishedButWaiting)
					job.mutex.Unlock()
					for atomic.LoadInt32(&job.waiting) > 0 {
						job.waiter <- *job.result
					}
				}
			}()
			if VerbosityEvent != nil {
				VerbosityEvent("GJob.ExecInBackground.deps", job)
			}
			for _, dep := range job.deps {
				dep.ExecInBackground()
				dep.Wait()
			}
			if VerbosityEvent != nil {
				VerbosityEvent("GJob.ExecInBackground.fn", job)
			}
			value, err := job.fn()
			result := GJobResult{
				value: value,
				err:   err,
			}
			job.mutex.Lock()
			job.result = &result
			job.mutex.Unlock()

			atomic.CompareAndSwapInt32(&job.state, JobStateRunning, JobStateFinishedButWaiting)
			for atomic.LoadInt32(&job.waiting) > 0 {
				job.waiter <- result
			}
		}()
	}
}

func (job *GJob) Wait() {
	if atomic.LoadInt32(&job.state) < JobStateFinished {
		if VerbosityEvent != nil {
			VerbosityEvent("GJob.Wait.start", job)
		}
		atomic.AddInt32(&job.waiting, 1)
		<-job.waiter
		atomic.AddInt32(&job.waiting, -1)
		if atomic.CompareAndSwapInt32(&job.state, JobStateFinishedButWaiting, JobStateFinished) {
			atomic.AddInt32(&job.waiting, -1)
		}
		if VerbosityEvent != nil {
			VerbosityEvent("GJob.ExecInBackground.finish", job)
		}
	}
}

func (job *GJob) Get() (interface{}, error) {
	job.ExecInBackground()
	job.Wait()

	job.mutex.Lock()
	var value = job.result.value
	var err = job.result.err
	job.mutex.Unlock()

	return value, err
}

func (job *GJob) GetDeps() []*GJob {
	return job.deps
}
