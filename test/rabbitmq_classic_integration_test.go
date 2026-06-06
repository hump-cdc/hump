package test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	amqp "github.com/rabbitmq/amqp091-go"
)

func TestPipeline_RabbitMQClassic_Integration(t *testing.T) {
	// Test setup
	ctx, db, teardown, wait, err := setup(t,
		withDatabaseURL("postgres://hump:hump@localhost:5437/hump?sslmode=disable"),
		withConfig(`
sinks:
    - name: rabbitmq1
      rabbitmq:
          url: "amqp://hump:hump@localhost:5672/"
          exchange:
              name: "hump.{{.Schema}}"
              type: topic
          routingKey: "{{.Schema}}.{{.Table}}"
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
          - preset: rabbitmq1
`),
		withPubName(fmt.Sprintf("rabbitmq_classic_test_pub_%d", time.Now().UnixNano())),
		withSlotName(fmt.Sprintf("rabbitmq_classic_test_slot_%d", time.Now().UnixNano())),
	)
	if err != nil {
		t.Fatalf("Failed to setup test: %v", err)
	}

	// RabbitMQ setup
	t.Log("Connecting to rabbitmq...")
	conn, reader, err := connectRabbit("amqp://hump:hump@localhost:5672/")
	if err != nil {
		t.Fatalf("Failed to create RabbitMQ client: %v", err)
	}
	defer conn.Close()
	t.Log("Connected to rabbitmq")

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
	// cascade adds a leftover message to its routing key. assertTruncateRabbit
	// buffers those leftovers harmlessly under their own table.
	t.Run("TRUNCATE", func(t *testing.T) {
		t.Run("events", func(t *testing.T) {
			if err := truncateEvents(ctx, db); err != nil {
				t.Fatalf("Failed to truncate events: %v", err)
			}
			assertTruncateRabbit(ctx, t, reader, "events")
		})

		t.Run("comments", func(t *testing.T) {
			if err := truncateComments(ctx, db); err != nil {
				t.Fatalf("Failed to truncate comments: %v", err)
			}
			assertTruncateRabbit(ctx, t, reader, "comments")
		})

		t.Run("posts", func(t *testing.T) {
			if err := truncatePosts(ctx, db); err != nil {
				t.Fatalf("Failed to truncate posts: %v", err)
			}
			assertTruncateRabbit(ctx, t, reader, "posts")
		})

		t.Run("users", func(t *testing.T) {
			if err := truncateUsers(ctx, db); err != nil {
				t.Fatalf("Failed to truncate users: %v", err)
			}
			assertTruncateRabbit(ctx, t, reader, "users")
		})
	})

	teardown()
	if err := wait(); err != nil {
		t.Fatalf("Failed to wait for pipeline shutdown: %v", err)
	}
}

// assertTruncateRabbit drains every truncate message on the table's routing key
// — a single TRUNCATE may emit more than one for the hash-partitioned
// `comments` and the `events` hypertable — and checks each carries op "t" with
// no row image.
func assertTruncateRabbit(ctx context.Context, t *testing.T, reader *rabbitReader, table string) {
	t.Helper()

	// Poll until the whole stream goes idle (no new message on any table within
	// the quiet window), then take and remove the messages buffered for table.
	// Other tables' buffered messages — e.g. those emitted by a TRUNCATE ...
	// CASCADE on a parent table — are left in place for their own subtest.
	// Draining to global idle (rather than only the target table) ensures all
	// cascade messages have settled before the count is asserted.
	for {
		before := reader.bufLen()
		if err := reader.poll(ctx, time.Second); err != nil {
			t.Fatalf("Failed to drain rabbitmq: %v", err)
		}
		if reader.bufLen() == before {
			break
		}
	}
	msgs := reader.buf[table]
	delete(reader.buf, table)

	if len(msgs) == 0 {
		t.Fatalf("Expected at least one truncate message for %s, got none", table)
	}
	for i, msg := range msgs {
		p, err := payloadDecode(msg.Body)
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

// connectRabbit opens an AMQP connection and a single consumer bound to the
// sink's "hump.public" topic exchange with the "public.#" routing key, so every
// table's messages land in one queue. The exchange is declared (idempotently,
// matching the sink's create-when-missing defaults) to guard against the
// consumer binding before the sink first declares it. Binding and consume are
// established before any row is produced, so no message is missed.
func connectRabbit(url string) (*amqp.Connection, *rabbitReader, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, nil, err
	}

	// durable=true, autoDelete=false — must match the sink's exchange defaults
	// or the broker raises PRECONDITION_FAILED.
	if err := ch.ExchangeDeclare("hump.public", "topic", true, false, false, false, nil); err != nil {
		conn.Close()
		return nil, nil, err
	}

	// Server-named, non-durable, auto-delete, exclusive queue scoped to this test.
	q, err := ch.QueueDeclare("", false, true, true, false, nil)
	if err != nil {
		conn.Close()
		return nil, nil, err
	}

	if err := ch.QueueBind(q.Name, "public.#", "hump.public", false, nil); err != nil {
		conn.Close()
		return nil, nil, err
	}

	deliveries, err := ch.Consume(q.Name, "", true, true, false, false, nil)
	if err != nil {
		conn.Close()
		return nil, nil, err
	}

	return conn, newRabbitReader(deliveries), nil
}

// rabbitReader buffers consumed deliveries per table so the synchronous "produce
// one row, read one message" assertion loop is not desynchronised by a poll that
// returns a message for another table (the queue is bound to all four routing
// keys at once via "public.#"). RabbitMQ preserves FIFO order within a single
// queue, so per-table order equals produce order.
type rabbitReader struct {
	deliveries <-chan amqp.Delivery
	buf        map[string][]amqp.Delivery
}

func newRabbitReader(deliveries <-chan amqp.Delivery) *rabbitReader {
	return &rabbitReader{deliveries: deliveries, buf: map[string][]amqp.Delivery{}}
}

// next returns the decoded CDC payload of the next message for table in produce
// order, polling until one is available or timeout elapses. Messages for other
// tables are buffered.
func (r *rabbitReader) next(ctx context.Context, table string, timeout time.Duration) (payload, error) {
	deadline := time.Now().Add(timeout)
	for len(r.buf[table]) == 0 {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return payload{}, fmt.Errorf("no message on table %q within %s", table, timeout)
		}
		if err := r.poll(ctx, remaining); err != nil {
			return payload{}, err
		}
	}
	msg := r.buf[table][0]
	r.buf[table] = r.buf[table][1:]
	p, err := payloadDecode(msg.Body)
	if err != nil {
		return payload{}, fmt.Errorf("decode payload on table %q: %w", table, err)
	}
	return p, nil
}

// poll drains deliveries currently available on the consumer channel and buckets
// them by table (the routing key with its "public." schema prefix removed). A
// poll that times out with no message is not an error (the caller's deadline
// bounds the overall wait). The total wait is bounded by timeout.
func (r *rabbitReader) poll(ctx context.Context, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		remaining := time.Until(deadline)
		if remaining <= 0 {
			return nil
		}
		if remaining > 100*time.Millisecond {
			remaining = 100 * time.Millisecond
		}

		timer := time.NewTimer(remaining)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
			return nil
		case d, ok := <-r.deliveries:
			timer.Stop()
			if !ok {
				return fmt.Errorf("rabbitmq delivery channel closed")
			}
			table := strings.TrimPrefix(d.RoutingKey, "public.")
			r.buf[table] = append(r.buf[table], d)
		}
	}
}

func (r *rabbitReader) bufLen() int {
	n := 0
	for _, msgs := range r.buf {
		n += len(msgs)
	}
	return n
}
