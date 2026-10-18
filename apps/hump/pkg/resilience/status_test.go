package resilience

import (
	"context"
	"errors"
	"testing"
)

type recordingStatusHooks struct {
	success int
	failure int
}

func (h *recordingStatusHooks) OnSuccess() { h.success++ }
func (h *recordingStatusHooks) OnFailure() { h.failure++ }

func TestWithStatus_nilHooks(t *testing.T) {
	wantErr := errors.New("boom")

	tests := map[string]error{
		"success": nil,
		"failure": wantErr,
	}

	for name, want := range tests {
		t.Run(name, func(t *testing.T) {
			h := WithStatus(func(context.Context, int) error { return want }, nil)

			if err := h(context.Background(), 1); !errors.Is(err, want) {
				t.Fatalf("got error %v, want %v", err, want)
			}
		})
	}
}

func TestWithStatus_hooksCalled(t *testing.T) {
	wantErr := errors.New("boom")

	tests := map[string]struct {
		err                      error
		wantSuccess, wantFailure int
	}{
		"success": {err: nil, wantSuccess: 1, wantFailure: 0},
		"failure": {err: wantErr, wantSuccess: 0, wantFailure: 1},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			hooks := &recordingStatusHooks{}
			h := WithStatus(func(context.Context, int) error { return tt.err }, hooks)

			if err := h(context.Background(), 1); !errors.Is(err, tt.err) {
				t.Fatalf("got error %v, want %v", err, tt.err)
			}

			if hooks.success != tt.wantSuccess {
				t.Errorf("got %d OnSuccess calls, want %d", hooks.success, tt.wantSuccess)
			}

			if hooks.failure != tt.wantFailure {
				t.Errorf("got %d OnFailure calls, want %d", hooks.failure, tt.wantFailure)
			}
		})
	}
}
