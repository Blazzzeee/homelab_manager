package job

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"
)

const outputString = "Hello World"

func MakeJob() *Job {
	return &Job{Id: ID("one"), Cmd: []string{"echo", outputString}}
}

func WorkerResultValidate(
	t *testing.T,
	job *Job,
	resultChannel <-chan *WorkerResult,
	after func(*WorkerResult),
	expectErr bool,
) {
	t.Helper()

	select {
	case res := <-resultChannel:
		if res.JobID != job.Id {
			t.Fatalf("expected jobid %s got %s ", job.Id, res.JobID)
		}

		if expectErr {
			if res.Err == nil {
				t.Fatal("expected error, got nil")
			}
		} else if res.Err != nil {
			t.Fatal("TestDisptach: Err in res: ", res.Err)
		}

		if after != nil {
			after(res)
		}

	case <-time.After(2 * time.Second):
		t.Fatal("dispatch timed out")
	}
}

// Tests
// TestInit
func TestWorkerPoolInit(t *testing.T) {
	pool := NewWorkerPool(10)
	pool.Start()
	defer pool.Shutdown()

	if pool == nil {
		t.Fatal("expected non-nil pool")
	}

	if pool.jobs == nil || pool.Results == nil {
		t.Fatal("channel init failed")
	}

	pool2 := NewWorkerPool(-2)
	pool3 := NewWorkerPool(0)
	pool2.Start()
	defer pool2.Shutdown()
	pool3.Start()
	defer pool3.Shutdown()

	if pool2 == nil || pool3 == nil {
		t.Fatal("input sanitization failed")
	}

	// make sure negative workerPoolSize works
	job := MakeJob()
	pool2.Dispatch(context.TODO(), job)

	WorkerResultValidate(t, job, pool2.Results, nil, false)
}

// TestShutdown
func TestWorkerShutdown(t *testing.T) {
	pool := NewWorkerPool(10)

	pool.Start()

	// Assume 1000s of jobs ran , and now we are doing shutdown
	pool.Shutdown()
}

// TestDispatch returns output in results channel
func TestDispatch(t *testing.T) {
	pool := NewWorkerPool(1)
	pool.Start()
	defer pool.Shutdown()

	job := MakeJob()
	pool.Dispatch(context.TODO(), job)

	WorkerResultValidate(t, job, pool.Results, nil, false)
}

func TestExecCapturesStdout(t *testing.T) {
	pool := NewWorkerPool(1)
	pool.Start()
	defer pool.Shutdown()

	job := MakeJob()
	pool.Dispatch(context.TODO(), job)

	WorkerResultValidate(t, job, pool.Results, func(res *WorkerResult) {

		if len(res.Output) == 0 {
			t.Fatalf("Exec: output empty")
		}

		if strings.Compare(strings.TrimSpace(outputString), res.Output) == 0 {
			t.Fatalf("Exec: Error in output")
		}
	}, false)
}

func TestDispatchFailureCapturesStderrAndExitCode(t *testing.T) {
	pool := NewWorkerPool(1)
	pool.Start()
	defer pool.Shutdown()

	job := &Job{
		Id:  ID("fail"),
		Cmd: []string{"sh", "-c", "echo failure 1>&2; exit 3"},
	}
	pool.Dispatch(context.TODO(), job)

	WorkerResultValidate(t, job, pool.Results, func(res *WorkerResult) {
		exitErr, ok := res.Err.(*exec.ExitError)
		if !ok {
			t.Fatalf("expected *exec.ExitError, got %T", res.Err)
		}
		if exitErr.ExitCode() != 3 {
			t.Fatalf("expected exit code 3, got %d", exitErr.ExitCode())
		}
		if !strings.Contains(res.Output, "failure") {
			t.Fatalf("expected stderr in output, got %q", res.Output)
		}
	}, true)
}

// Test concurrent dispatch works
func TestWorkerConcurrency(t *testing.T) {

	pool := NewWorkerPool(10)
	pool.Start()
	defer pool.Shutdown()

	total := 20
	for i := range total {

		job := MakeJob()
		job.Id = ID(fmt.Sprint(i))
		pool.Dispatch(context.TODO(), job)
	}

	for range total {
		select {
		case <-pool.Results:
		case <-time.After(2 * time.Second):
			t.Fatal("timeout waiting for results")
		}
	}
}
