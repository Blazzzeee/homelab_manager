package job

import (
	"context"
	"time"
)

type ID string

type JobState int

const (
	StateQueued JobState = iota
	StateRunning
	StateSuccess
	StateCancelled
	StateFailed
	StateTimeout
)

// TODO: enforce strict FSM transition
// func (j *Job) Transition(to JobState) error

type JobTimeInfo struct {
	CreatedAt  time.Time
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

// This component is responsible for decision making
type Runner interface {
	Submit(ctx context.Context, cmd []string, options Options) (ID, error)
	Get(ctx context.Context, id ID) (*Job, error)
	Cancel(ctx context.Context, id ID) error
	List(ctx context.Context) ([]*Job, error)
	Shutdown(ctx context.Context) error
}

// The JobId will be the key ta the Job itself
type JobStore struct {
	buf map[ID]*Job
}

// Just takes a job and executes it blindly
// the context should provide other things like cancellation
type Executor interface {
	// gives back process id that will form an association with the job itself
	Dispatch(ctx context.Context, job *Job) error
}
