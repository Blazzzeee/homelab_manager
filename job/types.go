package job

import (
	"context"
	"fmt"
	"time"
)

type ID string

type JobState int

const (
	StateCreated JobState = iota
	StateQueued
	StateRunning
	StateSuccess
	StateCancelled
	StateTimeout
	StateFailed
)

var allowedTransitions = map[JobState]map[JobState]struct{}{
	StateCreated: {
		StateQueued: {},
		StateFailed: {},
	},
	StateQueued: {
		StateRunning:   {},
		StateFailed:    {},
		StateCancelled: {},
	},
	StateRunning: {
		StateSuccess: {},
		StateTimeout: {},
		StateFailed:  {},
	},
	StateSuccess:   {},
	StateCancelled: {},
	StateTimeout:   {},
	StateFailed:    {},
}

func (j *Job) Transition(to JobState) error {
	now := time.Now()

	next := allowedTransitions[j.State]

	// if the requested transition can be done after 'next'
	if _, ok := next[to]; !ok {
		return fmt.Errorf("invalid transition %v to %v", j.State, to)
	}

	switch to {
	case StateQueued:
		j.TimeStamps.QueuedAt = &now
	case StateRunning:
		j.TimeStamps.StartedAt = &now
	case StateSuccess, StateFailed, StateCancelled, StateTimeout:
		j.TimeStamps.FinishedAt = &now
	}
	j.State = to

	return nil
}

// TODO: enforce strict FSM transition
// func (j *Job) Transition(to JobState) error

type JobTimeInfo struct {
	CreatedAt  time.Time
	QueuedAt   *time.Time
	StartedAt  *time.Time
	FinishedAt *time.Time
}

type Options struct {
	Timeout     time.Duration
	Cancellable bool
	MaxRetries  int
}

// TODO: maybe This should be a opaque struct to outside
type Job struct {
	Id      ID
	Cmd     []string
	Options Options
	// should be a ref in future
	// logs string
	State      JobState
	TimeStamps JobTimeInfo
	Attempts   int

	ExitCode *int
	Error    error
}

// TOOD: move this to consumer
// This component is responsible for decision making
type Runner interface {
	Submit(ctx context.Context, cmd []string, options Options) (ID, error)
	Get(id ID) (*Job, error)
	Cancel(ctx context.Context, id ID) error
	List(ctx context.Context) []*Job
	Shutdown(ctx context.Context) error
}

// Just takes a job and executes it blindly
// the context should provide other things like cancellation
type Executor interface {
	// gives back process id that will form an association with the job itself
	Dispatch(ctx context.Context, job *Job) <-chan WorkerResult
}
