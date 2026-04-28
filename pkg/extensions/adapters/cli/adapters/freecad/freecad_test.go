package freecad

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	if os.Getenv("ANYCLAW_FREECAD_ADAPTER_HELPER") == "1" {
		if logPath := os.Getenv("ANYCLAW_FREECAD_ADAPTER_LOG"); logPath != "" {
			data, _ := json.Marshal(os.Args[1:])
			_ = os.WriteFile(logPath, data, 0o644)
		}
		fmt.Print(strings.Join(os.Args[1:], " "))
		os.Exit(0)
	}

	os.Exit(m.Run())
}

func TestOpenPassesAllFilesToFreeCAD(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "first.FCStd")
	second := filepath.Join(dir, "second.FCStd")
	writeFile(t, first)
	writeFile(t, second)
	logPath := filepath.Join(dir, "freecad-args.json")
	t.Setenv("ANYCLAW_FREECAD_ADAPTER_HELPER", "1")
	t.Setenv("ANYCLAW_FREECAD_ADAPTER_LOG", logPath)
	adapter := New(Config{FreecadPath: fakeFreeCADPath(t)})

	if _, err := adapter.Execute(context.Background(), []string{"open", first, second}); err != nil {
		t.Fatalf("open returned error: %v", err)
	}

	want := []string{"--background", first, second}
	if got := readFreeCADArgs(t, logPath); !reflect.DeepEqual(got, want) {
		t.Fatalf("freecad args = %#v, want %#v", got, want)
	}
}

func fakeFreeCADPath(t *testing.T) string {
	t.Helper()

	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}

	path := filepath.Join(t.TempDir(), "freecad-helper.exe")
	copyExecutable(t, exe, path)
	return path
}

func copyExecutable(t *testing.T, src, dst string) {
	t.Helper()

	in, err := os.Open(src)
	if err != nil {
		t.Fatalf("open source executable: %v", err)
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		t.Fatalf("create helper executable: %v", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		t.Fatalf("copy helper executable: %v", err)
	}
}

func writeFile(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("model"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

func readFreeCADArgs(t *testing.T, logPath string) []string {
	t.Helper()
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read helper log: %v", err)
	}
	var args []string
	if err := json.Unmarshal(data, &args); err != nil {
		t.Fatalf("decode helper args: %v", err)
	}
	return args
}
