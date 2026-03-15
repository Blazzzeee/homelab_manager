package job

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
)

const DEFAULT_WORKERS = 18

// the pool only recieves singular view
// of what to execute via Dispatch API
// the workerpool owner should listen over the
// Results channel to recieve processed jobs
type WorkerPool struct {
	maxWorkers int
	// Recives from the global queue
	jobs     <-chan *Job
	Results  chan *WorkerResult
	wg       sync.WaitGroup
	stopOnce sync.Once
}

type WorkerResult struct {
	JobID  ID
	Err    error
	Output string
}

func NewWorker(maxWorkers int) *WorkerPool {
	if maxWorkers <= 0 {
		fmt.Println("WARN: Atleast one worker needed , using fallback: ", DEFAULT_WORKERS)
		maxWorkers = DEFAULT_WORKERS
	}
	return &WorkerPool{
		maxWorkers: maxWorkers,
		// just meant for go routine buffering by the time worker picks it up
		Results: make(chan *WorkerResult, maxWorkers),
	}

}

// Starts all the workers , this can be over ssh etc in future
func (wp *WorkerPool) Start() {
	for i := 0; i < wp.maxWorkers; i++ {
		wp.wg.Add(1)

		go wp.WorkerLoop(i)
	}
}

// // closes resources on all nodes
// func (wp *WorkerPool) Shutdown() {
// 	wp.stopOnce.Do(func() {
// 		close(wp.jobs)
// 	})
// 	wp.wg.Wait()
// 	close(wp.Results)
// }

// This is the loop which executes incoming tasks from the tasks channel
func (wp *WorkerPool) WorkerLoop(id int) {
	defer wp.wg.Done()
	for {
		task, ok := <-wp.jobs
		if !ok {
			// TODO: error logs
			return
		}
		wp.execute(task)

	}
}

// Shell commands are launched from here
//
// This call is a blocking call ,
// meaning that the number of commands that
// can run at a time depends on the number of go routines
func (wp *WorkerPool) execute(job *Job) {

	// skip timed out jobs and cancelled jobs
	if job.State == StateCancelled || job.State == StateTimeout {
		return
	}

	if len(job.Cmd) == 0 {
		wp.Results <- &WorkerResult{
			JobID:  job.Id,
			Err:    fmt.Errorf("empty command"),
			Output: "",
		}
		return
	}

	command := job.Cmd[0]
	args := job.Cmd[1:]

	// execute command safely
	cmd := exec.Command(command, args...)

	output, err := cmd.CombinedOutput()

	res := &WorkerResult{
		JobID:  job.Id,
		Err:    err,
		Output: string(output),
	}

	wp.Results <- res

}
