package test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/rabbitmq/rabbitmq-stream-go-client/pkg/amqp"
	"github.com/rabbitmq/rabbitmq-stream-go-client/pkg/stream"
)

func TestPipeline_RabbitMQStream_Integration(t *testing.T) {
	// Test setup
	ctx, db, teardown, wait, err := setup(t,
		withDatabaseURL("postgres://hump:hump@localhost:5437/hump?sslmode=disable"),
		withConfig(`
sinks:
    - name: rmqstream1
      rabbitmq-stream:
          url: "rabbitmq-stream://hump:hump@localhost:5552/%2F"
          stream:
              name: "{{.Table}}"
              mode: create-when-missing
          payload: '{ {{if .Prev}}"before":{{.Prev | json}},{{end}}{{if .Next}}"after":{{.Next | json}},{{end}}"source":{"version":{{.Version | json}},"connector":"postgres","name":"hump","db":{{.Database | json}},"schema":{{.Schema | json}},"table":{{.Table | json}},"ts_ms":{{.Timestamp | unixepoch | mul 1000 | json}}},"op":{{.Operation | lower | trunc 1 | replace "i" "c" | json}},"ts_ms":{{now | unixepoch | mul 1000 | json}} }'
          headers:
              X-Idempotency-Key: '{{ sha256 (cat .Schema .Table .Operation .Timestamp (.Prev | json) (.Next | json)) }}'
              traceparent: '{{ index .Headers "traceparent" }}'
      failure:
          mode: crash
      retry:
          maxAttempts: 5
          attemptTimeout: 10s
          backoff:
              type: exponential
              initial: 200ms
              max: 2s
              multiplier: 2
              jitter: none
      delivery:
          mode: reliable
          maxConcurrent: 64
          concurrencyKey: '{{.SinkType}}({{.SinkName}})'

tables:
    - tables:
          - schema: public
            table: users
          - schema: public
            table: posts
          - schema: public
            table: comments
          - schema: public
            table: events
      sinks:
          - preset: rmqstream1
`),
		withPubName(fmt.Sprintf("rmq_stream_test_pub_%d", time.Now().UnixNano())),
		withSlotName(fmt.Sprintf("rmq_stream_test_slot_%d", time.Now().UnixNano())),
	)
	if err != nil {
		t.Fatalf("Failed to setup test: %v", err)
	}

	// RabbitMQ Stream setup
	t.Log("Connecting to rabbitmq stream...")
	reader, err := connectRabbitMQStream(
		ctx,
		"rabbitmq-stream://hump:hump@localhost:5552/%2F",
		[]string{"users", "posts", "comments", "events"})
	if err != nil {
		t.Fatalf("Failed to create RabbitMQ stream client: %v", err)
	}
	defer reader.close()
	t.Log("Connected to rabbitmq stream")

	// Cross-phase state. INSERT seeds these; UPDATE/DELETE read back the
	// previously known value to assert the before-image.
	userIDs := []int64{}
	userNames := map[int64]string{}

	postIDs := []string{}
	postTitles := map[string]string{}
	postAuthor := map[string]int64{}

	commentRefs := []commentRef{}
	commentBodies := map[int64]string{}

	eventIDs := []int64{}
	eventPayloads := map[int64]string{}

	t.Run("INSERT", func(t *testing.T) {
		t.Run("users", func(t *testing.T) {
			for range 5 {
				email := gofakeit.Email()
				name := gofakeit.Name()
				t.Run(email, func(t *testing.T) {
					id, err := createUser(ctx, db, email, name)
					if err != nil {
						t.Fatalf("Failed to insert user: %v", err)
					}
					userIDs = append(userIDs, id)
					userNames[id] = name

					p, err := reader.next(ctx, "users", 5*time.Second)
					if err != nil {
						t.Fatalf("Failed to poll rabbitmq: %v", err)
					}
					if p.Op != "c" {
						t.Errorf("Expected op c, got %s", p.Op)
					}
					if p.Source.Table != "users" {
						t.Errorf("Expected source table users, got %s", p.Source.Table)
					}
					if p.After["name"] != name {
						t.Errorf("Expected name %s, got %v", name, p.After["name"])
					}
					if p.After["email"] != email {
						t.Errorf("Expected email %s, got %v", email, p.After["email"])
					}
				})
			}
		})

		t.Run("posts", func(t *testing.T) {
			for _, authorID := range userIDs {
				title := gofakeit.Sentence(4)
				body := gofakeit.Sentence(8)
				t.Run(fmt.Sprintf("Author %d", authorID), func(t *testing.T) {
					id, err := createPost(ctx, db, authorID, title, body)
					if err != nil {
						t.Fatalf("Failed to insert post: %v", err)
					}
					postIDs = append(postIDs, id)
					postTitles[id] = title
					postAuthor[id] = authorID

					p, err := reader.next(ctx, "posts", 5*time.Second)
					if err != nil {
						t.Fatalf("Failed to poll rabbitmq: %v", err)
					}
					if p.Op != "c" {
						t.Errorf("Expected op c, got %s", p.Op)
					}
					if p.Source.Table != "posts" {
						t.Errorf("Expected source table posts, got %s", p.Source.Table)
					}
					if got := stringify(p.After["id"]); got != id {
						t.Errorf("Expected after id %s, got %s", id, got)
					}
					if p.After["title"] != title {
						t.Errorf("Expected title %s, got %v", title, p.After["title"])
					}
				})
			}
		})

		t.Run("comments", func(t *testing.T) {
			for i, postID := range postIDs {
				authorID := postAuthor[postID]
				body := gofakeit.Sentence(6)
				t.Run(fmt.Sprintf("Post %d", i), func(t *testing.T) {
					id, err := createComment(ctx, db, postID, authorID, body)
					if err != nil {
						t.Fatalf("Failed to insert comment: %v", err)
					}
					commentRefs = append(commentRefs, commentRef{id: id, authorID: authorID})
					commentBodies[id] = body

					p, err := reader.next(ctx, "comments", 5*time.Second)
					if err != nil {
						t.Fatalf("Failed to poll rabbitmq: %v", err)
					}
					if p.Op != "c" {
						t.Errorf("Expected op c, got %s", p.Op)
					}
					if p.Source.Table != "comments" {
						t.Errorf("Expected source table comments, got %s", p.Source.Table)
					}
					if p.After["body"] != body {
						t.Errorf("Expected body %s, got %v", body, p.After["body"])
					}
				})
			}
		})

		t.Run("events", func(t *testing.T) {
			for _, userID := range userIDs {
				kind := gofakeit.Word()
				payload := fmt.Sprintf(`{"v":%d}`, gofakeit.Number(1, 1000))
				t.Run(fmt.Sprintf("User %d", userID), func(t *testing.T) {
					id, err := createEvent(ctx, db, userID, kind, payload)
					if err != nil {
						t.Fatalf("Failed to insert event: %v", err)
					}
					eventIDs = append(eventIDs, id)
					eventPayloads[id] = payload

					p, err := reader.next(ctx, "events", 5*time.Second)
					if err != nil {
						t.Fatalf("Failed to poll rabbitmq: %v", err)
					}
					if p.Op != "c" {
						t.Errorf("Expected op c, got %s", p.Op)
					}
					if p.Source.Table != "events" {
						t.Errorf("Expected source table events, got %s", p.Source.Table)
					}
					if p.After["kind"] != kind {
						t.Errorf("Expected kind %s, got %v", kind, p.After["kind"])
					}
					if got := stringify(p.After["payload"]); got != payload {
						t.Errorf("Expected payload %s, got %s", payload, got)
					}
				})
			}
		})
	})

	t.Run("UPDATE", func(t *testing.T) {
		t.Run("users", func(t *testing.T) {
			for _, userID := range userIDs {
				t.Run(fmt.Sprintf("User %d", userID), func(t *testing.T) {
					oldName := userNames[userID]
					newName := gofakeit.Name()
					if err := updateUser(ctx, db, userID, newName); err != nil {
						t.Fatalf("Failed to update user: %v", err)
					}

					p, err := reader.next(ctx, "users", 5*time.Second)
					if err != nil {
						t.Fatalf("Failed to poll rabbitmq: %v", err)
					}
					assertUpdate(t, p, "users")
					if p.Before["name"] != oldName {
						t.Errorf("Expected before name %s, got %v", oldName, p.Before["name"])
					}
					if p.After["name"] != newName {
						t.Errorf("Expected after name %s, got %v", newName, p.After["name"])
					}
					userNames[userID] = newName
				})
			}
		})

		t.Run("posts", func(t *testing.T) {
			for i, postID := range postIDs {
				t.Run(fmt.Sprintf("Post %d", i), func(t *testing.T) {
					oldTitle := postTitles[postID]
					newTitle := gofakeit.Sentence(4)
					if err := updatePost(ctx, db, postID, newTitle); err != nil {
						t.Fatalf("Failed to update post: %v", err)
					}

					p, err := reader.next(ctx, "posts", 5*time.Second)
					if err != nil {
						t.Fatalf("Failed to poll rabbitmq: %v", err)
					}
					assertUpdate(t, p, "posts")
					if p.Before["title"] != oldTitle {
						t.Errorf("Expected before title %s, got %v", oldTitle, p.Before["title"])
					}
					if p.After["title"] != newTitle {
						t.Errorf("Expected after title %s, got %v", newTitle, p.After["title"])
					}
					postTitles[postID] = newTitle
				})
			}
		})

		t.Run("comments", func(t *testing.T) {
			for _, c := range commentRefs {
				t.Run(fmt.Sprintf("Comment %d", c.id), func(t *testing.T) {
					oldBody := commentBodies[c.id]
					newBody := gofakeit.Sentence(6)
					if err := updateComment(ctx, db, c.id, c.authorID, newBody); err != nil {
						t.Fatalf("Failed to update comment: %v", err)
					}

					p, err := reader.next(ctx, "comments", 5*time.Second)
					if err != nil {
						t.Fatalf("Failed to poll rabbitmq: %v", err)
					}
					assertUpdate(t, p, "comments")
					if p.Before["body"] != oldBody {
						t.Errorf("Expected before body %s, got %v", oldBody, p.Before["body"])
					}
					if p.After["body"] != newBody {
						t.Errorf("Expected after body %s, got %v", newBody, p.After["body"])
					}
					commentBodies[c.id] = newBody
				})
			}
		})

		t.Run("events", func(t *testing.T) {
			for _, eventID := range eventIDs {
				t.Run(fmt.Sprintf("Event %d", eventID), func(t *testing.T) {
					oldPayload := eventPayloads[eventID]
					newPayload := fmt.Sprintf(`{"v":%d}`, gofakeit.Number(1001, 2000))
					if err := updateEvent(ctx, db, eventID, newPayload); err != nil {
						t.Fatalf("Failed to update event: %v", err)
					}

					p, err := reader.next(ctx, "events", 5*time.Second)
					if err != nil {
						t.Fatalf("Failed to poll rabbitmq: %v", err)
					}
					assertUpdate(t, p, "events")
					if got := stringify(p.Before["payload"]); got != oldPayload {
						t.Errorf("Expected before payload %s, got %s", oldPayload, got)
					}
					if got := stringify(p.After["payload"]); got != newPayload {
						t.Errorf("Expected after payload %s, got %s", newPayload, got)
					}
					eventPayloads[eventID] = newPayload
				})
			}
		})
	})

	// DELETE runs child-first (events, comments, posts, users): deleting a user
	// first would cascade-delete its posts/comments and emit extra delete
	// records, desyncing the one-op/one-poll assertion loop.
	t.Run("DELETE", func(t *testing.T) {
		t.Run("events", func(t *testing.T) {
			for _, eventID := range eventIDs {
				t.Run(fmt.Sprintf("Event %d", eventID), func(t *testing.T) {
					if err := deleteEvent(ctx, db, eventID); err != nil {
						t.Fatalf("Failed to delete event: %v", err)
					}
					p, err := reader.next(ctx, "events", 5*time.Second)
					if err != nil {
						t.Fatalf("Failed to poll rabbitmq: %v", err)
					}
					assertDelete(t, p, "events", fmt.Sprintf("%d", eventID))
				})
			}
		})

		t.Run("comments", func(t *testing.T) {
			for _, c := range commentRefs {
				t.Run(fmt.Sprintf("Comment %d", c.id), func(t *testing.T) {
					if err := deleteComment(ctx, db, c.id, c.authorID); err != nil {
						t.Fatalf("Failed to delete comment: %v", err)
					}
					p, err := reader.next(ctx, "comments", 5*time.Second)
					if err != nil {
						t.Fatalf("Failed to poll rabbitmq: %v", err)
					}
					assertDelete(t, p, "comments", fmt.Sprintf("%d", c.id))
				})
			}
		})

		t.Run("posts", func(t *testing.T) {
			for i, postID := range postIDs {
				t.Run(fmt.Sprintf("Post %d", i), func(t *testing.T) {
					if err := deletePost(ctx, db, postID); err != nil {
						t.Fatalf("Failed to delete post: %v", err)
					}
					p, err := reader.next(ctx, "posts", 5*time.Second)
					if err != nil {
						t.Fatalf("Failed to poll rabbitmq: %v", err)
					}
					assertDelete(t, p, "posts", postID)
				})
			}
		})

		t.Run("users", func(t *testing.T) {
			for _, userID := range userIDs {
				t.Run(fmt.Sprintf("User %d", userID), func(t *testing.T) {
					if err := deleteUser(ctx, db, userID); err != nil {
						t.Fatalf("Failed to delete user: %v", err)
					}
					p, err := reader.next(ctx, "users", 5*time.Second)
					if err != nil {
						t.Fatalf("Failed to poll rabbitmq: %v", err)
					}
					assertDelete(t, p, "users", fmt.Sprintf("%d", userID))
				})
			}
		})
	})

	// TRUNCATE per table, child-first (events, comments, posts, users): every
	// truncate helper uses CASCADE because posts/comments/users are FK-referenced,
	// so a parent's truncate re-truncates already-emptied children. Verifying
	// child-first means each table's own truncate is checked before a parent
	// cascade adds a leftover message to its stream. The reader buffers those
	// leftovers harmlessly under their own stream.
	t.Run("TRUNCATE", func(t *testing.T) {
		t.Run("events", func(t *testing.T) {
			if err := truncateEvents(ctx, db); err != nil {
				t.Fatalf("Failed to truncate events: %v", err)
			}
			assertTruncateRabbitMQStream(ctx, t, reader, "events")
		})

		t.Run("comments", func(t *testing.T) {
			if err := truncateComments(ctx, db); err != nil {
				t.Fatalf("Failed to truncate comments: %v", err)
			}
			assertTruncateRabbitMQStream(ctx, t, reader, "comments")
		})

		t.Run("posts", func(t *testing.T) {
			if err := truncatePosts(ctx, db); err != nil {
				t.Fatalf("Failed to truncate posts: %v", err)
			}
			assertTruncateRabbitMQStream(ctx, t, reader, "posts")
		})

		t.Run("users", func(t *testing.T) {
			if err := truncateUsers(ctx, db); err != nil {
				t.Fatalf("Failed to truncate users: %v", err)
			}
			assertTruncateRabbitMQStream(ctx, t, reader, "users")
		})
	})

	teardown()
	if err := wait(); err != nil {
		t.Fatalf("Failed to wait for pipeline shutdown: %v", err)
	}
}

