package job

import (
	"context"
	"sync"
	"time"
)

type Task struct {
	// The task has direct association to lifecycle of job
	// to simplify query we link them with same job id
	Id      ID
	Argv    []string
	CWD     string
	Timeout time.Duration
}

// the pool only recieves singular view
// of what to execute via Dispatch API
type WorkerPool struct {
	maxWorkers int
	tasks      chan Task
	results    chan WorkerResult
	wg         sync.WaitGroup
	stopOnce   sync.Once
	quit       chan struct{}
}

type WorkerResult struct {
	JobID    ID
	Exitcode int
	Err      error
}

func NewWorkerPool(maxWorkers int) *WorkerPool {
	return &WorkerPool{
		maxWorkers: maxWorkers,
		// this shouldnt be unbounded
		tasks:   make(chan Task),
		results: make(chan WorkerResult),
		quit:    make(chan struct{}),
	}

}

// Starts all the workers , this can be over ssh etc in future
func (wp *WorkerPool) Start() {
	for i := 0; i < wp.maxWorkers; i++ {
		wp.wg.Add(1)

		go wp.WorkerLoop(i)
	}
}

// This will be visible to the job queue
func (wp *WorkerPool) Dispatch(ctx context.Context, task *Task) {

	// Send the task to be picked up by some worker
	wp.tasks <- *task
}

// closes resources on nodes
func (wp *WorkerPool) Shutdown(worker *WorkerPool) {
	wp.stopOnce.Do(func() {
		close(wp.tasks)
		close(wp.quit)
	})
	wp.wg.Wait()
}

func (wp *WorkerPool) WorkerLoop(id int) {
	defer wp.wg.Done()
	for {
		select {
		case task := <-wp.tasks:
			wp.execute(&task)
		case <-wp.quit:
			return
		}

	}
}

func (wp *WorkerPool) execute(task *Task) {

}
