package nats

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/nats-io/nkeys"

	"hump/internal/types/config"
	"hump/internal/types/errs"
	"hump/internal/types/sink"
	"hump/pkg/app/shutdown"
)

// clientKey identifies a NATS client by the rendered connection parameters
// that affect broker connectivity, so clients can be cached and reused across
// sink invocations with identical settings.
type clientKey struct {
	url  string // rendered nats.Connect url string
	name string // rendered nats.Name (empty if unset)

	username    string // rendered username (empty if unset)
	password    string // rendered password (empty if unset)
	token       string // rendered token (empty if unset)
	credentials string // rendered credentials (empty if unset)
	nkeySeed    string // rendered nkey seed (empty if unset)
}

// natsConn bundles an open NATS connection with its derived JetStream context
// and a per-stream reconcile guard so each (connection, stream) pair runs the
// reconcile path at most once across concurrent publishers.
type natsConn struct {
	conn       *nats.Conn
	js         jetstream.JetStream
	reconciler *reconciler
}

var (
	clientsMu sync.Mutex
	clients   = map[clientKey]*natsConn{}
)

// getClient returns a cached [natsConn] for the rendered NATS connection
// described by cfg, creating one on first use.
func getClient(cfg config.NatsCommon, operationContext *sink.SinkContext) (*natsConn, error) {
	url, err := cfg.URL.Render(operationContext)
	if err != nil {
		return nil, fmt.Errorf("render url: %w", err)
	}

	name, err := cfg.Name.Render(operationContext)
	if err != nil {
		return nil, fmt.Errorf("render name: %w", err)
	}

	auth, err := resolveAuth(cfg, operationContext)
	if err != nil {
		return nil, err
	}

	key := clientKey{
		url:         url,
		name:        name,
		username:    auth.username,
		password:    auth.password,
		token:       auth.token,
		credentials: auth.credentials,
		nkeySeed:    auth.nkeySeed,
	}

	clientsMu.Lock()
	defer clientsMu.Unlock()

	if c, ok := clients[key]; ok {
		return c, nil
	}

	if name == "" {
		if hn, hnErr := os.Hostname(); hnErr == nil {
			name = hn
		}
	}

	options := []nats.Option{
		nats.MaxReconnects(-1),
		nats.RetryOnFailedConnect(true),
		nats.ReconnectWait(2 * time.Second),
		nats.ReconnectJitter(100*time.Millisecond, time.Second),
		nats.ReconnectBufSize(-1),
		nats.DisconnectErrHandler(func(c *nats.Conn, err error) {
			logger.Warn("NATS disconnected", "url", url, "err", err, "last_err", c.LastError())
		}),
		nats.ReconnectHandler(func(c *nats.Conn) {
			logger.Info("NATS reconnected", "url", c.ConnectedUrl())
		}),
		nats.ClosedHandler(func(c *nats.Conn) {
			logger.Error("NATS connection closed", "url", url, "last_err", c.LastError())
		}),
	}
	if name != "" {
		options = append(options, nats.Name(name))
	}
	if auth.option != nil {
		options = append(options, auth.option)
	}

	connection, err := nats.Connect(url, options...)
	if err != nil {
		return nil, fmt.Errorf("nats client for %s: %w", cfg.URL.Source(), err)
	}

	js, err := jetstream.New(connection)
	if err != nil {
		connection.Close()
		return nil, fmt.Errorf("nats jetstream for %s: %w", cfg.URL.Source(), err)
	}

	c := &natsConn{
		conn:       connection,
		js:         js,
		reconciler: newReconciler(js),
	}
	clients[key] = c

	shutdown.OnShutdown(fmt.Sprintf("nats_sink(%s)", url), func(_ context.Context) error {
		return connection.Drain()
	})

	return c, nil
}

// natsAuth holds the rendered auth fields that feed the cache key alongside
// the [nats.Option] that applies the credentials at connect time.
type natsAuth struct {
	username    string
	password    string
	token       string
	credentials string
	nkeySeed    string

	option nats.Option
}

// resolveAuth renders the configured auth group and returns the rendered field
// values used to derive the cache key plus the [nats.Option] that applies the
// credentials at connect time.
func resolveAuth(cfg config.NatsCommon, opCtx *sink.SinkContext) (natsAuth, error) {
	if cfg.Credentials != nil && cfg.Credentials.Source() != "" {
		credentials, err := cfg.Credentials.Render(opCtx)
		if err != nil {
			return natsAuth{}, fmt.Errorf("render credentials: %w", err)
		}

		jwt, err := nkeys.ParseDecoratedJWT([]byte(credentials))
		if err != nil {
			return natsAuth{}, fmt.Errorf("credentials jwt: %w: %w", errs.ErrSinkPermanent, err)
		}

		nkey, err := nkeys.ParseDecoratedNKey([]byte(credentials))
		if err != nil {
			return natsAuth{}, fmt.Errorf("credentials nkey: %w: %w", errs.ErrSinkPermanent, err)
		}

		seed, err := nkey.Seed()
		if err != nil {
			return natsAuth{}, fmt.Errorf("credentials seed: %w: %w", errs.ErrSinkPermanent, err)
		}

		return natsAuth{credentials: credentials, option: nats.UserJWTAndSeed(jwt, string(seed))}, nil
	}

	if cfg.NKeySeed != nil && cfg.NKeySeed.Source() != "" {
		seed, err := cfg.NKeySeed.Render(opCtx)
		if err != nil {
			return natsAuth{}, fmt.Errorf("render nkeySeed: %w", err)
		}

		kp, err := nkeys.FromSeed([]byte(seed))
		if err != nil {
			return natsAuth{}, fmt.Errorf("nkeySeed parse: %w: %w", errs.ErrSinkPermanent, err)
		}

		pub, err := kp.PublicKey()
		if err != nil {
			return natsAuth{}, fmt.Errorf("nkeySeed pubkey: %w: %w", errs.ErrSinkPermanent, err)
		}

		sigCB := func(nonce []byte) ([]byte, error) {
			signKP, sErr := nkeys.FromSeed([]byte(seed))
			if sErr != nil {
				return nil, sErr
			}
			return signKP.Sign(nonce)
		}

		return natsAuth{nkeySeed: seed, option: nats.Nkey(pub, sigCB)}, nil
	}

	hasUser := cfg.Username != nil && cfg.Username.Source() != ""
	hasPass := cfg.Password != nil && cfg.Password.Source() != ""
	if hasUser || hasPass {
		username, err := cfg.Username.Render(opCtx)
		if err != nil {
			return natsAuth{}, fmt.Errorf("render username: %w", err)
		}

		password, err := cfg.Password.Render(opCtx)
		if err != nil {
			return natsAuth{}, fmt.Errorf("render password: %w", err)
		}

		return natsAuth{username: username, password: password, option: nats.UserInfo(username, password)}, nil
	}

	if cfg.Token != nil && cfg.Token.Source() != "" {
		token, err := cfg.Token.Render(opCtx)
		if err != nil {
			return natsAuth{}, fmt.Errorf("render token: %w", err)
		}

		return natsAuth{token: token, option: nats.Token(token)}, nil
	}

	return natsAuth{}, nil
}
