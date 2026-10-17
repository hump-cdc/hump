// Package getoperations serves the operations log of the web UI: one page of
// sink deliveries, newest first, cut by a filter.
package getoperations

import (
	"context"
	"errors"
	"time"

	operationscollector "hump/internal/port/operations_collector"

	"github.com/danielgtaylor/huma/v2"
)

// Controller serves POST /api/operations.
type Controller struct {
	useCase *UseCase
}

// NewController returns a controller serving the log searches of useCase.
func NewController(useCase *UseCase) *Controller {
	return &Controller{
		useCase: useCase,
	}
}

// Handler returns the huma handler rendering one page of the operations log.
func (c *Controller) Handler() func(ctx context.Context, input *OperationsRequest) (*OperationsResponse, error) {
	return func(_ context.Context, input *OperationsRequest) (*OperationsResponse, error) {
		page, err := c.useCase.Search(Filter{
			From:       input.Body.From,
			To:         input.Body.To,
			Sinks:      input.Body.Sinks,
			Hosts:      input.Body.Hosts,
			Targets:    input.Body.Targets,
			Tables:     input.Body.Tables,
			Operations: input.Body.Operations,
			Statuses:   input.Body.Statuses,
			Text:       input.Body.Text,
		})
		if errors.Is(err, ErrDisabled) {
			return nil, huma.Error503ServiceUnavailable(err.Error())
		}
		if err != nil {
			return nil, huma.Error422UnprocessableEntity(err.Error())
		}

		resp := OperationsResponse{}
		resp.Body.Total = page.Total
		resp.Body.From = page.Start
		resp.Body.To = page.End
		resp.Body.Written = page.Written
		resp.Body.Items = c.items(page.Records)

		return &resp, nil
	}
}

// items renders the records of the page, keeping the newest-first order the
// collector answered in.
func (c *Controller) items(source []operationscollector.Record) []OperationItem {
	result := make([]OperationItem, 0, len(source))

	for i := range source {
		rec := &source[i]
		result = append(result, OperationItem{
			Seq:         rec.Seq,
			StartedAt:   rec.StartedAt,
			Duration:    toMillis(rec.Duration),
			Timestamp:   rec.Timestamp,
			Database:    string(rec.Database),
			Table:       string(rec.Table.Schema) + "." + string(rec.Table.Name),
			Operation:   string(rec.Operation),
			Sink:        rec.SinkType,
			Host:        rec.Host,
			Target:      rec.Target,
			Status:      string(rec.Status),
			Error:       rec.Error,
			Retries:     rec.Retries,
			Payload:     string(rec.Payload),
			PayloadSize: rec.PayloadSize,
			Metadata:    metadata(rec.Metadata),
			Attempts:    c.attempts(rec.Attempts),
		})
	}

	return result
}

// attempts renders the attempts of one record, numbering them from 1 so the
// UI never has to derive the number from a slice position.
func (c *Controller) attempts(source []operationscollector.Attempt) []OperationAttempt {
	result := make([]OperationAttempt, 0, len(source))

	for i, attempt := range source {
		result = append(result, OperationAttempt{
			Attempt:   i + 1,
			StartedAt: attempt.StartedAt,
			Duration:  toMillis(attempt.Duration),
			Status:    string(attempt.Status),
			Error:     attempt.Error,
		})
	}

	return result
}

// metadata substitutes an empty map for one a sink never filled, since the
// field is required and a null would fail the client's schema.
func metadata(source map[string]string) map[string]string {
	if source == nil {
		return map[string]string{}
	}

	return source
}

// toMillis rounds a duration to whole milliseconds, which is the unit every
// latency on the API is carried in.
func toMillis(d time.Duration) int64 {
	return d.Round(time.Millisecond).Milliseconds()
}
