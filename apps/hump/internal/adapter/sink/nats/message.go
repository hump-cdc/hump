package nats

import (
	"fmt"
	"slices"
	"strings"

	sinkcfg "hump/config/sink"
	sink "hump/internal/port/sink"

	"github.com/nats-io/nats.go"
)

// JetStream message header names.
const (
	headerNatsMsgID          = "Nats-Msg-Id"
	headerNatsExpectedStream = "Nats-Expected-Stream"
)

// buildMessage renders the configured subject, payload, and headers against
// operationContext and returns the resulting [nats.Msg]. The rendered subject
// is validated. JetStream-specific control headers are applied separately by
// the JetStream publish path.
func buildMessage(cfg sinkcfg.NatsCommon, operationContext *sink.Context) (*nats.Msg, error) {
	subject, err := cfg.Subject.Render(operationContext)
	if err != nil {
		return nil, fmt.Errorf("render subject: %w", err)
	}

	if err := validateSubject(subject); err != nil {
		return nil, err
	}

	payload, err := cfg.Payload.Render(operationContext)
	if err != nil {
		return nil, fmt.Errorf("render payload: %w", err)
	}

	header := make(nats.Header, len(cfg.Headers))
	for name, tmpl := range cfg.Headers {
		v, renderErr := tmpl.Render(operationContext)
		if renderErr != nil {
			return nil, fmt.Errorf("render header %s: %w", name, renderErr)
		}
		if len(v) == 0 {
			continue
		}
		header[name] = []string{v}
	}

	return &nats.Msg{
		Subject: subject,
		Data:    []byte(payload),
		Header:  header,
	}, nil
}

// validateSubject enforces NATS subject grammar: non-empty, no whitespace, no
// wildcards (`*`/`>`), only `.` as a segment separator, every segment non-empty.
func validateSubject(subject string) error {
	if subject == "" {
		return fmt.Errorf("invalid subject %q: %w", subject, errInvalidSubject)
	}

	if strings.ContainsAny(subject, " \t\n\r") {
		return fmt.Errorf("invalid subject %q: %w", subject, errInvalidSubject)
	}

	if strings.ContainsAny(subject, "*>") {
		return fmt.Errorf("invalid subject %q: %w", subject, errInvalidSubject)
	}

	if slices.Contains(strings.Split(subject, "."), "") {
		return fmt.Errorf("invalid subject %q: %w", subject, errInvalidSubject)
	}

	return nil
}

// applyJetStreamHeaders renders the JetStream-specific control headers and
// merges them into header. Empty rendered values cause the corresponding
// header to be omitted. Nats-Expected-Stream is always set from the static
// stream name and overwrites any user-declared entry.
func applyJetStreamHeaders(header nats.Header, cfg sinkcfg.NatsJetStreamSink, operationContext *sink.Context) error {
	if cfg.MsgID != nil {
		msgID, err := cfg.MsgID.Render(operationContext)
		if err != nil {
			return fmt.Errorf("render jetstream.msgId: %w", err)
		}
		if len(msgID) > 0 {
			header[headerNatsMsgID] = []string{msgID}
		} else {
			delete(header, headerNatsMsgID)
		}
	}

	header[headerNatsExpectedStream] = []string{cfg.Stream.Name}

	return nil
}
