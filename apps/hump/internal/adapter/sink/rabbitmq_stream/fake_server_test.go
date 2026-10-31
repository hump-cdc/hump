package rabbitmq_stream

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"strconv"
	"sync"
	"testing"
	"time"
)

// RabbitMQ Streams protocol command keys and response codes fakeServer uses.
const (
	keyDeclarePublisher = 1
	keyPublish          = 2
	keyPublishConfirm   = 3
	keyPublishError     = 4
	keyDeletePublisher  = 6
	keyCreateStream     = 13
	keyMetadata         = 15
	keyPeerProperties   = 17
	keySaslHandshake    = 18
	keySaslAuthenticate = 19
	keyTune             = 20
	keyOpen             = 21

	responseFlag = 0x8000

	codeOK                 = 1
	codeStreamDoesNotExist = 2
	codeStreamExists       = 5
	codeStreamNotAvailable = 6
	codeAccessRefused      = 16
	codePreconditionFailed = 17
)

// publishOutcome is how fakeServer answers a publish.
type publishOutcome int

const (
	// outcomeConfirm confirms every message of the publish.
	outcomeConfirm publishOutcome = iota
	// outcomeError rejects every message of the publish.
	outcomeError
	// outcomeSilent never answers the publish.
	outcomeSilent
)

// fakeServer speaks just enough of the RabbitMQ Streams protocol for the
// stream client to open an environment, query metadata, create streams,
// declare a publisher and publish. Its hooks decide each answer.
type fakeServer struct {
	t  *testing.T
	ln net.Listener

	mu         sync.Mutex
	conns      []net.Conn
	metadata   func(stream string) uint16 // metadata returns the response code for stream.
	create     func(stream string) uint16 // create returns the response code of a create.
	publish    func() publishOutcome      // publish decides the answer to each publish.
	delay      time.Duration              // delay holds every create back.
	creates    []string                   // creates lists the created streams in order.
	published  []string                   // published lists the message bodies received.
	declared   int                        // declared counts publisher declarations.
	handshakes []string
}

// newFakeServer starts a fakeServer on a loopback port where every stream
// exists and every publish is confirmed, and stops it when the test ends.
func newFakeServer(t *testing.T) *fakeServer {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	s := &fakeServer{
		t:        t,
		ln:       ln,
		metadata: func(string) uint16 { return codeOK },
		create:   func(string) uint16 { return codeOK },
		publish:  func() publishOutcome { return outcomeConfirm },
	}
	go s.accept()
	t.Cleanup(s.close)

	return s
}

// url returns the rabbitmq-stream:// URL of the server.
func (s *fakeServer) url() string {
	return "rabbitmq-stream://guest:guest@" + s.ln.Addr().String() + "/"
}

// set runs fn under the server lock, for changing hooks safely.
func (s *fakeServer) set(fn func(s *fakeServer)) {
	s.mu.Lock()
	defer s.mu.Unlock()

	fn(s)
}

// snapshot returns the created streams, published bodies and publisher
// declarations seen so far.
func (s *fakeServer) snapshot() (creates, published []string, declared int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]string(nil), s.creates...), append([]string(nil), s.published...), s.declared
}

// dropClients closes every client connection while the listener stays open.
func (s *fakeServer) dropClients() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, c := range s.conns {
		_ = c.Close()
	}
	s.conns = nil
}

// close stops accepting and drops every client connection.
func (s *fakeServer) close() {
	_ = s.ln.Close()
	s.dropClients()
}

// accept serves every incoming connection until the listener closes.
func (s *fakeServer) accept() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return
		}

		s.mu.Lock()
		s.conns = append(s.conns, conn)
		s.mu.Unlock()

		go s.serve(conn)
	}
}

// serve answers the frames of one client connection.
func (s *fakeServer) serve(conn net.Conn) {
	defer func() { _ = conn.Close() }()

	host, port, _ := net.SplitHostPort(s.ln.Addr().String())
	portNum, _ := strconv.Atoi(port)
	w := &frameWriter{conn: conn}
	r := bufio.NewReader(conn)

	for {
		frame, err := readFrame(r)
		if err != nil {
			return
		}
		key, body := binary.BigEndian.Uint16(frame), frame[4:]

		switch key {
		case keyPeerProperties:
			w.respond(key, body, codeOK, u32(0))
		case keySaslHandshake:
			w.respond(key, body, codeOK, u32(1), str("PLAIN"))
		case keySaslAuthenticate:
			s.set(func(s *fakeServer) { s.handshakes = append(s.handshakes, string(body)) })
			w.respond(key, body, codeOK)
			w.frame(u16(keyTune), u16(1), u32(1048576), u32(0))
		case keyOpen:
			s.set(func(s *fakeServer) { s.handshakes = append(s.handshakes, string(body)) })
			w.respond(key, body, codeOK, u32(2), str("advertised_host"), str(host), str("advertised_port"), str(port))
		case keyMetadata:
			s.handleMetadata(w, body, host, portNum)
		case keyCreateStream:
			s.handleCreate(w, body)
		case keyDeclarePublisher:
			s.set(func(s *fakeServer) { s.declared++ })
			w.respond(key, body, codeOK)
		case keyDeletePublisher:
			w.respond(key, body, codeOK)
		case keyPublish:
			s.handlePublish(w, body)
		}
	}
}

