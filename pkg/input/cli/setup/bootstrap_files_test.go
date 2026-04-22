package setup

import (
	"path/filepath"
	"testing"
)

func TestEnsureAndLoadBootstrapFiles(t *testing.T) {
	dir := t.TempDir()

	err := ensureBootstrapFiles(dir, bootstrapSeed{
		AgentName:        "AnyClaw",
		AgentDescription: "CLI onboarding",
		UserProfile:      "Default language: zh-CN",
		WorkspaceFocus:   "Coding",
		AssistantStyle:   "Direct",
		Constraints:      "Do not delete files without approval.",
	})
	if err != nil {
		t.Fatalf("ensureBootstrapFiles returned error: %v", err)
	}

	files, err := loadBootstrapFiles(dir, 128)
	if err != nil {
		t.Fatalf("loadBootstrapFiles returned error: %v", err)
	}
	if len(files) != len(bootstrapFileOrder) {
		t.Fatalf("expected %d bootstrap files, got %d", len(bootstrapFileOrder), len(files))
	}

	for _, name := range []string{"AGENTS.md", "IDENTITY.md", "MEMORY.md"} {
		if !fileExists(filepath.Join(dir, name)) {
			t.Fatalf("expected %s to exist", name)
		}
	}
	for _, file := range files {
		if file.Missing {
			t.Fatalf("expected bootstrap file %s to be present", file.Name)
		}
	}
}
