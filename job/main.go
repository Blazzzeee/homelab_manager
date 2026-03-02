package job

import (
	"context"
	"time"
)

type JobIdType int

type JobState int

const (
	QUEUED = iota
	RUNNING
	SUCCESS
	CANCELED
	FAILED
)

// TODO: enforce FSM transition

type JobTimeInfo struct {
	CreatedAt  time.Time
	StartedAt  time.Time
	FinishedAt time.Time
}

type Job struct {
	JobId   JobIdType
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

type Options struct {
	Timeout     time.Duration
	Cancellable bool
	MaxRetries  int
}

type Runner interface {
	Submit(ctx context.Context, cmd []string, options Options) (JobIdType, error)
	Get(ctx context.Context, job_id JobIdType) (*Job, error)
	Cancel(ctx context.Context, job_id JobIdType) error
	List(ctx context.Context) ([]*Job, error)
	Shutdown(ctx context.Context) error
}