// handleMetadata answers a metadata query with this server as the leader of
// every stream, coded by the metadata hook.
func (s *fakeServer) handleMetadata(w *frameWriter, body []byte, host string, port int) {
	corr, rest := body[:4], body[4:]
	count := int(binary.BigEndian.Uint32(rest))
	rest = rest[4:]

	s.mu.Lock()
	metadata := s.metadata
	s.mu.Unlock()

	parts := [][]byte{u16(keyMetadata | responseFlag), u16(1), corr, u32(1), u16(0), str(host), u32(uint32(port)), u32(uint32(count))}
	for range count {
		var name string
		name, rest = readStr(rest)
		parts = append(parts, str(name), u16(metadata(name)), u16(0), u32(0))
	}
	w.frame(parts...)
}

// handleCreate answers a create stream through the create hook.
func (s *fakeServer) handleCreate(w *frameWriter, body []byte) {
	name, _ := readStr(body[4:])

	s.mu.Lock()
	s.creates = append(s.creates, name)
	create, delay := s.create, s.delay
	s.mu.Unlock()

	time.Sleep(delay)
	w.respond(keyCreateStream, body, create(name))
}

// handlePublish records the messages of a publish and answers it through the
// publish hook.
func (s *fakeServer) handlePublish(w *frameWriter, body []byte) {
	publisher := body[0]
	count := int(binary.BigEndian.Uint32(body[1:]))
	rest := body[5:]

	ids := make([][]byte, 0, count)
	var bodies []string
	for range count {
		ids = append(ids, rest[:8])
		size := int(binary.BigEndian.Uint32(rest[8:]))
		bodies = append(bodies, string(rest[12:12+size]))
		rest = rest[12+size:]
	}

	s.mu.Lock()
	s.published = append(s.published, bodies...)
	publish := s.publish
	s.mu.Unlock()

	switch publish() {
	case outcomeConfirm:
		w.frame(append([][]byte{u16(keyPublishConfirm), u16(1), {publisher}, u32(uint32(count))}, ids...)...)
	case outcomeError:
		parts := [][]byte{u16(keyPublishError), u16(1), {publisher}, u32(uint32(count))}
		for _, id := range ids {
			parts = append(parts, id, u16(codeAccessRefused))
		}
		w.frame(parts...)
	case outcomeSilent:
	}
}

// frameWriter serialises frames onto one connection.
type frameWriter struct {
	mu   sync.Mutex
	conn net.Conn
}

// respond answers the request key whose body starts with its correlation id.
func (w *frameWriter) respond(key uint16, body []byte, code uint16, parts ...[]byte) {
	w.frame(append([][]byte{u16(key | responseFlag), u16(1), body[:4], u16(code)}, parts...)...)
}

// frame writes the concatenated parts as one size-prefixed frame.
func (w *frameWriter) frame(parts ...[]byte) {
	payload := bytes.Join(parts, nil)

	w.mu.Lock()
	defer w.mu.Unlock()

	_, _ = w.conn.Write(append(u32(uint32(len(payload))), payload...))
}

// readFrame reads one size-prefixed frame.
func readFrame(r *bufio.Reader) ([]byte, error) {
	head := make([]byte, 4)
	if _, err := io.ReadFull(r, head); err != nil {
		return nil, err
	}

	frame := make([]byte, binary.BigEndian.Uint32(head))
	if _, err := io.ReadFull(r, frame); err != nil {
		return nil, err
	}

	return frame, nil
}

// readStr decodes a protocol string from b and returns it with the rest of b.
func readStr(b []byte) (string, []byte) {
	n := int(binary.BigEndian.Uint16(b))
	return string(b[2 : 2+n]), b[2+n:]
}

// u16 encodes v as a big-endian uint16.
func u16(v uint16) []byte { return binary.BigEndian.AppendUint16(nil, v) }

// u32 encodes v as a big-endian uint32.
func u32(v uint32) []byte { return binary.BigEndian.AppendUint32(nil, v) }

// str encodes s as a protocol string: an int16 length and the bytes.
func str(s string) []byte { return append(u16(uint16(len(s))), s...) }
