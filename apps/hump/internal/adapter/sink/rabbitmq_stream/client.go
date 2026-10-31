package rabbitmq_stream

import (
	"context"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/rabbitmq/rabbitmq-stream-go-client/pkg/stream"

	sinkcfg "hump/config/sink"
	sink "hump/internal/port/sink"
	"hump/pkg/app/shutdown"
	"hump/pkg/observability/diagnostic"
)

const (
	reconnectBase      = 2 * time.Second
	minReconnectJitter = 100 * time.Millisecond
	maxReconnectJitter = 1 * time.Second
)

// envKey identifies a cached [stream.Environment] by the rendered URL.
type envKey struct {
	url string
}

// envEntry wraps the cached environment with the per-environment stream
// reconciler so the reconciler resets at the same lifecycle as the
// environment (both rebuild together after an unrecoverable failure).
type envEntry struct {
	env        *stream.Environment
	reconciler *reconciler
}

// producerKey identifies a cached [stream.Producer] by the connection URL
// and the rendered stream name.
type producerKey struct {
	url    string
	stream string
}

// rabbitProducer serializes sends and confirmation setup with publishMu.
// NotifyPublishConfirmation replaces the channel on each call; awaitConfirm holds the mutex.
type rabbitProducer struct {
	prod      *stream.Producer
	publishMu sync.Mutex
}

var (
	envsMu sync.Mutex
	envs   = map[envKey]*envEntry{}

	producersMu sync.Mutex
	producers   = map[producerKey]*rabbitProducer{}
)

// getEnvironment returns a cached [stream.Environment] for url, creating one
// on first use. A [shutdown.OnShutdown] hook is registered to close the
// environment at process exit.
func getEnvironment(url string) (*envEntry, error) {
	envsMu.Lock()
	defer envsMu.Unlock()

	if e, ok := envs[envKey{url: url}]; ok {
		return e, nil
	}

	env, err := stream.NewEnvironment(stream.NewEnvironmentOptions().SetUri(url))
	if err != nil {
		return nil, diagnostic.Error("create stream environment for "+diagnostic.Endpoint(url), err)
	}

	entry := &envEntry{
		env:        env,
		reconciler: newReconciler(),
	}
	envs[envKey{url: url}] = entry

	shutdown.OnShutdown(shutdown.StageSinks, fmt.Sprintf("rabbitmq_stream_sink_env(%s)", diagnostic.Endpoint(url)), func(_ context.Context) error {
		envsMu.Lock()
		delete(envs, envKey{url: url})
		envsMu.Unlock()
		return diagnostic.Error("RabbitMQ stream environment close failed", env.Close())
	})

	logger.Info("RabbitMQ stream environment established", "url", diagnostic.Endpoint(url))

	return entry, nil
}

// getProducer caches producers by (url, streamName), reconciling the stream on first use.
// Shutdown and unexpected closure evict the producer from the cache.
func getProducer(ctx context.Context, cfg sinkcfg.RabbitMQStreamSink, operationContext *sink.Context, streamName string) (*rabbitProducer, error) {
	url, err := cfg.URL.Render(operationContext)
	if err != nil {
		return nil, fmt.Errorf("render url: %w", err)
	}

	key := producerKey{url: url, stream: streamName}

	producersMu.Lock()
	if p, ok := producers[key]; ok {
		producersMu.Unlock()
		return p, nil
	}
	producersMu.Unlock()

	// Resolve env (separate cache; OK to take its mutex outside producersMu).
	entry, err := getEnvironment(url)
	if err != nil {
		return nil, err
	}

	if err := entry.reconciler.ensure(ctx, entry.env, cfg, streamName); err != nil {
		return nil, err
	}

	// Re-check the cache under lock — a concurrent caller may have raced
	// past us between the lookup above and now.
	producersMu.Lock()
	defer producersMu.Unlock()

	if p, ok := producers[key]; ok {
		return p, nil
	}

	streamProd, err := entry.env.NewProducer(streamName, nil)
	if err != nil {
		return nil, diagnostic.Error("create stream producer for "+diagnostic.Endpoint(url), err)
	}

	rp := &rabbitProducer{prod: streamProd}
	producers[key] = rp

	go watchProducerClose(key, streamProd)

	shutdown.OnShutdown(shutdown.StageSinks, fmt.Sprintf("rabbitmq_stream_sink_producer(%s)", diagnostic.Endpoint(url)), func(_ context.Context) error {
		producersMu.Lock()
		delete(producers, key)
		producersMu.Unlock()
		return diagnostic.Error("RabbitMQ stream producer close failed", streamProd.Close())
	})

	logger.Info("RabbitMQ stream producer created", "url", diagnostic.Endpoint(url), "stream", streamName)

	return rp, nil
}

// watchProducerClose evicts the producer entry on close so the next publish
// recreates it. The stream client emits a single Event when the producer
// goes down; after that the channel returns the zero Event repeatedly.
func watchProducerClose(key producerKey, prod *stream.Producer) {
	ch := prod.NotifyClose()
	ev, ok := <-ch
	if !ok {
		return
	}
	logger.Warn("RabbitMQ stream producer closed", "url", diagnostic.Endpoint(key.url), "stream", key.stream, "err", diagnostic.Error("RabbitMQ stream producer closed", ev.Err))

	producersMu.Lock()
	current, present := producers[key]
	if present && current.prod == prod {
		delete(producers, key)
	}
	producersMu.Unlock()

	// Pace eviction to avoid hammering the broker on a transient blip — the
	// next Process call recreates the producer immediately after this sleep.
	wait := reconnectBase + jitter(minReconnectJitter, maxReconnectJitter)
	time.Sleep(wait)
}

// jitter returns a duration uniformly distributed in [minD, maxD].
func jitter(minD, maxD time.Duration) time.Duration {
	if maxD <= minD {
		return minD
	}
	delta := maxD - minD
	return minD + time.Duration(rand.Int64N(int64(delta))) //nolint:gosec // reconnect jitter, not security-sensitive
}