// assertTruncateRabbitMQStream drains every truncate message on the table's
// stream — a single TRUNCATE may emit more than one for the hash-partitioned
// `comments` and the `events` hypertable — and checks each carries op "t" with
// no row image.
func assertTruncateRabbitMQStream(ctx context.Context, t *testing.T, reader *rmqStreamReader, table string) {
	t.Helper()

	// Wait until the whole reader goes idle (no new message on any stream within
	// the quiet window), then take and remove the messages buffered for table.
	// Other streams' buffered messages — e.g. those emitted by a TRUNCATE ...
	// CASCADE on a parent table — are left in place for their own subtest.
	// Settling on global idle (rather than only the target stream) ensures all
	// cascade messages have arrived before the count is asserted. Messages are
	// delivered asynchronously by the consumer callbacks, so idle means the
	// total buffered count is unchanged across one quiet window.
	for {
		if err := ctx.Err(); err != nil {
			t.Fatalf("Failed to drain rabbitmq: %v", err)
		}
		before := reader.bufLen()
		time.Sleep(time.Second)
		if reader.bufLen() == before {
			break
		}
	}

	msgs := reader.take(table)

	if len(msgs) == 0 {
		t.Fatalf("Expected at least one truncate message for %s, got none", table)
	}
	for i, data := range msgs {
		p, err := payloadDecode(data)
		if err != nil {
			t.Fatalf("Failed to decode payload: %v", err)
		}
		if p.Op != "t" {
			t.Errorf("message %d: expected op t, got %s", i, p.Op)
		}
		if p.Source.Table != table {
			t.Errorf("message %d: expected source table %s, got %s", i, table, p.Source.Table)
		}
		if p.Before != nil {
			t.Errorf("message %d: expected no before image, got %v", i, p.Before)
		}
		if p.After != nil {
			t.Errorf("message %d: expected no after image, got %v", i, p.After)
		}
	}
}

