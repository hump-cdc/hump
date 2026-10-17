package kafka

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"
	"sync"

	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/sasl"
	"github.com/twmb/franz-go/pkg/sasl/plain"
	"github.com/twmb/franz-go/pkg/sasl/scram"

	sinkcfg "hump/config/sink"
	sink "hump/internal/port/sink"
	"hump/pkg/app/shutdown"
)

// clientKey identifies a Kafka client by the rendered url parameters
// that affect broker connectivity, so clients can be cached and reused across
// sink invocations with identical settings.
type clientKey struct {
	url      string
	clientID string

	username string
	password string
	sasl     string
	tls      bool
}

var (
	clientsMu sync.Mutex
	clients   = map[clientKey]*kgo.Client{}
)

// getClient returns a cached [kgo.Client] for the rendered Kafka url
// described by config, creating one on first use. The returned client is
// shared across callers with identical url parameters and is closed on
// process shutdown via [shutdown.OnShutdown].
func getClient(config sinkcfg.KafkaSink, operationContext *sink.Context) (*kgo.Client, error) {
	url, err := config.URL.Render(operationContext)
	if err != nil {
		return nil, fmt.Errorf("render url: %w", err)
	}

	username, err := config.Username.Render(operationContext)
	if err != nil {
		return nil, fmt.Errorf("render username: %w", err)
	}

	password, err := config.Password.Render(operationContext)
	if err != nil {
		return nil, fmt.Errorf("render password: %w", err)
	}

	clientID, err := config.ClientID.Render(operationContext)
	if err != nil {
		return nil, fmt.Errorf("render clientId: %w", err)
	}

	key := clientKey{
		url:      url,
		tls:      config.TLS,
		sasl:     config.SASL,
		username: username,
		password: password,
		clientID: clientID,
	}

	clientsMu.Lock()
	defer clientsMu.Unlock()

	if client, ok := clients[key]; ok {
		return client, nil
	}

	seeds := strings.Split(key.url, ",")
	for i, seed := range seeds {
		seeds[i] = strings.TrimSpace(seed)
	}

	options := []kgo.Opt{
		kgo.SeedBrokers(seeds...),
		kgo.WithHooks(newConnHooks(url)),
	}

	if key.clientID != "" {
		options = append(options, kgo.ClientID(key.clientID))
	}

	if key.tls {
		options = append(options, kgo.DialTLSConfig(&tls.Config{MinVersion: tls.VersionTLS12}))
	}

	if key.sasl != "" {
		mechanism, err := saslMechanism(key.sasl, key.username, key.password)
		if err != nil {
			return nil, fmt.Errorf("kafka client for %s: %w", url, err)
		}

		options = append(options, kgo.SASL(mechanism))
	}

	client, err := kgo.NewClient(options...)
	if err != nil {
		return nil, fmt.Errorf("kafka client for %s: %w", url, err)
	}

	clients[key] = client

	shutdown.OnShutdown(shutdown.StageSinks, fmt.Sprintf("kafka_sink(%s)", url), func(_ context.Context) error {
		client.Close()
		return nil
	})

	return client, nil
}

func saslMechanism(mechanism, user, pass string) (sasl.Mechanism, error) {
	switch mechanism {
	case "plain":
		return plain.Auth{User: user, Pass: pass}.AsMechanism(), nil
	case "scram-sha-256":
		return scram.Auth{User: user, Pass: pass}.AsSha256Mechanism(), nil
	case "scram-sha-512":
		return scram.Auth{User: user, Pass: pass}.AsSha512Mechanism(), nil
	default:
		return nil, fmt.Errorf("unsupported sasl mechanism: %s", mechanism)
	}
}
