package concurrency

import (
	"time"
)

type ThrottledWorker struct {
	jobCallback func(arg string) error
}

func NewThrottledWorker(jobCallback func(arg string) error) ThrottledWorker {
	return ThrottledWorker{jobCallback: jobCallback}
}

func (w *ThrottledWorker) Run(jobArgs []string) {

	jobArgsChannel := make(chan string, len(jobArgs))

	for _, arg := range jobArgs {
		jobArgsChannel <- arg
	}
	close(jobArgsChannel)
	limiter := time.NewTicker(100 * time.Millisecond)

	for arg := range jobArgsChannel {
		<-limiter.C
		w.jobCallback(arg)
	}

}
