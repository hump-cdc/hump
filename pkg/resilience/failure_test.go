package resilience

import (
	"context"
	"errors"
	"testing"
)

func TestWithFailureMode_crashPropagates(t *testing.T) {
	wantErr := errors.New("boom")

	for _, mode := range []FailureMode{FailureModeCrash, FailureMode("")} {
		t.Run(string(mode), func(t *testing.T) {
			hooks := &recordingSkipHooks{}
			h := WithFailureMode(func(context.Context, int) error { return wantErr }, mode, hooks)

			if err := h(context.Background(), 1); !errors.Is(err, wantErr) {
				t.Fatalf("got error %v, want %v", err, wantErr)
			}

			if hooks.calls != 0 {
				t.Errorf("got %d OnSkip calls, want 0", hooks.calls)
			}
		})
	}
}

func TestWithFailureMode_skipSwallows(t *testing.T) {
	wantErr := errors.New("boom")

	tests := map[string]struct {
		err       error
		wantCalls int
	}{
		"success": {err: nil, wantCalls: 0},
		"failure": {err: wantErr, wantCalls: 1},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			hooks := &recordingSkipHooks{}
			h := WithFailureMode(func(context.Context, int) error { return tt.err }, FailureModeSkip, hooks)

			if err := h(context.Background(), 1); err != nil {
				t.Fatalf("got error %v, want nil", err)
			}

			if hooks.calls != tt.wantCalls {
				t.Fatalf("got %d OnSkip calls, want %d", hooks.calls, tt.wantCalls)
			}

			if tt.wantCalls > 0 && !errors.Is(hooks.err, tt.err) {
				t.Errorf("got skipped error %v, want %v", hooks.err, tt.err)
			}
		})
	}
}

func TestWithFailureMode_skipPropagatesCancelledContext(t *testing.T) {
	wantErr := errors.New("boom")
	hooks := &recordingSkipHooks{}
	h := WithFailureMode(func(context.Context, int) error { return wantErr }, FailureModeSkip, hooks)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := h(ctx, 1); !errors.Is(err, wantErr) {
		t.Fatalf("got error %v, want %v", err, wantErr)
	}

	if hooks.calls != 0 {
		t.Errorf("got %d OnSkip calls, want 0", hooks.calls)
	}
}

func TestWithFailureMode_nilHooks(t *testing.T) {
	h := WithFailureMode(func(context.Context, int) error { return errors.New("boom") }, FailureModeSkip, nil)

	if err := h(context.Background(), 1); err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
}

type recordingSkipHooks struct {
	calls int
	err   error
}

func (h *recordingSkipHooks) OnSkip(err error) {
	h.calls++
	h.err = err
}
