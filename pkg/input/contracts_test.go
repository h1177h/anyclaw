package input

import "testing"

func TestBaseAdapterSetRunningMarksHealthyWithoutError(t *testing.T) {
	adapter := NewBaseAdapter("demo", true)

	adapter.SetRunning(true)

	status := adapter.Status()
	if !status.Running {
		t.Fatalf("expected adapter to be running")
	}
	if !status.Healthy {
		t.Fatalf("expected adapter to be healthy after entering running state")
	}
}

func TestBaseAdapterSetRunningDoesNotClearExistingError(t *testing.T) {
	adapter := NewBaseAdapter("demo", true)
	adapter.SetError(assertErr("boom"))

	adapter.SetRunning(true)

	status := adapter.Status()
	if status.Healthy {
		t.Fatalf("expected adapter with last error to remain unhealthy")
	}
	if status.LastError != "boom" {
		t.Fatalf("expected last error to be preserved, got %q", status.LastError)
	}
}

type assertErr string

func (e assertErr) Error() string { return string(e) }
