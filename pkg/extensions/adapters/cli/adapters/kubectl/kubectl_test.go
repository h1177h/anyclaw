package kubectl

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
	if os.Getenv("ANYCLAW_KUBECTL_ADAPTER_HELPER") == "1" {
		if logPath := os.Getenv("ANYCLAW_KUBECTL_ADAPTER_LOG"); logPath != "" {
			data, _ := json.Marshal(os.Args[1:])
			_ = os.WriteFile(logPath, data, 0o644)
		}
		fmt.Print(strings.Join(os.Args[1:], " "))
		os.Exit(0)
	}

	os.Exit(m.Run())
}

func TestGetConsumesKnownResourceType(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "kubectl-args.json")
	t.Setenv("ANYCLAW_KUBECTL_ADAPTER_HELPER", "1")
	t.Setenv("ANYCLAW_KUBECTL_ADAPTER_LOG", logPath)
	client := NewClient(Config{KubectlPath: fakeKubectlPath(t), Namespace: "default"})

	if _, err := client.Run(context.Background(), []string{"get", "services"}); err != nil {
		t.Fatalf("Run get returned error: %v", err)
	}

	want := []string{"get", "services", "-n", "default", "-o", "wide"}
	if got := readKubectlArgs(t, logPath); !reflect.DeepEqual(got, want) {
		t.Fatalf("kubectl args = %#v, want %#v", got, want)
	}
}

func fakeKubectlPath(t *testing.T) string {
	t.Helper()

	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}

	path := filepath.Join(t.TempDir(), "kubectl-helper.exe")
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

func readKubectlArgs(t *testing.T, logPath string) []string {
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
