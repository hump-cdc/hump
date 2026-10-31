package rabbitmq

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"sync"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
)

// AMQP 0.9.1 frame types and the class/method ids fakeServer handles.
const (
	frameMethod = 1
	frameHeader = 2
	frameBody   = 3
	frameEnd    = 0xCE

	classConnection = 10
	classChannel    = 20
	classExchange   = 40
	classBasic      = 60
	classConfirm    = 85
)

// publishOutcome is how fakeServer answers a basic.publish.
type publishOutcome int

const (
	// outcomeAck acknowledges the publish.
	outcomeAck publishOutcome = iota
	// outcomeNack rejects the publish.
	outcomeNack
	// outcomeReturn returns the publish as unroutable, then acknowledges it.
	outcomeReturn
)

// fakeServer speaks just enough AMQP 0.9.1 for amqp091-go to connect, open a
// confirm-mode channel, declare exchanges and publish. Its hooks decide the
// outcome of each declare and publish.
type fakeServer struct {
	t  *testing.T
	ln net.Listener

	mu            sync.Mutex
	conns         []net.Conn
	declare       func(exchange string, passive bool) uint16 // declare returns 0 to accept, or the reply code to close the channel with.
	publish       func(exchange, routingKey string) publishOutcome
	failChannel   bool // failChannel closes every channel.open with 504.
	failConfirm   bool // failConfirm closes every confirm.select with 540.
	rejectConnect bool // rejectConnect drops a connection before the handshake.
	rejected      int
	declares      []string
	published     []string
	handshakes    []string
}

// newFakeServer starts a fakeServer on a loopback port that accepts every
// declare and acknowledges every publish, and stops it when the test ends.
func newFakeServer(t *testing.T) *fakeServer {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	s := &fakeServer{
		t:       t,
		ln:      ln,
		declare: func(string, bool) uint16 { return 0 },
		publish: func(string, string) publishOutcome { return outcomeAck },
	}
	go s.accept()
	t.Cleanup(s.close)

	return s
}

// url returns the amqp:// URL of the server.
func (s *fakeServer) url() string {
	return "amqp://guest:guest@" + s.ln.Addr().String() + "/"
}

// set runs fn under the server lock, for changing hooks and flags safely.
func (s *fakeServer) set(fn func(s *fakeServer)) {
	s.mu.Lock()
	defer s.mu.Unlock()

	fn(s)
}

// declared returns the exchange declarations received so far, each as
// "passive:<name>" or "declare:<name>".
func (s *fakeServer) declared() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]string(nil), s.declares...)
}

// publishes returns the "<exchange>/<routingKey>:<body>" of each publish
// received so far.
func (s *fakeServer) publishes() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]string(nil), s.published...)
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
		reject := s.rejectConnect
		if reject {
			s.rejected++
		} else {
			s.conns = append(s.conns, conn)
		}
		s.mu.Unlock()

		if reject {
			_ = conn.Close()
			continue
		}

		go s.serve(conn)
	}
}

// serve runs the protocol for one client connection.
func (s *fakeServer) serve(conn net.Conn) {
	defer func() { _ = conn.Close() }()

	r := bufio.NewReader(conn)
	if _, err := io.ReadFull(r, make([]byte, 8)); err != nil {
		return
	}

	w := &frameWriter{conn: conn}
	w.method(0, classConnection, 10, octet(0), octet(9), table(), longstr("PLAIN"), longstr("en_US"))

	tags := map[uint16]uint64{}
	for {
		typ, channel, payload, err := readFrame(r)
		if err != nil || typ != frameMethod {
			if err != nil {
				return
			}
			continue
		}

		class, method := binary.BigEndian.Uint16(payload), binary.BigEndian.Uint16(payload[2:])
		args := payload[4:]

		switch [2]uint16{class, method} {
		case [2]uint16{classConnection, 11}: // start-ok
			s.set(func(s *fakeServer) { s.handshakes = append(s.handshakes, string(args)) })
			w.method(0, classConnection, 30, short(0), long(131072), short(0))
		case [2]uint16{classConnection, 40}: // open
			s.set(func(s *fakeServer) { s.handshakes = append(s.handshakes, string(args)) })
			w.method(0, classConnection, 41, shortstr(""))
		case [2]uint16{classConnection, 50}: // close
			w.method(0, classConnection, 51)
			return
		case [2]uint16{classChannel, 10}: // open
			if s.flag(func(s *fakeServer) bool { return s.failChannel }) {
				w.closeChannel(channel, amqp.ChannelError, class, method)
				continue
			}
			w.method(channel, classChannel, 11, longstr(""))
		case [2]uint16{classChannel, 40}: // close
			w.method(channel, classChannel, 41)
		case [2]uint16{classConfirm, 10}: // select
			if s.flag(func(s *fakeServer) bool { return s.failConfirm }) {
				w.closeChannel(channel, amqp.NotImplemented, class, method)
				continue
			}
			tags[channel] = 0
			w.method(channel, classConfirm, 11)
		case [2]uint16{classExchange, 10}: // declare
			s.handleDeclare(w, channel, args)
		case [2]uint16{classBasic, 40}: // publish
			tags[channel]++
			if !s.handlePublish(r, w, channel, args, tags[channel]) {
				return
			}
		}
	}
}

