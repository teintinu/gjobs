package gjobs

import (
	"errors"
	"sync"
)

type GJobs struct {
	mutex    *sync.Mutex
	pending  map[string](func() *GJob)
	children map[string]*GJob
}

func NewJobs() *GJobs {
	return &GJobs{
		mutex:    &sync.Mutex{},
		pending:  map[string](func() *GJob){},
		children: map[string]*GJob{},
	}
}

func (jobs *GJobs) NewJob(name string, deps []string, fn func() (interface{}, error)) {
	jobs.mutex.Lock()
	if _, exists := jobs.pending[name]; exists {
		jobs.mutex.Unlock()
		panic("duplicated job name:" + name)
	}
	if _, exists := jobs.children[name]; exists {
		jobs.mutex.Unlock()
		panic("duplicated job name:" + name)
	}
	jobs.pending[name] = func() *GJob {
		jobs.mutex.Lock()
		if _, wasResolved := jobs.children[name]; wasResolved {
			jobs.mutex.Unlock()
			return nil
		}
		jdeps := []*GJob{}
		for _, depname := range deps {
			if jdep, exists := jobs.children[depname]; exists {
				jdeps = append(jdeps, jdep)
			} else if subjobfactory, exists := jobs.pending[depname]; exists {
				jobs.mutex.Unlock()
				jdep := subjobfactory()
				jobs.mutex.Lock()
				jdeps = append(jdeps, jdep)
			} else {
				jobs.mutex.Unlock()
				panic("can't resolve job name: " + depname + " on " + name)
			}
		}
		j := NewJob(jdeps, fn)
		jobs.children[name] = j
		delete(jobs.pending, name)
		jobs.mutex.Unlock()
		return j
	}
	jobs.mutex.Unlock()
}

func (jobs *GJobs) ExecInBackground() {
	factories := [](func() *GJob){}
	newJobs := []*GJob{}
	jobs.mutex.Lock()
	for _, jobfactory := range jobs.pending {
		factories = append(factories, jobfactory)
	}
	jobs.mutex.Unlock()
	for _, jobfactory := range factories {
		job := jobfactory()
		if job != nil {
			newJobs = append(newJobs, job)
		}
	}
	for _, job := range newJobs {
		job.ExecInBackground()
	}
}

func (jobs *GJobs) Run() []*GJob {
	jobs.ExecInBackground()
	jobs.mutex.Lock()
	children := []*GJob{}
	for _, job := range jobs.children {
		children = append(children, job)
	}
	jobs.mutex.Unlock()
	for _, job := range children {
		job.Wait()
	}
	return children
}

func (jobs *GJobs) AddGroup(name string, depJobs *GJobs) {
	jobs.NewJob(name, []string{}, func() (interface{}, error) {
		depResults := depJobs.Run()
		for _, depResult := range depResults {
			err := depResult.result.err
			if err != nil {
				return depResults, err
			}
		}
		return depResults, nil
	})
}

func (jobs *GJobs) Get(name string) (interface{}, error) {
	jobs.Run()
	if res, exists := jobs.children[name]; exists {
		return res.result.value, res.result.err
	} else {
		return nil, errors.New("there is no job " + name + " job")
	}
}
