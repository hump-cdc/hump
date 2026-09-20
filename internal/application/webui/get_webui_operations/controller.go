package getwebuioperations

import (
	"context"
	"time"

	"hump/internal/domain"
	operationscollector "hump/internal/port/operations_collector"

	"github.com/danielgtaylor/huma/v2"
)

type Controller struct {
	collector operationscollector.Collector
}

// NewController returns a controller serving the log of collector, which is
// nil when the web UI is disabled.
func NewController(collector operationscollector.Collector) *Controller {
	return &Controller{
		collector: collector,
	}
}

func (c *Controller) Handler() func(ctx context.Context, input *OperationsRequest) (*OperationsResponse, error) {
	return func(_ context.Context, input *OperationsRequest) (*OperationsResponse, error) {
		if c.collector == nil {
			return nil, huma.Error503ServiceUnavailable("operations log disabled")
		}

		if input.Body.To < input.Body.From {
			return nil, huma.Error422UnprocessableEntity("to is before from")
		}

		page := c.collector.Search(operationscollector.Query{
			SinkTypes:  input.Body.Sinks,
			Tables:     input.Body.Tables,
			Operations: convert[domain.OperationType](input.Body.Operations),
			Statuses:   convert[domain.DeliveryStatus](input.Body.Statuses),
			Text:       input.Body.Text,
			Start:      input.Body.From,
			End:        input.Body.To,
		})

		resp := OperationsResponse{}
		resp.Body.Total = page.Total
		resp.Body.From = page.Start
		resp.Body.To = page.End
		resp.Body.Written = page.Written
		resp.Body.Items = c.items(page.Records)

		return &resp, nil
	}
}

// convert casts a requested list to the domain vocabulary, keeping an empty
// one nil so it still matches everything.
func convert[T ~string](source []string) []T {
	if len(source) == 0 {
		return nil
	}

	result := make([]T, 0, len(source))
	for _, value := range source {
		result = append(result, T(value))
	}

	return result
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