// connectRabbitMQStream opens a stream environment, declares each stream, and
// attaches a push consumer per stream starting at the next offset. Streams are
// declared with the same retention defaults the sink applies
// (internal/port/sink/config_rabbitmq_stream.go: 24h / 2 GiB / 500 MiB)
// so the sink's create-when-missing declare stays idempotent rather than
// failing with a precondition mismatch. The Next() offset means only messages
// published after the consumer attaches are delivered, so persisted messages
// from a previous run are ignored.
func connectRabbitMQStream(_ context.Context, uri string, streams []string) (*rmqStreamReader, error) {
	env, err := stream.NewEnvironment(stream.NewEnvironmentOptions().SetUri(uri))
	if err != nil {
		return nil, fmt.Errorf("create stream environment: %w", err)
	}

	r := &rmqStreamReader{env: env, buf: map[string][][]byte{}}

	opts := stream.NewStreamOptions().
		SetMaxAge(24 * time.Hour).
		SetMaxLengthBytes(stream.ByteCapacity{}.B(2 * 1024 * 1024 * 1024)).
		SetMaxSegmentSizeBytes(stream.ByteCapacity{}.B(500 * 1024 * 1024))

	for _, name := range streams {
		if err := env.DeclareStream(name, opts); err != nil {
			r.close()
			return nil, fmt.Errorf("declare stream %s: %w", name, err)
		}

		name := name // capture for the handler closure
		handler := func(_ stream.ConsumerContext, msg *amqp.Message) {
			r.mu.Lock()
			r.buf[name] = append(r.buf[name], msg.GetData())
			r.mu.Unlock()
		}

		consumer, err := env.NewConsumer(name, handler, stream.NewConsumerOptions().
			SetConsumerName(fmt.Sprintf("hump-test-%d", time.Now().UnixNano())).
			SetOffset(stream.OffsetSpecification{}.Next()))
		if err != nil {
			r.close()
			return nil, fmt.Errorf("create consumer for %s: %w", name, err)
		}
		r.consumers = append(r.consumers, consumer)
	}

	return r, nil
}

