package job

import (
	"runtime"
	"testing"
	"time"
)

func TestQueue_NewInvalidSize(t *testing.T) {
	if _, err := NewQueue(0); err == nil {
		t.Fatal("expected error for size 0")
	}
	if _, err := NewQueue(-1); err == nil {
		t.Fatal("expected error for negative size")
	}
}

func TestQueue_EnqueueDequeue(t *testing.T) {
	q, err := NewQueue(2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	job := &Job{Id: ID("one")}
	if err := q.Enqueue(job); err != nil {
		t.Fatalf("unexpected enqueue error: %v", err)
	}

	got, err := q.Dequeue()
	if err != nil {
		t.Fatalf("unexpected dequeue error: %v", err)
	}
	if got != job {
		t.Fatal("expected same job pointer from dequeue")
	}
}

func TestQueue_Full(t *testing.T) {
	q, err := NewQueue(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := q.Enqueue(&Job{Id: ID("a")}); err != nil {
		t.Fatalf("unexpected enqueue error: %v", err)
	}
	if err := q.Enqueue(&Job{Id: ID("b")}); err != ErrQueueFull {
		t.Fatalf("expected ErrQueueFull, got %v", err)
	}
}

func TestQueue_EmptyOnClosedBuffer(t *testing.T) {
	q, err := NewQueue(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	q.Close()

	if _, err := q.Dequeue(); err != ErrQueueEmpty {
		t.Fatalf("expected ErrQueueEmpty, got %v", err)
	}
}

func TestQueue_FifoOrder(t *testing.T) {
	q, err := NewQueue(3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	jobA := &Job{Id: ID("a")}
	jobB := &Job{Id: ID("b")}
	jobC := &Job{Id: ID("c")}

	if err := q.Enqueue(jobA); err != nil {
		t.Fatalf("unexpected enqueue error: %v", err)
	}
	if err := q.Enqueue(jobB); err != nil {
		t.Fatalf("unexpected enqueue error: %v", err)
	}
	if err := q.Enqueue(jobC); err != nil {
		t.Fatalf("unexpected enqueue error: %v", err)
	}

	got, err := q.Dequeue()
	if err != nil || got != jobA {
		t.Fatalf("expected jobA first, got %v, err %v", got, err)
	}
	got, err = q.Dequeue()
	if err != nil || got != jobB {
		t.Fatalf("expected jobB second, got %v, err %v", got, err)
	}
	got, err = q.Dequeue()
	if err != nil || got != jobC {
		t.Fatalf("expected jobC third, got %v, err %v", got, err)
	}
}

func TestQueue_ConcurrentEnqueueDequeue(t *testing.T) {
	const (
		totalJobs = 1000
		queueSize = 10
		timeout   = 2 * time.Second
	)

	q, err := NewQueue(queueSize)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

		enqueued := make(chan struct{}, totalJobs)
		dequeued := make(chan struct{}, totalJobs)
		doneEnq := make(chan struct{})
		doneDeq := make(chan struct{})

		go func() {
			for i := 0; i < totalJobs; i++ {
				job := &Job{Id: ID("job")}
				for {
					err := q.Enqueue(job)
					if err == nil {
						enqueued <- struct{}{}
						break
					}
					if err == ErrQueueFull {
						runtime.Gosched()
						continue
					}
					t.Errorf("unexpected enqueue error: %v", err)
					return
				}
			}
			q.Close()
			close(doneEnq)
		}()

		go func() {
			for {
				job, err := q.Dequeue()
				if err == ErrQueueEmpty {
					close(doneDeq)
					return
				}
				if err != nil {
					t.Errorf("unexpected dequeue error: %v", err)
					close(doneDeq)
					return
				}
				if job == nil {
					t.Errorf("unexpected nil job")
					close(doneDeq)
					return
				}
				dequeued <- struct{}{}
			}
		}()

		select {
		case <-doneEnq:
		case <-time.After(timeout):
			t.Fatal("timeout waiting for enqueue completion")
		}

		select {
		case <-doneDeq:
		case <-time.After(timeout):
			t.Fatal("timeout waiting for dequeue completion")
		}

	if len(enqueued) != totalJobs {
		t.Fatalf("expected %d enqueued, got %d", totalJobs, len(enqueued))
	}
	if len(dequeued) != totalJobs {
		t.Fatalf("expected %d dequeued, got %d", totalJobs, len(dequeued))
	}
}
