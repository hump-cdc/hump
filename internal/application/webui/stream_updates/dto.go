package streamupdates

// OperationEvent is one finished delivery, reduced to where it went.
type OperationEvent struct {
	Schema string `json:"schema" doc:"Schema of the source table"`
	Table  string `json:"table"  doc:"Name of the source table"`
	Sink   string `json:"sink"   doc:"Sink type the row was delivered to"`
	Host   string `json:"host"   doc:"Connection host the sink URL renders to"`
	Target string `json:"target" doc:"Target the sink rendered for this row: topic, subject, exchange, stream or URL"`
}

// WelcomeEvent opens every stream, so the client learns it is connected before
// the first delivery arrives.
type WelcomeEvent struct{}

// OperationsEvent carries every delivery the pipeline made since the previous
// one, so a busy pipeline reaches the client as a couple of events per second.
type OperationsEvent struct {
	Operations []OperationEvent `json:"operations" doc:"Deliveries made since the previous event, oldest first"`
}
