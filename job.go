package gjobs

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
)

type Job struct {
	deps    []*Job
	result  *JobResult
	waiter  chan JobResult
	state   int32
	waiting int32
	mutex   sync.Mutex
	fn      func() (interface{}, error)
}

type JobState int

const (
	JobStateNotStarted         = 0
	JobStateRunning            = 1
	JobStateFinishedButWaiting = 2
	JobStateFinished           = 3
)

type JobResult struct {
	value interface{}
	err   error
}

func NewJob(deps []*Job, fn func() (interface{}, error)) *Job {
	return &Job{
		deps:    deps,
		result:  nil,
		waiter:  make(chan JobResult),
		state:   JobStateNotStarted,
		waiting: 1,
		mutex:   sync.Mutex{},
		fn:      fn,
	}
}

func (job *Job) ExecInBackground() {

	if atomic.CompareAndSwapInt32(&job.state, JobStateNotStarted, JobStateRunning) {

		go func() {
			defer func() {
				if err := recover(); err != nil {
					job.mutex.Lock()
					job.result = &JobResult{
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
			for _, dep := range job.deps {
				dep.ExecInBackground()
				dep.Wait()
			}
			value, err := job.fn()
			result := JobResult{
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

func (job *Job) Wait() {
	if atomic.LoadInt32(&job.state) < JobStateFinished {
		atomic.AddInt32(&job.waiting, 1)
		<-job.waiter
		atomic.AddInt32(&job.waiting, -1)
		if atomic.CompareAndSwapInt32(&job.state, JobStateFinishedButWaiting, JobStateFinished) {
			atomic.AddInt32(&job.waiting, -1)
		}
	}
}

func (job *Job) Get() (interface{}, error) {
	job.ExecInBackground()
	job.Wait()

	job.mutex.Lock()
	var value = job.result.value
	var err = job.result.err
	job.mutex.Unlock()

	return value, err
}

func (job *Job) GetDeps() []*Job {
	return job.deps
}
