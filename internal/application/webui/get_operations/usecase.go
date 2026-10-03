package getoperations

import (
	"errors"

	"hump/internal/domain"
	operationscollector "hump/internal/port/operations_collector"
)

var (
	// ErrDisabled indicates that the web UI is off, so there is no log to search.
	ErrDisabled = errors.New("operations log disabled")
	// ErrInvertedWindow indicates a window whose end comes before its start.
	ErrInvertedWindow = errors.New("to is before from")
)

// Filter cuts one page out of the log. Every list is optional, and an empty
// one matches every record.
type Filter struct {
	From       int
	To         int
	Sinks      []string
	Hosts      []string
	Targets    []string
	Tables     []string
	Operations []string
	Statuses   []string
	Text       string
}

type UseCase struct {
	collector operationscollector.Collector // collector is nil when the web UI is disabled.
}

// NewUseCase returns a use case searching the log of collector.
func NewUseCase(collector operationscollector.Collector) *UseCase {
	return &UseCase{
		collector: collector,
	}
}

// Search returns the page of the log filter selects, newest first. It fails
// with ErrDisabled without a collector and ErrInvertedWindow for a window
// that ends before it starts.
func (uc *UseCase) Search(filter Filter) (operationscollector.Page, error) {
	if uc.collector == nil {
		return operationscollector.Page{}, ErrDisabled
	}

	if filter.To < filter.From {
		return operationscollector.Page{}, ErrInvertedWindow
	}

	return uc.collector.Search(operationscollector.Query{
		SinkTypes:  filter.Sinks,
		Hosts:      filter.Hosts,
		Targets:    filter.Targets,
		Tables:     filter.Tables,
		Operations: convert[domain.OperationType](filter.Operations),
		Statuses:   convert[domain.DeliveryStatus](filter.Statuses),
		Text:       filter.Text,
		Start:      filter.From,
		End:        filter.To,
	}), nil
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
