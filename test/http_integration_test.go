package test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path"
	"sync"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v7"
)

func TestPipeline_HTTP_Integration(t *testing.T) {
	// Recorder server is started before the pipeline so its URL can be injected
	// into the sink config and no webhook request is missed. The URL is passed
	// through an env var read by the sink's {{ env }} template function.
	recorder := startRecorder(t)
	defer recorder.close()
	t.Setenv("HUMP_HTTP_TEST_URL", recorder.url())

	// Test setup
	ctx, db, teardown, wait, err := setup(t,
		withDatabaseURL("postgres://hump:hump@localhost:5437/hump?sslmode=disable"),
		withConfig(`
sinks:
    - name: http1
      http:
          url: '{{ env "HUMP_HTTP_TEST_URL" }}/{{.Table}}'
          method: POST
          body: '{ {{if .Prev}}"before":{{.Prev | json}},{{end}}{{if .Next}}"after":{{.Next | json}},{{end}}"source":{"version":{{.Version | json}},"connector":"postgres","name":"hump","db":{{.Database | json}},"schema":{{.Schema | json}},"table":{{.Table | json}},"ts_ms":{{.Timestamp | unixepoch | mul 1000 | json}}},"op":{{.Operation | lower | trunc 1 | replace "i" "c" | json}},"ts_ms":{{now | unixepoch | mul 1000 | json}} }'
          headers:
              Content-Type: application/json
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
          mode: async
          maxConcurrent: 64
          concurrencyKey: '{{.Table}}'

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
          - preset: http1
`),
		withPubName(fmt.Sprintf("http_test_pub_%d", time.Now().UnixNano())),
		withSlotName(fmt.Sprintf("http_test_slot_%d", time.Now().UnixNano())),
	)
	if err != nil {
		t.Fatalf("Failed to setup test: %v", err)
	}

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

					p, req, err := recorder.next("users", 5*time.Second)
					if err != nil {
						t.Fatalf("Failed to receive webhook: %v", err)
					}
					// HTTP-specific surface: method, routed path, default header.
					if req.method != "POST" {
						t.Errorf("Expected method POST, got %s", req.method)
					}
					if req.path != "/users" {
						t.Errorf("Expected path /users, got %s", req.path)
					}
					if ct := req.headers.Get("Content-Type"); ct != "application/json" {
						t.Errorf("Expected Content-Type application/json, got %s", ct)
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

					p, _, err := recorder.next("posts", 5*time.Second)
					if err != nil {
						t.Fatalf("Failed to receive webhook: %v", err)
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

					p, _, err := recorder.next("comments", 5*time.Second)
					if err != nil {
						t.Fatalf("Failed to receive webhook: %v", err)
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

					p, _, err := recorder.next("events", 5*time.Second)
					if err != nil {
						t.Fatalf("Failed to receive webhook: %v", err)
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

					p, _, err := recorder.next("users", 5*time.Second)
					if err != nil {
						t.Fatalf("Failed to receive webhook: %v", err)
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

					p, _, err := recorder.next("posts", 5*time.Second)
					if err != nil {
						t.Fatalf("Failed to receive webhook: %v", err)
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

					p, _, err := recorder.next("comments", 5*time.Second)
					if err != nil {
						t.Fatalf("Failed to receive webhook: %v", err)
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

					p, _, err := recorder.next("events", 5*time.Second)
					if err != nil {
						t.Fatalf("Failed to receive webhook: %v", err)
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
	// records, desyncing the one-op/one-request assertion loop.
	t.Run("DELETE", func(t *testing.T) {
		t.Run("events", func(t *testing.T) {
			for _, eventID := range eventIDs {
				t.Run(fmt.Sprintf("Event %d", eventID), func(t *testing.T) {
					if err := deleteEvent(ctx, db, eventID); err != nil {
						t.Fatalf("Failed to delete event: %v", err)
					}
					p, _, err := recorder.next("events", 5*time.Second)
					if err != nil {
						t.Fatalf("Failed to receive webhook: %v", err)
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
					p, _, err := recorder.next("comments", 5*time.Second)
					if err != nil {
						t.Fatalf("Failed to receive webhook: %v", err)
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
					p, _, err := recorder.next("posts", 5*time.Second)
					if err != nil {
						t.Fatalf("Failed to receive webhook: %v", err)
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
					p, _, err := recorder.next("users", 5*time.Second)
					if err != nil {
						t.Fatalf("Failed to receive webhook: %v", err)
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
	// cascade adds a leftover request to its path. The recorder buffers those
	// leftovers harmlessly under their own table.
	t.Run("TRUNCATE", func(t *testing.T) {
		t.Run("events", func(t *testing.T) {
			if err := truncateEvents(ctx, db); err != nil {
				t.Fatalf("Failed to truncate events: %v", err)
			}
			assertTruncateHTTP(t, recorder, "events")
		})

		t.Run("comments", func(t *testing.T) {
			if err := truncateComments(ctx, db); err != nil {
				t.Fatalf("Failed to truncate comments: %v", err)
			}
			assertTruncateHTTP(t, recorder, "comments")
		})

		t.Run("posts", func(t *testing.T) {
			if err := truncatePosts(ctx, db); err != nil {
				t.Fatalf("Failed to truncate posts: %v", err)
			}
			assertTruncateHTTP(t, recorder, "posts")
		})

		t.Run("users", func(t *testing.T) {
			if err := truncateUsers(ctx, db); err != nil {
				t.Fatalf("Failed to truncate users: %v", err)
			}
			assertTruncateHTTP(t, recorder, "users")
		})
	})

	teardown()
	if err := wait(); err != nil {
		t.Fatalf("Failed to wait for pipeline shutdown: %v", err)
	}
}

// assertTruncateHTTP drains every truncate request on the table's path — a single
// TRUNCATE may emit more than one for the hash-partitioned `comments` and the
// `events` hypertable — and checks each carries op "t" with no row image.
func assertTruncateHTTP(t *testing.T, recorder *httpRecorder, table string) {
	t.Helper()

	// Poll until the whole stream goes idle (no new request on any path within
	// the quiet window), then take and remove the requests buffered for table.
	// Other tables' buffered requests — e.g. those emitted by a TRUNCATE ...
	// CASCADE on a parent table — are left in place for their own subtest.
	for {
		before := recorder.bufLen()
		recorder.wait(time.Second)
		if recorder.bufLen() == before {
			break
		}
	}
	records := recorder.take(table)

	if len(records) == 0 {
		t.Fatalf("Expected at least one truncate request for %s, got none", table)
	}
	for i, record := range records {
		p, err := payloadDecode(record.body)
		if err != nil {
			t.Fatalf("Failed to decode payload: %v", err)
		}
		if p.Op != "t" {
			t.Errorf("request %d: expected op t, got %s", i, p.Op)
		}
		if p.Source.Table != table {
			t.Errorf("request %d: expected source table %s, got %s", i, table, p.Source.Table)
		}
		if p.Before != nil {
			t.Errorf("request %d: expected no before image, got %v", i, p.Before)
		}
		if p.After != nil {
			t.Errorf("request %d: expected no after image, got %v", i, p.After)
		}
	}
}

// recordedRequest is a single webhook request captured by the recorder server.
type recordedRequest struct {
	method  string
	path    string
	headers http.Header
	body    []byte
}

// httpRecorder is an in-process HTTP server that buffers received webhook
// requests per table. The sink routes each table to a distinct URL path
// (/{{.Table}}), so the last path segment identifies the table — mirroring the
// per-topic buffering the Kafka test relies on. Default delivery serializes
// operations, so per-table requests arrive in produce order.
type httpRecorder struct {
	server *httptest.Server
	mu     sync.Mutex
	buf    map[string][]recordedRequest
}

// startRecorder starts a recorder server that returns 200 for every request.
func startRecorder(t *testing.T) *httpRecorder {
	t.Helper()

	r := &httpRecorder{buf: map[string][]recordedRequest{}}
	r.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		body, _ := io.ReadAll(req.Body)
		table := path.Base(req.URL.Path)

		r.mu.Lock()
		r.buf[table] = append(r.buf[table], recordedRequest{
			method:  req.Method,
			path:    req.URL.Path,
			headers: req.Header.Clone(),
			body:    body,
		})
		r.mu.Unlock()

		w.WriteHeader(http.StatusOK)
	}))
	return r
}

func (r *httpRecorder) url() string { return r.server.URL }

func (r *httpRecorder) close() { r.server.Close() }

// next returns the decoded CDC payload and the raw request of the next webhook
// for table in receive order, polling until one is available or timeout elapses.
func (r *httpRecorder) next(table string, timeout time.Duration) (payload, recordedRequest, error) {
	deadline := time.Now().Add(timeout)
	for {
		r.mu.Lock()
		if len(r.buf[table]) > 0 {
			rec := r.buf[table][0]
			r.buf[table] = r.buf[table][1:]
			r.mu.Unlock()
			p, err := payloadDecode(rec.body)
			if err != nil {
				return payload{}, rec, fmt.Errorf("decode payload for table %q: %w", table, err)
			}
			return p, rec, nil
		}
		r.mu.Unlock()

		if time.Now().After(deadline) {
			return payload{}, recordedRequest{}, fmt.Errorf("no request for table %q within %s", table, timeout)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// take removes and returns all buffered requests for table.
func (r *httpRecorder) take(table string) []recordedRequest {
	r.mu.Lock()
	defer r.mu.Unlock()
	records := r.buf[table]
	delete(r.buf, table)
	return records
}

// bufLen returns the total number of buffered requests across all tables.
func (r *httpRecorder) bufLen() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, recs := range r.buf {
		n += len(recs)
	}
	return n
}

// wait blocks for d to let in-flight requests settle into the buffer.
func (r *httpRecorder) wait(d time.Duration) { time.Sleep(d) }
