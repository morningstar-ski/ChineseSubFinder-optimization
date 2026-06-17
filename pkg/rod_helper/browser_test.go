package rod_helper

import (
	"fmt"
	"testing"
)

func TestShouldRetryNewPageNavigateOnlyForObjectReferenceChainError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "object chain", err: fmt.Errorf("probe_failed: error value: &cdp.Error{Code:-32000, Message:\"Object reference chain is too long\", Data:\"\"}"), want: true},
		{name: "timeout", err: fmt.Errorf("context deadline exceeded"), want: false},
		{name: "layout", err: fmt.Errorf("download_gate_changed: button changed"), want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldRetryNewPageNavigate(tc.err); got != tc.want {
				t.Fatalf("shouldRetryNewPageNavigate() = %v; want %v", got, tc.want)
			}
		})
	}
}

func TestNormalizeNewPageNavigateErrorCompactsRodStackNoise(t *testing.T) {
	rawErr := fmt.Errorf("error value: &cdp.Error{Code:-32000, Message:\"Object reference chain is too long\", Data:\"\"}\ngoroutine 1 [running]:\nruntime/debug.Stack()")

	got := normalizeNewPageNavigateError(rawErr)
	if got == nil {
		t.Fatal("normalizeNewPageNavigateError() returned nil")
	}
	if got.Error() != "object reference chain is too long" {
		t.Fatalf("normalizeNewPageNavigateError() = %q; want compact transient error", got.Error())
	}
}
