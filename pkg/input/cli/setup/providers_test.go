package setup

import "testing"

func TestCanonicalProviderRecognizesAliases(t *testing.T) {
	tests := map[string]string{
		"":           "openai",
		"claude":     "anthropic",
		"dashscope":  "qwen",
		"ollama":     "ollama",
		"compatible": "compatible",
	}

	for input, want := range tests {
		if got := CanonicalProvider(input); got != want {
			t.Fatalf("CanonicalProvider(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestProviderNeedsAPIKey(t *testing.T) {
	if ProviderNeedsAPIKey("ollama") {
		t.Fatal("expected ollama to skip API key requirement")
	}
	if !ProviderNeedsAPIKey("openai") {
		t.Fatal("expected openai to require API key")
	}
}
