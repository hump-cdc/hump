package nats

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// fakeServer speaks just enough of the NATS client protocol for nats.go to
// connect, flush, publish and receive a reply: INFO, CONNECT, PING/PONG,
// SUB/UNSUB, PUB/HPUB and MSG. Each publish with a reply subject is answered
// by reply, if set.
type fakeServer struct {
	t     *testing.T
	ln    net.Listener
	reply func(subject string) []byte

	mu        sync.Mutex
	conns     []net.Conn
	published []fakeMsg
	connects  []string
}

// fakeMsg is one publish received by fakeServer.
type fakeMsg struct {
	subject string
	header  string
	payload string
}

// newFakeServer starts a fakeServer on a loopback port and stops it when the
// test ends. reply answers publishes that carry a reply subject; nil leaves
// them unanswered.
func newFakeServer(t *testing.T, reply func(subject string) []byte) *fakeServer {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	s := &fakeServer{t: t, ln: ln, reply: reply}
	go s.accept()
	t.Cleanup(s.close)

	return s
}

// url returns the nats:// URL of the server.
func (s *fakeServer) url() string {
	return "nats://" + s.ln.Addr().String()
}

// messages returns the publishes received so far.
func (s *fakeServer) messages() []fakeMsg {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]fakeMsg(nil), s.published...)
}

// connectOptions returns the CONNECT payloads received so far.
func (s *fakeServer) connectOptions() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]string(nil), s.connects...)
}

// close stops accepting and drops every client connection.
func (s *fakeServer) close() {
	_ = s.ln.Close()

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, c := range s.conns {
		_ = c.Close()
	}
}

// clientCount returns how many client connections the server has accepted.
func (s *fakeServer) clientCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return len(s.conns)
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

// serve runs the protocol loop for one client connection.
func (s *fakeServer) serve(conn net.Conn) {
	defer func() { _ = conn.Close() }()

	var writeMu sync.Mutex
	write := func(format string, args ...any) {
		writeMu.Lock()
		defer writeMu.Unlock()
		_, _ = fmt.Fprintf(conn, format, args...)
	}

	write("INFO {\"server_id\":\"fake\",\"version\":\"2.10.0\",\"proto\":1,\"headers\":true,\"max_payload\":1048576,\"nonce\":\"bm9uY2U\"}\r\n")

	subs := map[string]string{}
	r := bufio.NewReader(conn)
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		switch strings.ToUpper(fields[0]) {
		case "CONNECT":
			s.mu.Lock()
			s.connects = append(s.connects, strings.TrimSpace(line[len("CONNECT"):]))
			s.mu.Unlock()
		case "PING":
			write("PONG\r\n")
		case "SUB":
			subs[fields[1]] = fields[len(fields)-1]
		case "PUB", "HPUB":
			msg, reply, ok := readPublish(r, fields)
			if !ok {
				return
			}

			s.mu.Lock()
			s.published = append(s.published, msg)
			s.mu.Unlock()

			if reply == "" || s.reply == nil {
				continue
			}
			if sid, found := matchSub(subs, reply); found {
				body := s.reply(msg.subject)
				write("MSG %s %s %d\r\n%s\r\n", reply, sid, len(body), body)
			}
		}
	}
}

// readPublish reads the body of a PUB or HPUB whose control line split into
// fields, returning the message and its reply subject.
func readPublish(r *bufio.Reader, fields []string) (fakeMsg, string, bool) {
	hpub := strings.EqualFold(fields[0], "HPUB")

	sizes := 1
	if hpub {
		sizes = 2
	}
	args := fields[1:]
	if len(args) < 1+sizes {
		return fakeMsg{}, "", false
	}

	reply := ""
	if len(args) == 2+sizes {
		reply = args[1]
	}

	total, err := strconv.Atoi(args[len(args)-1])
	if err != nil {
		return fakeMsg{}, "", false
	}
	headerLen := 0
	if hpub {
		if headerLen, err = strconv.Atoi(args[len(args)-2]); err != nil {
			return fakeMsg{}, "", false
		}
	}

	buf := make([]byte, total+2)
	if _, err := io.ReadFull(r, buf); err != nil {
		return fakeMsg{}, "", false
	}

	return fakeMsg{subject: args[0], header: string(buf[:headerLen]), payload: string(buf[headerLen:total])}, reply, true
}

// matchSub returns the sid of the subscription in subs whose subject matches
// subject, honouring a trailing "*" or ">" wildcard.
func matchSub(subs map[string]string, subject string) (string, bool) {
	for pattern, sid := range subs {
		switch {
		case pattern == subject:
			return sid, true
		case strings.HasSuffix(pattern, ".*") || strings.HasSuffix(pattern, ".>"):
			if strings.HasPrefix(subject, pattern[:len(pattern)-1]) {
				return sid, true
			}
		}
	}

	return "", false
}
