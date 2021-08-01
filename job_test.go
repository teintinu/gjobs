package gjobs

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestJob(t *testing.T) {
	mx := sync.Mutex{}
	run := []string{}
	job := NewJob([]*Job{}, func() (interface{}, error) {
		time.Sleep(time.Millisecond * 50)
		return "ok", nil
	})
	go func() {
		mx.Lock()
		run = append(run, "Doing")
		mx.Unlock()
		job.ExecInBackground()
	}()
	mx.Lock()
	run = append(run, "Wait")
	mx.Unlock()
	job.Wait()
	vresult, err := job.Get()
	mx.Lock()
	run = append(run, "check")
	mx.Unlock()
	res := strings.Join(run, ",")
	if res != "Wait,Doing,check" {
		t.Error(res, "!=", "Wait,Doing,check")
	}
	result := vresult.(string)
	if result != "ok" {
		t.Error("result actual=", result, " expected=ok")
	}
	if err != nil {
		t.Error("err actual=", err.Error(), " expect=nil")
	}
	jdeps := len(job.GetDeps())
	if jdeps != 0 {
		t.Error("jdeps len(job.GetDeps())=", jdeps, " expect=0")
	}
}

func TestJobWithPanic(t *testing.T) {
	mx := sync.Mutex{}
	run := []string{}
	a := len(run)
	job := NewJob([]*Job{}, func() (interface{}, error) {
		return fmt.Sprint(1 / a), nil
	})
	go func() {
		mx.Lock()
		run = append(run, "Doing")
		mx.Unlock()
		job.ExecInBackground()
	}()
	mx.Lock()
	run = append(run, "Wait")
	mx.Unlock()
	job.Wait()
	_, err := job.Get()
	mx.Lock()
	run = append(run, "check")
	mx.Unlock()
	res := strings.Join(run, ",")
	if res != "Wait,Doing,check" {
		t.Error(res, "!=", "Wait,Doing,check")
	}
	serr := fmt.Sprint(err)
	if serr != "runtime error: integer divide by zero" {
		t.Error("err actual=", serr, " expect=runtime error: integer divide by zero")
	}
	jdeps := len(job.GetDeps())
	if jdeps != 0 {
		t.Error("jdeps len(job.GetDeps())=", jdeps, " expect=0")
	}
}

func TestJobWithDep(t *testing.T) {
	mx := sync.Mutex{}
	run := []string{}
	job1 := NewJob([]*Job{}, func() (interface{}, error) {
		time.Sleep(time.Millisecond * 50)
		mx.Lock()
		run = append(run, "1.Doing")
		mx.Unlock()
		return "11", nil
	})
	job2 := NewJob([]*Job{job1}, func() (interface{}, error) {
		mx.Lock()
		run = append(run, "2.Doing")
		mx.Unlock()
		return "22", nil
	})
	go func() {
		mx.Lock()
		run = append(run, "Exec")
		mx.Unlock()
		job1.ExecInBackground()
		job2.ExecInBackground()
	}()
	mx.Lock()
	run = append(run, "Wait")
	mx.Unlock()
	job1.Wait()
	job2.Wait()
	vresult1, err1 := job1.Get()
	vresult2, err2 := job2.Get()
	mx.Lock()
	run = append(run, "check")
	mx.Unlock()
	res := strings.Join(run, ",")
	if res != "Wait,Exec,1.Doing,2.Doing,check" {
		t.Error(res, "!=", "Wait,Exec,1.Doing,2.Doing,check")
	}
	result1 := vresult1.(string)
	if result1 != "11" {
		t.Error("result1 actual=", result1, " expected=11")
	}
	if err1 != nil {
		t.Error("err1 actual=", err1.Error(), " expect=nil")
	}
	result2 := vresult2.(string)
	if result2 != "22" {
		t.Error("result2 actual=", result2, " expected=22")
	}
	if err2 != nil {
		t.Error("err2 actual=", err2.Error(), " expect=nil")
	}
	jdeps1 := len(job1.GetDeps())
	if jdeps1 != 0 {
		t.Error("jdeps1 len(jdeps1.GetDeps())=", jdeps1, " expect=0")
	}
	jdeps2 := len(job2.GetDeps())
	if jdeps2 != 1 {
		t.Error("jdeps2 len(jdeps2.GetDeps())=", jdeps2, " expect=1")
	}
}

func TestJobWithGetDepFromOther(t *testing.T) {
	run := []string{}
	job1 := NewJob([]*Job{}, func() (interface{}, error) {
		time.Sleep(time.Millisecond * 50)
		return "11", nil
	})
	job2 := NewJob([]*Job{}, func() (interface{}, error) {
		r1, err1 := job1.Get()
		if err1 != nil {
			return nil, err1
		}
		sr1 := r1.(string)
		time.Sleep(time.Millisecond * 50)
		return sr1 + "." + "22", nil
	})
	job2.ExecInBackground()
	job1.ExecInBackground()
	run = append(run, "Wait")
	vresult1, err1 := job1.Get()
	vresult2, err2 := job2.Get()
	run = append(run, "check")
	res := strings.Join(run, ",")
	if res != "Wait,check" {
		t.Error(res, "!=", "Wait,Done1,Exec1,Done2,Exec2,check")
	}
	result1 := vresult1.(string)
	if result1 != "11" {
		t.Error("result1 actual=", result1, " expected=11")
	}
	if err1 != nil {
		t.Error("err1 actual=", err1.Error(), " expect=nil")
	}
	result2 := vresult2.(string)
	if result2 != "11.22" {
		t.Error("result2 actual=", result2, " expected=11.22")
	}
	if err2 != nil {
		t.Error("err2 actual=", err2.Error(), " expect=nil")
	}
	jdeps1 := len(job1.GetDeps())
	if jdeps1 != 0 {
		t.Error("jdeps1 len(jdeps1.GetDeps())=", jdeps1, " expect=0")
	}
	jdeps2 := len(job2.GetDeps())
	if jdeps2 != 0 {
		t.Error("jdeps2 len(jdeps2.GetDeps())=", jdeps2, " expect=0")
	}
}
