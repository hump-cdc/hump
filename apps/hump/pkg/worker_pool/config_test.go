package workerpool

import "testing"

func TestNewConfig_correct(t *testing.T) {
	tests := map[string]struct {
		opts []Option
		want config
	}{
		"default": {
			want: config{
				workers:            10,
				queueSize:          10,
				errorBuf:           10,
				backpressurePolicy: BackpressureBlock,
			},
		},
		"20 workers": {
			opts: []Option{WithWorkers(20), WithQueueSize(20)},
			want: config{
				workers:            20,
				queueSize:          20,
				errorBuf:           10,
				backpressurePolicy: BackpressureBlock,
			},
		},
		"20 items queue": {
			opts: []Option{WithQueueSize(20)},
			want: config{
				workers:            10,
				queueSize:          20,
				errorBuf:           10,
				backpressurePolicy: BackpressureBlock,
			},
		},
		"20 items error buffer": {
			opts: []Option{WithErrorBufferSize(20)},
			want: config{
				workers:            10,
				queueSize:          10,
				errorBuf:           20,
				backpressurePolicy: BackpressureBlock,
			},
		},
		"drop backpressure": {
			opts: []Option{WithBackpressurePolicy(BackpressureDrop)},
			want: config{
				workers:            10,
				queueSize:          10,
				errorBuf:           10,
				backpressurePolicy: BackpressureDrop,
			},
		},
		"one worker": {
			opts: []Option{WithWorkers(1)},
			want: config{
				workers:            1,
				queueSize:          10,
				errorBuf:           10,
				backpressurePolicy: BackpressureBlock,
			},
		},
		"unbuffered error channel": {
			opts: []Option{WithErrorBufferSize(0)},
			want: config{
				workers:            10,
				queueSize:          10,
				errorBuf:           0,
				backpressurePolicy: BackpressureBlock,
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := newConfig(test.opts...)
			if *got != test.want {
				t.Errorf("got %+v, want %+v", got, test.want)
			}
		})
	}
}

func TestNewConfig_invalid(t *testing.T) {
	tests := map[string]struct {
		opts []Option
		want string
	}{
		"zero workers": {
			opts: []Option{WithWorkers(0)},
			want: "worker_pool: worker count must be greater than zero",
		},
		"negative workers": {
			opts: []Option{WithWorkers(-1)},
			want: "worker_pool: worker count must be greater than zero",
		},
		"negative queue size": {
			opts: []Option{WithQueueSize(-1)},
			want: "worker_pool: queue size must be greater or equal than zero",
		},
		"queue size smaller than worker count": {
			opts: []Option{WithQueueSize(0)},
			want: "worker_pool: queue size must be greater or equal than worker count",
		},
		"negative error buffer": {
			opts: []Option{WithErrorBufferSize(-1)},
			want: "worker_pool: error buffer size must be greater or equal than zero",
		},
		"backpressure policy below range": {
			opts: []Option{WithBackpressurePolicy(BackpressureBlock - 1)},
			want: "worker_pool: invalid backpressure policy",
		},
		"backpressure policy above range": {
			opts: []Option{WithBackpressurePolicy(BackpressureDrop + 1)},
			want: "worker_pool: invalid backpressure policy",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			defer func() {
				got := recover()
				if got == nil {
					t.Fatalf("got no panic, want %q", test.want)
				}
				if got != test.want {
					t.Errorf("got panic %v, want %q", got, test.want)
				}
			}()

			newConfig(test.opts...)
		})
	}
}
