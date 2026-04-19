package channels

import (
	"errors"
	"testing"
)

func TestStreamWithMessageFallbackHidesInternalError(t *testing.T) {
	var final string
	err := streamWithMessageFallback(func(onChunk func(chunk string)) error {
		onChunk("partial response")
		return errors.New("backend exploded: token=secret")
	}, func(text string) error {
		final = text
		return nil
	})

	if err == nil {
		t.Fatalf("expected original stream error to be returned")
	}
	if final != "partial response\n\n"+streamFallbackErrorNotice {
		t.Fatalf("unexpected fallback message: %q", final)
	}
}

func TestStreamWithMessageFallbackSkipsFinalSendWithoutChunks(t *testing.T) {
	called := false
	err := streamWithMessageFallback(func(onChunk func(chunk string)) error {
		return errors.New("backend exploded")
	}, func(text string) error {
		called = true
		return nil
	})

	if err == nil {
		t.Fatalf("expected original stream error to be returned")
	}
	if called {
		t.Fatalf("expected no final message when no chunks were produced")
	}
}
