package schedule

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"
)

type stubExecutor struct {
	mu      sync.Mutex
	output  string
	err     error
	blockCh chan struct{}
	started chan struct{}
	calls   int
}

func (s *stubExecutor) Execute(ctx context.Context, cmd string, input map[string]any) (string, error) {
	s.mu.Lock()
	s.calls++
	blockCh := s.blockCh
	output := s.output
	err := s.err
	started := s.started
	s.mu.Unlock()

	if started != nil {
		select {
		case <-started:
		default:
			close(started)
		}
	}

	if blockCh != nil {
		select {
		case <-blockCh:
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	return output, err
}

type failingPersister struct {
	saveTasksErr error
	saveRunsErr  error
}

func (p *failingPersister) SaveTasks(tasks []*Task) error  { return p.saveTasksErr }
func (p *failingPersister) LoadTasks() ([]*Task, error)    { return nil, nil }
func (p *failingPersister) SaveRuns(runs []*TaskRun) error { return p.saveRunsErr }
func (p *failingPersister) LoadRuns() ([]*TaskRun, error)  { return nil, nil }

func TestSchedulerAddTaskAndCopies(t *testing.T) {
	scheduler := New()
	taskID, err := scheduler.AddTask(&Task{
		Name:     "hourly",
		Schedule: "0 * * * *",
		Command:  "echo hi",
		Input:    map[string]any{"k": "v"},
		Enabled:  true,
	})
	if err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	task, ok := scheduler.GetTask(taskID)
	if !ok {
		t.Fatalf("GetTask(%s) returned not found", taskID)
	}
	task.Input["k"] = "mutated"

	again, ok := scheduler.GetTask(taskID)
	if !ok {
		t.Fatalf("GetTask(%s) returned not found on second lookup", taskID)
	}
	if got := again.Input["k"]; got != "v" {
		t.Fatalf("expected defensive copy, got %v", got)
	}

	listed := scheduler.ListTasks()
	listed[0].Name = "changed"
	again, _ = scheduler.GetTask(taskID)
	if again.Name != "hourly" {
		t.Fatalf("expected list copy to be isolated, got %q", again.Name)
	}
}

func TestSchedulerRunTaskNowAndCancel(t *testing.T) {
	executor := &stubExecutor{blockCh: make(chan struct{})}
	scheduler := NewScheduler(executor)

	taskID, err := scheduler.AddTask(&Task{
		Name:     "blocking",
		Schedule: "@every 1m",
		Command:  "sleep",
		Timeout:  1,
		Enabled:  true,
	})
	if err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	if err := scheduler.RunTaskNow(taskID); err != nil {
		t.Fatalf("RunTaskNow failed: %v", err)
	}

	var runID string
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		runs := scheduler.GetTaskRuns(taskID)
		if len(runs) > 0 {
			runID = runs[0].ID
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if runID == "" {
		t.Fatal("expected run to be recorded")
	}

	if err := scheduler.CancelRun(runID); err != nil {
		t.Fatalf("CancelRun failed: %v", err)
	}
	close(executor.blockCh)

	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		runs := scheduler.GetTaskRuns(taskID)
		if len(runs) > 0 && runs[0].Status == "cancelled" {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("expected run to become cancelled")
}

func TestSchedulerStartStopRestart(t *testing.T) {
	scheduler := New()
	if err := scheduler.Start(); err != nil {
		t.Fatalf("first Start failed: %v", err)
	}
	scheduler.Stop()
	if err := scheduler.Start(); err != nil {
		t.Fatalf("second Start failed: %v", err)
	}
	scheduler.Stop()
}

func TestSchedulerMarshalJSON(t *testing.T) {
	scheduler := New()
	if _, err := scheduler.AddTask(&Task{
		Name:     "json",
		Schedule: "@hourly",
		Command:  "echo",
		Enabled:  true,
	}); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	data, err := json.Marshal(scheduler)
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty JSON payload")
	}
}

func TestSchedulerLoadPersisted(t *testing.T) {
	persister, err := NewFilePersister(t.TempDir())
	if err != nil {
		t.Fatalf("NewFilePersister failed: %v", err)
	}

	savedNext := time.Date(2026, 4, 22, 11, 0, 0, 0, time.UTC)
	if err := persister.SaveTasks([]*Task{{
		ID:       "task-1",
		Name:     "persisted",
		Schedule: "0 * * * *",
		Command:  "echo",
		Enabled:  true,
		NextRun:  &savedNext,
	}}); err != nil {
		t.Fatalf("SaveTasks failed: %v", err)
	}
	if err := persister.SaveRuns([]*TaskRun{{
		ID:        "run-1",
		TaskID:    "task-1",
		StartTime: time.Now().UTC(),
		Status:    "success",
	}}); err != nil {
		t.Fatalf("SaveRuns failed: %v", err)
	}

	scheduler := New()
	scheduler.SetPersister(persister)
	if err := scheduler.LoadPersisted(); err != nil {
		t.Fatalf("LoadPersisted failed: %v", err)
	}

	tasks := scheduler.ListTasks()
	if len(tasks) != 1 || tasks[0].ID != "task-1" {
		t.Fatalf("unexpected tasks after load: %+v", tasks)
	}
	runs := scheduler.GetTaskRuns("task-1")
	if len(runs) != 1 || runs[0].ID != "run-1" {
		t.Fatalf("unexpected runs after load: %+v", runs)
	}
}

func TestSchedulerRetryAndStats(t *testing.T) {
	executor := &stubExecutor{err: errors.New("boom")}
	scheduler := NewScheduler(executor)

	taskID, err := scheduler.AddTask(&Task{
		Name:         "retrying",
		Schedule:     "@every 1m",
		Command:      "echo",
		MaxRetries:   1,
		RetryBackoff: "linear",
		Timeout:      5,
		Enabled:      true,
	})
	if err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	if err := scheduler.RunTaskNow(taskID); err != nil {
		t.Fatalf("RunTaskNow failed: %v", err)
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		runs := scheduler.GetTaskRuns(taskID)
		if len(runs) > 0 && runs[0].Status == "failed" {
			stats := scheduler.Stats()
			if stats["failed_runs"] != 1 {
				t.Fatalf("expected failed_runs=1, got %+v", stats)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("expected run to fail")
}

func TestSchedulerUpdateTaskCanDisableTask(t *testing.T) {
	scheduler := New()
	taskID, err := scheduler.AddTask(&Task{
		Name:     "toggle",
		Schedule: "@hourly",
		Command:  "echo",
		Enabled:  true,
	})
	if err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	if err := scheduler.UpdateTask(&Task{
		ID:       taskID,
		Name:     "toggle",
		Schedule: "@hourly",
		Command:  "echo",
		Enabled:  false,
	}); err != nil {
		t.Fatalf("UpdateTask failed: %v", err)
	}

	task, ok := scheduler.GetTask(taskID)
	if !ok {
		t.Fatalf("GetTask(%s) returned not found", taskID)
	}
	if task.Enabled {
		t.Fatal("expected task to be disabled after update")
	}
	if task.NextRun != nil {
		t.Fatalf("expected disabled task to have nil next run, got %v", *task.NextRun)
	}
}

func TestSchedulerDeleteTaskCancelsActiveRun(t *testing.T) {
	executor := &stubExecutor{
		blockCh: make(chan struct{}),
		started: make(chan struct{}),
	}
	scheduler := NewScheduler(executor)

	taskID, err := scheduler.AddTask(&Task{
		Name:     "delete-running",
		Schedule: "@every 1m",
		Command:  "sleep",
		Timeout:  5,
		Enabled:  true,
	})
	if err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}
	if err := scheduler.RunTaskNow(taskID); err != nil {
		t.Fatalf("RunTaskNow failed: %v", err)
	}

	select {
	case <-executor.started:
	case <-time.After(2 * time.Second):
		t.Fatal("expected executor to start")
	}

	if err := scheduler.DeleteTask(taskID); err != nil {
		t.Fatalf("DeleteTask failed: %v", err)
	}

	if _, ok := scheduler.GetTask(taskID); ok {
		t.Fatal("expected task to be deleted")
	}
	if runs := scheduler.GetTaskRuns(taskID); len(runs) != 0 {
		t.Fatalf("expected deleted task runs to be cleared, got %+v", runs)
	}
}

func TestSchedulerRecordsPersistenceErrorFromRunSnapshots(t *testing.T) {
	persister := &failingPersister{saveRunsErr: errors.New("disk full")}
	scheduler := NewScheduler(nil)
	scheduler.SetPersister(persister)

	taskID, err := scheduler.AddTask(&Task{
		Name:     "persist-error",
		Schedule: "@every 1m",
		Command:  "echo",
		Enabled:  true,
	})
	if err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}
	if err := scheduler.RunTaskNow(taskID); err != nil {
		t.Fatalf("RunTaskNow failed: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		runs := scheduler.GetTaskRuns(taskID)
		if len(runs) > 0 && runs[0].Status == "success" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if err := scheduler.LastPersistenceError(); err == nil || err.Error() != "disk full" {
		t.Fatalf("expected last persistence error to be recorded, got %v", err)
	}
}
