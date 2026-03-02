package job

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestJobStore_NewEmpty(t *testing.T) {
	store := New()
	if store == nil {
		t.Fatal("expected non-nil store")
	}
	if len(store.buf) != 0 {
		t.Fatalf("expected empty store, got %d", len(store.buf))
	}
}

func TestJobStore_PushNil(t *testing.T) {
	store := New()
	if err := store.Push(nil); err != ErrJobNull {
		t.Fatalf("expected ErrJobNull, got %v", err)
	}
}

func TestJobStore_PushDuplicate(t *testing.T) {
	store := New()
	job := &Job{Id: ID("dup")}
	if err := store.Push(job); err != nil {
		t.Fatalf("unexpected error pushing job: %v", err)
	}
	if err := store.Push(job); err != ErrJobAlreadyExists {
		t.Fatalf("expected ErrJobAlreadyExists, got %v", err)
	}
}

func TestJobStore_GetMissing(t *testing.T) {
	store := New()
	if _, err := store.Get(ID("missing")); err != ErrJobNotFound {
		t.Fatalf("expected ErrJobNotFound, got %v", err)
	}
}

func TestJobStore_GetReturnsCopy(t *testing.T) {
	store := New()
	now := time.Now()
	job := &Job{
		Id:    ID("copy"),
		Cmd:   []string{"echo", "hello"},
		State: StateQueued,
		TimeStamps: JobTimeInfo{
			CreatedAt: now,
		},
	}

	if err := store.Push(job); err != nil {
		t.Fatalf("unexpected error pushing job: %v", err)
	}

	got, err := store.Get(job.Id)
	if err != nil {
		t.Fatalf("unexpected error getting job: %v", err)
	}

	if got == job {
		t.Fatal("expected Get to return a copy, got same pointer")
	}

	job.State = StateRunning
	got, err = store.Get(job.Id)
	if err != nil {
		t.Fatalf("unexpected error getting job after state change: %v", err)
	}
	if got.State != StateRunning {
		t.Fatalf("expected state %v, got %v", StateRunning, got.State)
	}
	got.State = StateSuccess
	if store.buf[job.Id].State == StateSuccess {
		t.Fatal("mutation to copy should not affect stored job state")
	}

	if len(got.Cmd) == 0 {
		t.Fatal("expected non-empty cmd")
	}
}

func TestJobStore_PopRemovesAndReturns(t *testing.T) {
	store := New()
	job := &Job{Id: ID("pop")}
	if err := store.Push(job); err != nil {
		t.Fatalf("unexpected error pushing job: %v", err)
	}

	got, err := store.Pop(job.Id)
	if err != nil {
		t.Fatalf("unexpected error popping job: %v", err)
	}
	if got != job {
		t.Fatal("expected Pop to return stored job pointer")
	}

	if _, err := store.Get(job.Id); err != ErrJobNotFound {
		t.Fatalf("expected ErrJobNotFound after Pop, got %v", err)
	}
}

func TestJobStore_ConcurrentAccess(t *testing.T) {
	store := New()
	const n = 200

	errs := make(chan error, n)
	var wg sync.WaitGroup
	wg.Add(n)

	for i := range n {
		go func(i int) {
			defer wg.Done()
			id := ID(fmt.Sprintf("job-%d", i))
			job := &Job{Id: id}
			if err := store.Push(job); err != nil {
				errs <- err
				return
			}

			if _, err := store.Get(id); err != nil {
				errs <- err
				return
			}

			if _, err := store.Pop(id); err != nil {
				errs <- err
				return
			}

			errs <- nil
		}(i)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("unexpected error in concurrent access: %v", err)
		}
	}
}
