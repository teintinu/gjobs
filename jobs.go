package gjobs

import (
	"sync"
)

type Jobs struct {
	mutex    *sync.Mutex
	pending  map[string](func() *Job)
	children map[string]*Job
}

func NewJobs() *Jobs {
	return &Jobs{
		mutex:    &sync.Mutex{},
		pending:  map[string](func() *Job){},
		children: map[string]*Job{},
	}
}

func (jobs *Jobs) NewJob(name string, deps []string, fn func() (interface{}, error)) {
	println("NewJob-begin ", name)
	jobs.mutex.Lock()
	if _, exists := jobs.pending[name]; exists {
		jobs.mutex.Unlock()
		panic("duplicate job name:" + name)
	}
	if _, exists := jobs.children[name]; exists {
		jobs.mutex.Unlock()
		panic("duplicate job name:" + name)
	}
	jobs.pending[name] = func() *Job {
		println("NewJob-resolving ", name)
		jobs.mutex.Lock()
		println("NewJob-resolving-locked ", name)
		if _, wasResolved := jobs.children[name]; wasResolved {
			jobs.mutex.Unlock()
			return nil
		}
		jdeps := []*Job{}
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
				println("NewJob-panbcresolved: ", depname, " on ", name)
				panic("can't resolve job name: " + depname + " on " + name)
			}
		}
		j := NewJob(jdeps, fn)
		jobs.children[name] = j
		delete(jobs.pending, name)
		jobs.mutex.Unlock()
		println("NewJob-resolved: ", name)
		return j
	}
	jobs.mutex.Unlock()
	println("NewJob-end: ", name)
}

func (jobs *Jobs) ExecInBackground() {
	factories := [](func() *Job){}
	newJobs := []*Job{}
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

func (jobs *Jobs) ExecAndWait() []*Job {
	jobs.ExecInBackground()
	jobs.mutex.Lock()
	children := []*Job{}
	for _, job := range jobs.children {
		children = append(children, job)
	}
	jobs.mutex.Unlock()
	for _, job := range children {
		job.Wait()
	}
	return children
}

func (jobs *Jobs) AddGroup(name string, depJobs *Jobs) {
	jobs.NewJob(name, []string{}, func() (interface{}, error) {
		println("subexec")
		depResults := depJobs.ExecAndWait()
		println("subexec-after")
		for _, depResult := range depResults {
			err := depResult.result.err
			if err != nil {
				return depResults, err
			}
		}
		return depResults, nil
	})
}
