package workerpool

type config struct {
	workers            int
	queueSize          int
	errorBuf           int
	backpressurePolicy BackpressurePolicy
}

func newConfig(opts ...Option) *config {
	c := configDefaults()
	c.apply(opts...)
	c.validate()
	return c
}

func configDefaults() *config {
	return &config{
		workers:            10,
		queueSize:          10,
		errorBuf:           10,
		backpressurePolicy: BackpressureBlock,
	}
}

func (c *config) apply(opts ...Option) {
	for _, opt := range opts {
		opt(c)
	}
}

func (c *config) validate() {
	if c.workers <= 0 {
		panic("worker_pool: worker count must be greater than zero")
	}

	if c.queueSize < 0 {
		panic("worker_pool: queue size must be greater or equal than zero")
	}

	if c.queueSize < c.workers {
		panic("worker_pool: queue size must be greater or equal than worker count")
	}

	if c.errorBuf < 0 {
		panic("worker_pool: error buffer size must be greater or equal than zero")
	}

	if c.backpressurePolicy < BackpressureBlock || c.backpressurePolicy > BackpressureDrop {
		panic("worker_pool: invalid backpressure policy")
	}
}

// Option customizes the worker pool configuration passed to New.
type Option func(c *config)

// WithWorkers sets the number of worker goroutines in the pool (10 by default).
// Panics if the value is not greater than zero.
func WithWorkers(workers int) Option {
	return func(c *config) {
		c.workers = workers
	}
}

// WithQueueSize sets the capacity of the task queue (10 by default).
// Panics if the value is negative or smaller than the worker count.
func WithQueueSize(queueSize int) Option {
	return func(c *config) {
		c.queueSize = queueSize
	}
}

// WithErrorBufferSize sets the capacity of the Errors channel (10 by default);
// errors reported while the channel is full are dropped. Panics if the value is negative.
func WithErrorBufferSize(errorBuf int) Option {
	return func(c *config) {
		c.errorBuf = errorBuf
	}
}

// WithBackpressurePolicy defines what Submit does when the queue is full: block until
// there is room (BackpressureBlock, the default) or drop the task (BackpressureDrop).
func WithBackpressurePolicy(policy BackpressurePolicy) Option {
	return func(c *config) {
		c.backpressurePolicy = policy
	}
}
