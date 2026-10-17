package workerpool

// BackpressurePolicy decides what Submit does when the task queue is full.
type BackpressurePolicy int

const (
	// BackpressureBlock blocks the caller when the queue is full.
	BackpressureBlock BackpressurePolicy = iota
	// BackpressureDrop drops the task when the queue is full.
	BackpressureDrop
)
