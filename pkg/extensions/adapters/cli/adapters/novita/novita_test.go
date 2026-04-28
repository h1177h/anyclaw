package novita

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewReadsAPIKeyFromEnvironment(t *testing.T) {
	t.Setenv("NOVITA_API_KEY", "env-key")

	client := New(Config{})

	if client.apiKey != "env-key" {
		t.Fatalf("expected API key from NOVITA_API_KEY, got %q", client.apiKey)
	}
}

func TestModelsRequiresAPIKeyBeforeRequest(t *testing.T) {
	t.Setenv("NOVITA_API_KEY", "")

	var called bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()

	client := New(Config{})
	client.baseURL = server.URL
	client.httpClient = server.Client()

	_, err := client.Execute(context.Background(), []string{"models"})
	if err == nil || !strings.Contains(err.Error(), "NOVITA_API_KEY") {
		t.Fatalf("expected missing NOVITA_API_KEY error, got %v", err)
	}
	if called {
		t.Fatal("expected no remote request without API key")
	}
}
