package app

import (
	"context"
	"errors"
	"os"
	"syscall"
	"testing"
	"time"
)

func TestSignalsCancelsWithSignalCause(t *testing.T) {
	tests := []struct {
		name string
		sig  syscall.Signal
		want error
	}{
		{name: "interrupt", sig: syscall.SIGINT, want: errSignalInterrupt},
		{name: "terminate", sig: syscall.SIGTERM, want: errSignalTerminate},
		{name: "other", sig: syscall.SIGUSR1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, release := signals(context.Background(), tt.sig)
			t.Cleanup(release)

			if err := syscall.Kill(os.Getpid(), tt.sig); err != nil {
				t.Fatalf("Kill() error = %v", err)
			}

			select {
			case <-ctx.Done():
			case <-time.After(time.Second):
				t.Fatal("context not cancelled after signal")
			}

			cause := context.Cause(ctx)
			if tt.want != nil {
				if !errors.Is(cause, tt.want) {
					t.Errorf("Cause() = %v, want %v", cause, tt.want)
				}
			} else if cause.Error() != tt.sig.String() {
				t.Errorf("Cause() = %q, want %q", cause, tt.sig.String())
			}
		})
	}
}

func TestSignalsReleaseCancelsContext(t *testing.T) {
	ctx, release := signals(context.Background(), syscall.SIGUSR1)
	release()

	if !errors.Is(context.Cause(ctx), context.Canceled) {
		t.Errorf("Cause() = %v, want context.Canceled", context.Cause(ctx))
	}
}

func TestSignalsWithCancelledParent(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	cancel()

	ctx, release := signals(parent, syscall.SIGUSR1)
	t.Cleanup(release)

	if ctx.Err() == nil {
		t.Error("context is not cancelled with a cancelled parent")
	}
}