// flag reads a server flag under the lock.
func (s *fakeServer) flag(get func(s *fakeServer) bool) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return get(s)
}

// handleDeclare answers an exchange.declare through the declare hook.
func (s *fakeServer) handleDeclare(w *frameWriter, channel uint16, args []byte) {
	name, rest := readShortstr(args[2:])
	_, rest = readShortstr(rest)
	passive := rest[0]&1 == 1

	s.mu.Lock()
	kind := "declare:"
	if passive {
		kind = "passive:"
	}
	s.declares = append(s.declares, kind+name)
	declare := s.declare
	s.mu.Unlock()

	if code := declare(name, passive); code != 0 {
		w.closeChannel(channel, code, classExchange, 10)
		return
	}
	w.method(channel, classExchange, 11)
}

// handlePublish reads the content of a basic.publish and answers it through
// the publish hook. It reports false when the connection broke.
func (s *fakeServer) handlePublish(r *bufio.Reader, w *frameWriter, channel uint16, args []byte, tag uint64) bool {
	exchange, rest := readShortstr(args[2:])
	routingKey, _ := readShortstr(rest)

	_, _, header, err := readFrame(r)
	if err != nil {
		return false
	}
	size := binary.BigEndian.Uint64(header[4:])

	var body []byte
	for uint64(len(body)) < size {
		_, _, chunk, err := readFrame(r)
		if err != nil {
			return false
		}
		body = append(body, chunk...)
	}

	s.mu.Lock()
	s.published = append(s.published, exchange+"/"+routingKey+":"+string(body))
	publish := s.publish
	s.mu.Unlock()

	switch publish(exchange, routingKey) {
	case outcomeNack:
		w.method(channel, classBasic, 120, longlong(tag), octet(0))
	case outcomeReturn:
		w.method(channel, classBasic, 50, short(amqp.NoRoute), shortstr("NO_ROUTE"), shortstr(exchange), shortstr(routingKey))
		w.frame(frameHeader, channel, concat(short(classBasic), short(0), longlong(0), short(0)))
		w.method(channel, classBasic, 80, longlong(tag), octet(0))
	case outcomeAck:
		w.method(channel, classBasic, 80, longlong(tag), octet(0))
	}

	return true
}

// frameWriter serialises frames onto one connection.
type frameWriter struct {
	mu   sync.Mutex
	conn net.Conn
}

// method writes a method frame of class/method with the encoded args.
func (w *frameWriter) method(channel, class, method uint16, args ...[]byte) {
	w.frame(frameMethod, channel, concat(append([][]byte{short(class), short(method)}, args...)...))
}

// closeChannel closes channel with code, blaming class/method.
func (w *frameWriter) closeChannel(channel, code, class, method uint16) {
	w.method(channel, classChannel, 40, short(code), shortstr("closed by fake server"), short(class), short(method))
}

// frame writes one frame of typ on channel.
func (w *frameWriter) frame(typ byte, channel uint16, payload []byte) {
	w.mu.Lock()
	defer w.mu.Unlock()

	_, _ = w.conn.Write(concat(octet(typ), short(channel), long(uint32(len(payload))), payload, octet(frameEnd)))
}

// readFrame reads one frame and returns its type, channel and payload.
func readFrame(r *bufio.Reader) (byte, uint16, []byte, error) {
	head := make([]byte, 7)
	if _, err := io.ReadFull(r, head); err != nil {
		return 0, 0, nil, err
	}

	payload := make([]byte, binary.BigEndian.Uint32(head[3:])+1)
	if _, err := io.ReadFull(r, payload); err != nil {
		return 0, 0, nil, err
	}

	return head[0], binary.BigEndian.Uint16(head[1:]), payload[:len(payload)-1], nil
}

// readShortstr decodes a shortstr from b and returns it with the rest of b.
func readShortstr(b []byte) (string, []byte) {
	n := int(b[0])
	return string(b[1 : 1+n]), b[1+n:]
}

// octet encodes v as one byte.
func octet(v byte) []byte { return []byte{v} }

// short encodes v as a big-endian uint16.
func short(v uint16) []byte { return binary.BigEndian.AppendUint16(nil, v) }

// long encodes v as a big-endian uint32.
func long(v uint32) []byte { return binary.BigEndian.AppendUint32(nil, v) }

// longlong encodes v as a big-endian uint64.
func longlong(v uint64) []byte { return binary.BigEndian.AppendUint64(nil, v) }

// shortstr encodes s with a one-byte length prefix.
func shortstr(s string) []byte { return append(octet(byte(len(s))), s...) }

// longstr encodes s with a four-byte length prefix.
func longstr(s string) []byte { return append(long(uint32(len(s))), s...) }

// table encodes an empty field table.
func table() []byte { return long(0) }

// concat joins byte slices.
func concat(parts ...[]byte) []byte { return bytes.Join(parts, nil) }
