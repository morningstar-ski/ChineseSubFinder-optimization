package downloader

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestRunDownloaderErrorStepReturnsError(t *testing.T) {
	want := errors.New("boom")
	err, panicked, canceled := runDownloaderErrorStep(context.Background(), func() error {
		return want
	})
	if panicked != nil {
		t.Fatalf("unexpected panic = %#v", panicked)
	}
	if canceled {
		t.Fatal("unexpected canceled result")
	}
	if !errors.Is(err, want) {
		t.Fatalf("error = %v, want %v", err, want)
	}
}

func TestRunDownloaderErrorStepReturnsPanic(t *testing.T) {
	err, panicked, canceled := runDownloaderErrorStep(context.Background(), func() error {
		panic("panic-value")
	})
	if err != nil {
		t.Fatalf("unexpected error = %v", err)
	}
	if canceled {
		t.Fatal("unexpected canceled result")
	}
	panicText, ok := panicked.(string)
	if ok == false {
		t.Fatalf("panic type = %T, want string", panicked)
	}
	if strings.Contains(panicText, "panic-value") == false {
		t.Fatalf("panic = %#v", panicked)
	}
}
