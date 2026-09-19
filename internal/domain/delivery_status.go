package domain

// DeliveryStatus is how one delivery of a row to a sink ended.
type DeliveryStatus string

const (
	// DeliveryStatusSuccess marks a delivery that finished without an error.
	DeliveryStatusSuccess DeliveryStatus = "success"
	// DeliveryStatusRetry marks a failed attempt about to be repeated. It is an
	// attempt rather than an outcome, so it is left out of delivery counts.
	DeliveryStatusRetry DeliveryStatus = "retry"
	// DeliveryStatusFailure marks a delivery that exhausted its attempts.
	DeliveryStatusFailure DeliveryStatus = "failure"
)
