package job

import (
	"context"
	"fmt"
	"strconv"
	"time"
)

const queueSize = 1000

type EventType int

const (
	EventCancel EventType = iota
	EventResult
	EventTimeout
	EventShutdown
)

type Event struct {
	Type   EventType
	JobId  ID
	Result *WorkerResult
	// caller waits on this
	// TODO: Why do we need this ?
	Reply chan *any
}

// the consumer should start the event loop
// // the consumer should start the event loop
type runner struct {
	store  *JobStore
	queue  *Queue
	worker *WorkerPool
	result <-chan *WorkerResult
	events chan *Event
}

// TODO: use this interface for type safety
type Store interface {
	Get(id ID) (*Job, error)
	Push(job *Job) error
	Pop(id ID) (*Job, error)
	// TODO: Allow update to mutate Job
}

func NewRunner() *runner {

	runner := &runner{
		queue:  NewQueue(queueSize),
		worker: NewWorker(queueSize),
		store:  NewStore(),
		// TODO: Deal with size in a better way
		events: make(chan *Event, queueSize),
	}

	runner.result = runner.worker.Results
	runner.worker.jobs = runner.queue.buf
	runner.worker.Start()
	fmt.Println("Started worker pool...")

	return runner
}

func TransitionToFailed(runner *runner, job *Job) error {
	store := runner.store
	return store.Update(job.Id, func(j *Job) error {
		return j.Transition(StateFailed)
	})
}

func (runner *runner) Submit(ctx context.Context, cmd []string, options Options) (ID, error) {
	job := MakeJob(cmd, options)
	store := runner.store

	// add it in our persistence layer
	if err := store.Push(job); err != nil {
		_ = TransitionToFailed(runner, job)
		return "", err
	}

	// This will now get picked up by the worker
	if err := runner.queue.Enqueue(job); err != nil {
		_ = TransitionToFailed(runner, job)
		return "", err
	}

	// if job was enqueued mark it enqueue to the store
	store.Update(job.Id, func(j *Job) error {
		// init state machine
		// transition returns error
		return j.Transition(StateQueued)

	})

	return job.Id, nil
}

func (runner *runner) Get(id ID) (*Job, error) {
	// maybe we should do more lol!
	return runner.store.Get(id)

}

func MakeJob(cmd []string, options Options) *Job {

	job := &Job{
		Id:      JobId(),
		Options: options,
		State:   StateCreated,
		TimeStamps: JobTimeInfo{
			CreatedAt: time.Now(),
		},
	}

	return job
}

func (runner *runner) Cancel(ctx context.Context, id ID) error {
	ev := &Event{Type: EventCancel, JobId: id}

	select {
	case runner.events <- ev:
		return nil

	default:
		return fmt.Errorf("Cancel failed")
	}
}

func (runner *runner) List(ctx context.Context) []*Job {
	return runner.store.GetAll()
}

func JobId() ID {
	return ID(strconv.FormatInt(time.Now().UnixNano(), 10))
}

func (r *runner) loop(ctx context.Context) {
	timers := map[ID]*time.Timer{}

	for {
		select {
		case ev := <-r.events:

			switch ev.Type {

			case EventCancel:
				_, err := r.store.Get(ev.JobId)

				if err == ErrJobNotFound {
					// inform job not found
				}

				// The transition will handle if we can mark it cancelled
				// the worker must check at time of execution if this can be done or not
				r.store.Update(ev.JobId, func(j *Job) error {
					return j.Transition(StateCancelled)
				})

				// remove timeout since the job was cancelled
				// TODO: make this idiomatic later
				if t, ok := timers[ev.JobId]; ok {
					t.Stop()
					delete(timers, ev.JobId)
				}

				ev.Reply <- nil

			case EventTimeout:
				// the worker will pick it up from queue and exit instantly
				if job, err := r.store.Get(ev.JobId); err == nil {
					r.store.Update(job.Id, func(j *Job) error {
						return j.Transition(StateTimeout)
					})
				}

			case EventShutdown:
				// TODO: add gracefull shutdown later

			case EventResult:

				job, err := r.store.Get(ev.JobId)

				if err != nil {
					// send reply to sender
				}

				if ev.Result.Err != nil && job.Attempts < job.Options.MaxRetries {
					r.store.Update(job.Id, func(j *Job) error {
						j.Attempts++
						return nil
					})
					r.queue.Enqueue(job)
				} else if ev.Result.Err == nil {
					r.store.Update(job.Id, func(j *Job) error {
						return j.Transition(StateSuccess)
					})
				} else {
					r.store.Update(job.Id, func(j *Job) error {
						return j.Transition(StateFailed)
					})
				}
			}

		// this adds buffer , but it removes circular depdency from the worker
		case res := <-r.worker.Results:
			r.events <- &Event{Type: EventResult, Result: res, JobId: res.JobID}
		}

	}
}
