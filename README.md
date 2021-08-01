# gjobs

Easy jobs in Golang

# how to install

```bash
go get github.com/tentinu/gjobs
```

# how to use

```go

import (
	"github.com/teintinu/gjobs"
)

jobs := gjobs.NewJobs()

// the job "a" depends on jobs "b" and "c"
jobs.NewJob("a", []string{"b", "c"}, func() (interface{}, error) {
	return "a", nil
})
// the job "b" depends on job "c"
jobs.NewJob("b", []string{"c"}, func() (interface{}, error) {
	return "b", nil
})
// the job "c" has no dependencies
jobs.NewJob("c", []string{}, func() (interface{}, error) {
	return "c", nil
})

jobs.Run() // will exec firstly c, b and finally a

value,err := jobs.Get("a") // you can get the job return
```

## alternative

```go

c := gjobs.NewJob([]GJob{}, func() (interface{}, error) {
	return "c", nil
})

b := gjobs.NewJob([]GJob{c}, func() (interface{}, error) {
	return "b", nil
})

a := gjobs.NewJob([]GJob{b,c}, func() (interface{}, error) {
	return "a", nil
})

value,err := a.Get() // will exec firstly c, b and finally a

```