// rmqStreamReader buffers consumed messages per stream so the synchronous
// "produce one row, read one message" assertion loop is not desynchronised by a
// message arriving for another stream. RabbitMQ Streams' consumer is
// push-based: each consumer's handler appends to buf under mu, and next polls
// the buffer. A single stream preserves publish order, so per-stream FIFO order
// equals produce order. Stream names equal table names (config: name "{{.Table}}").
type rmqStreamReader struct {
	env       *stream.Environment
	consumers []*stream.Consumer

	mu  sync.Mutex
	buf map[string][][]byte
}

// next returns the decoded CDC payload of the next message for stream in
// produce order, polling the buffer until one is available or timeout elapses.
// Messages for other streams are left buffered.
func (r *rmqStreamReader) next(ctx context.Context, streamName string, timeout time.Duration) (payload, error) {
	deadline := time.Now().Add(timeout)
	for {
		if err := ctx.Err(); err != nil {
			return payload{}, err
		}

		r.mu.Lock()
		if len(r.buf[streamName]) > 0 {
			data := r.buf[streamName][0]
			r.buf[streamName] = r.buf[streamName][1:]
			r.mu.Unlock()

			p, err := payloadDecode(data)
			if err != nil {
				return payload{}, fmt.Errorf("decode payload on stream %q: %w", streamName, err)
			}
			return p, nil
		}
		r.mu.Unlock()

		if time.Now().After(deadline) {
			return payload{}, fmt.Errorf("no message on stream %q within %s", streamName, timeout)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// take removes and returns all messages buffered for stream.
func (r *rmqStreamReader) take(streamName string) [][]byte {
	r.mu.Lock()
	defer r.mu.Unlock()
	msgs := r.buf[streamName]
	delete(r.buf, streamName)
	return msgs
}

func (r *rmqStreamReader) bufLen() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, msgs := range r.buf {
		n += len(msgs)
	}
	return n
}

// close stops all consumers and closes the environment.
func (r *rmqStreamReader) close() {
	for _, c := range r.consumers {
		_ = c.Close()
	}
	if r.env != nil {
		_ = r.env.Close()
	}
}
