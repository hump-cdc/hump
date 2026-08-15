package workerpool

type BackpressurePolicy int

const (
	// BackpressureBlock blocks the caller when the queue is full.
	BackpressureBlock BackpressurePolicy = iota
	// BackpressureDrop drops the task when the queue is full.
	BackpressureDrop
)
