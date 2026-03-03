package job

import (
	"errors"
	"fmt"
)

// Errors
var (
	ErrQueueFull  = errors.New("queue full")
	ErrQueueEmpty = errors.New("queue empty")
	ErrClosed     = errors.New("queue closed")
)

type Queue struct {
	buf  chan *Job
	done chan struct{}
}

// TODO: maybe add context later
func NewQueue(size int) (*Queue, error) {
	if size <= 0 {
		return nil, fmt.Errorf("queue size cannot be negative")
	}

	queue := &Queue{
		buf:  make(chan *Job, size),
		done: make(chan struct{}),
	}

	return queue, nil
}

func (q *Queue) Enqueue(job *Job) error {

	select {
	case <-q.done:
		return ErrClosed
	case q.buf <- job:
		return nil
	default:
		return ErrQueueFull
	}
}

func (q *Queue) Dequeue() (*Job, error) {
	select {
	case <-q.done:
		return nil, ErrClosed
	case job, ok := <-q.buf:
		if !ok {
			return nil, ErrQueueEmpty
		}
		return job, nil
	}

}

func (q *Queue) Close() {
	close(q.done)
}
