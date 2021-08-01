package gjobs

import (
	"strings"
	"sync"
	"testing"
	"time"
)

func TestJobs(t *testing.T) {
	run := []string{}
	var jobs = NewJobs()
	jobs.NewJob("a", []string{"b"}, func() (interface{}, error) {
		run = append(run, "a")
		return "a", nil
	})

	jobs.NewJob("b", []string{}, func() (interface{}, error) {
		time.Sleep(time.Millisecond * 50)
		run = append(run, "b")
		return "b", nil
	})

	children := jobs.ExecAndWait()

	if len(children) != 2 {
		t.Error("expect 2 children for ExecAndWait but was", len(children))
	}
	res := strings.Join(run, ",")
	if res != "b,a" {
		t.Error(res, "!=", "b,a")
	}
}

func TestJobsABCDE(t *testing.T) {
	run := []string{}
	var jobs = NewJobs()
	jobs.NewJob("a", []string{"b"}, func() (interface{}, error) {
		run = append(run, "a")
		return "a", nil
	})

	jobs.NewJob("b", []string{"c", "d"}, func() (interface{}, error) {
		time.Sleep(time.Millisecond * 50)
		run = append(run, "b")
		return "b", nil
	})

	jobs.NewJob("c", []string{"d", "e"}, func() (interface{}, error) {
		time.Sleep(time.Millisecond * 50)
		run = append(run, "c")
		return "c", nil
	})

	jobs.NewJob("d", []string{"e"}, func() (interface{}, error) {
		time.Sleep(time.Millisecond * 50)
		run = append(run, "d")
		return "d", nil
	})

	jobs.NewJob("e", []string{}, func() (interface{}, error) {
		time.Sleep(time.Millisecond * 50)
		run = append(run, "e")
		return "e", nil
	})

	jobs.ExecAndWait()

	res := strings.Join(run, ",")
	if res != "e,d,c,b,a" {
		t.Error(res, "!=", "e,d,c,b,a")
	}
}

func TestGrouping(t *testing.T) {
	mx := sync.Mutex{}
	run := []string{}
	var mainJobs = NewJobs()
	mainJobs.NewJob("a1", []string{"b1"}, func() (interface{}, error) {
		mx.Lock()
		run = append(run, "a1")
		mx.Unlock()
		return "a1", nil
	})

	mainJobs.NewJob("b1", []string{}, func() (interface{}, error) {
		time.Sleep(time.Millisecond * 50)
		mx.Lock()
		run = append(run, "b1")
		mx.Unlock()
		return "b1", nil
	})

	var subJobs = NewJobs()
	subJobs.NewJob("a2", []string{"b2"}, func() (interface{}, error) {
		mx.Lock()
		run = append(run, "a2")
		mx.Unlock()
		return "a2", nil
	})

	subJobs.NewJob("b2", []string{}, func() (interface{}, error) {
		time.Sleep(time.Millisecond * 50)
		mx.Lock()
		run = append(run, "b2")
		mx.Unlock()
		return "b2", nil
	})

	mainJobs.AddGroup("sub", subJobs)

	mainJobs.ExecAndWait()

	if len(run) != 4 {
		t.Error(strings.Join(run, ","), " not executed well")
	}
}
