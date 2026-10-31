package diagnostic

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"testing"
)

func TestEndpoint(t *testing.T) {
	tests := []struct{ raw, want string }{
		{"https://user:password@example.com:443/private?token=secret#fragment", "https://example.com:443"},
		{"nats://token@example.com:4222", "nats://example.com:4222"},
		{"amqp://user:password@broker/vhost", "amqp://broker"},
		{"amqps://broker:5671", "amqps://broker:5671"},
		{"rabbitmq-stream://user:password@broker:5552/%2F", "rabbitmq-stream://broker:5552"},
		{"rabbitmq-stream+tls://broker:5551", "rabbitmq-stream+tls://broker:5551"},
		{"http://[2001:db8::1]:80/private", "http://[2001:db8::1]:80"},
		{"https://user:p%40ss@host/private", "https://host"},
		{"", hiddenEndpoint},
		{"broker:4222", hiddenEndpoint},
		{"{{ .Secret }}", hiddenEndpoint},
		{"https://host/{{ .Secret }}", hiddenEndpoint},
		{"postgres://user:password@host/db", hiddenEndpoint},
		{"http:opaque-secret", hiddenEndpoint},
		{"https://host:wrong/private", hiddenEndpoint},
		{"https://host:99999/private", hiddenEndpoint},
		{"https://host:/private", hiddenEndpoint},
		{"https://[not-an-ip]/private", hiddenEndpoint},
		{"https://host/private%zz", hiddenEndpoint},
		{"https://host/private\nsecret", hiddenEndpoint},
		{"https://host\\private", hiddenEndpoint},
		{"https://user:password@/private", hiddenEndpoint},
		{"nats://one,nats://two", hiddenEndpoint},
	}
	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			if got := Endpoint(tt.raw); got != tt.want {
				t.Errorf("Endpoint() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEndpoints(t *testing.T) {
	got := Endpoints("nats://user:password@one:4222/private?token=secret, tls://token@two:4223,{{ .Secret }}")
	want := "nats://one:4222,tls://two:4223," + hiddenEndpoint
	if got != want {
		t.Fatalf("Endpoints() = %q, want %q", got, want)
	}
}

func TestErrorPreservesCauseWithoutFormattingIt(t *testing.T) {
	dns := &net.DNSError{Err: "sentinel-secret", Name: "secret-host", IsNotFound: true}
	cause := errors.Join(context.DeadlineExceeded, dns)
	err := Error("connection failed", cause)
	if got := fmt.Sprintf("%v %+v %s", err, err, err); strings.Contains(got, "secret") {
		t.Fatalf("formatted error leaks cause: %s", got)
	}
	if !errors.Is(err, cause) || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error lost cause: %v", err)
	}
	var got *net.DNSError
	if !errors.As(err, &got) || got != dns {
		t.Fatalf("error lost DNS type: %v", err)
	}
	if !errors.Is(errors.Unwrap(err), cause) || Error("nothing", nil) != nil {
		t.Fatal("incorrect nil or unwrap semantics")
	}
}
