package postgres

import (
	"context"
	"encoding/binary"
	"strconv"
	"testing"
	"time"

	"hump/pkg/messageid"
)

func TestWalIdentitySurvivesReplayAndDistinguishesChanges(t *testing.T) {
	w, _ := newTestWal(t, usersServer(t), configuredTable("public", "users"))
	registerRelation(t, w, 1, "public", "users", idColumn())
	recorder := newHandlerRecorder(nil)
	commitTime := time.Date(2026, 1, 2, 3, 4, 5, 123000, time.UTC)
	for replay := range 2 {
		serverTime := commitTime.Add(time.Duration(replay+1) * time.Hour)
		processIdentityMessages(t, w, recorder, serverTime,
			identityBegin(0x100, commitTime),
			insertMessage(1, textColumn("7")),
			insertMessage(1, textColumn("7")),
			commitMessage(0x180),
		)
	}
	processIdentityMessages(t, w, recorder, time.Now(), identityBegin(0x200, commitTime), insertMessage(1, textColumn("7")))
	calls := recorder.recorded()
	for i, call := range calls {
		if !call.timestamp.Equal(commitTime) {
			t.Errorf("call %d timestamp = %s, want %s", i, call.timestamp, commitTime)
		}
		if len(call.metadata.MessageID) != 64 || call.metadata.SnapshotID != "" {
			t.Errorf("call %d metadata = %+v", i, call.metadata)
		}
	}
	if calls[0].metadata != calls[2].metadata || calls[1].metadata != calls[3].metadata {
		t.Error("replayed changes have different identities")
	}
	if calls[0].metadata == calls[1].metadata || calls[0].metadata == calls[4].metadata {
		t.Error("different ordinal or FinalLSN reused an identity")
	}
}

func TestWalOrdinalCountsFilteredChangesButNotServiceMessages(t *testing.T) {
	w, _ := newTestWal(t, usersServer(t), configuredTable("public", "users"))
	registerRelation(t, w, 1, "public", "users", idColumn())
	registerRelation(t, w, 2, "public", "audit", idColumn())
	recorder := newHandlerRecorder(nil)
	processIdentityMessages(t, w, recorder, time.Now(),
		identityBegin(0x100, time.Now()),
		insertMessage(2, textColumn("1")),
		updateMessage(2, nil, []tupleColumn{textColumn("1")}),
		deleteMessage(2, textColumn("1")),
		truncateMessage(2),
		logicalDecodingMessage(true, "trace", "abc"),
		relationMessage(1, "public", "users", idColumn()),
		insertMessage(1, textColumn("7")),
	)
	calls := recorder.recorded()
	if len(calls) != 1 {
		t.Fatalf("handler calls = %d, want 1", len(calls))
	}
	want := messageid.Hash("WAL", "db", "public", "users", "0/100", "5")
	if calls[0].metadata.MessageID != want {
		t.Errorf("MessageID = %s, want %s", calls[0].metadata.MessageID, want)
	}
}

func TestWalTruncateUsesTableNamesAndOneChangeOrdinal(t *testing.T) {
	w, _ := newTestWal(t, twoTableServer(t), configuredTable("public", "users"), configuredTable("public", "orders"))
	registerRelation(t, w, 1, "public", "users", idColumn())
	registerRelation(t, w, 2, "public", "orders", idColumn())
	recorder := newHandlerRecorder(nil)
	commitTime := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	processIdentityMessages(t, w, recorder, time.Now(), identityBegin(0x100, commitTime), truncateMessage(1, 2), insertMessage(1, textColumn("7")))
	calls := recorder.recorded()
	if len(calls) != 3 {
		t.Fatalf("handler calls = %d, want 3", len(calls))
	}
	for i, call := range calls {
		ordinal := 1
		if i == 2 {
			ordinal = 2
		}
		want := messageid.Hash("WAL", "db", "public", string(call.table), "0/100", strconv.Itoa(ordinal))
		if call.metadata.MessageID != want || !call.timestamp.Equal(commitTime) {
			t.Errorf("call %d metadata = %+v, timestamp = %s", i, call.metadata, call.timestamp)
		}
	}
	if calls[0].metadata == calls[1].metadata {
		t.Error("truncated tables reused an identity")
	}
}

func TestWalTransactionStateResetsOnBeginCommitAndStart(t *testing.T) {
	for _, boundary := range []string{"begin", "commit", "start"} {
		t.Run(boundary, func(t *testing.T) {
			server := usersServer(t)
			w, _ := newReplicationWal(t, server, newTestConnection(t, server, withReplication()), configuredTable("public", "users"))
			w.inTransaction = true
			w.commitTime = time.Now()
			w.finalLSN = 99
			w.changeOrdinal = 12
			w.messageContext.append("stale", "value")
			var wantTime time.Time
			switch boundary {
			case "begin":
				wantTime = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
				processIdentityMessages(t, w, newHandlerRecorder(nil), time.Now(), identityBegin(0, wantTime))
			case "commit":
				processIdentityMessages(t, w, newHandlerRecorder(nil), time.Now(), commitMessage(0))
			case "start":
				server.failReplicationStart("42704", "missing slot")
				if err := w.Start(context.Background(), failingHandler(t)); err == nil {
					t.Fatal("Start() unexpectedly succeeded")
				}
			}
			if !w.commitTime.Equal(wantTime) || w.finalLSN != 0 || w.changeOrdinal != 0 || len(w.messageContext.dump()) != 0 {
				t.Errorf("transaction state survived %s", boundary)
			}
			if w.inTransaction != (boundary == "begin") {
				t.Errorf("inTransaction = %v after %s", w.inTransaction, boundary)
			}
		})
	}
}

// identityBegin encodes the commit identity carried by pgoutput Begin.
func identityBegin(lsn uint64, commitTime time.Time) []byte {
	data := beginMessage()
	binary.BigEndian.PutUint64(data[1:9], lsn)
	micros := commitTime.UnixMicro() - time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC).UnixMicro()
	binary.BigEndian.PutUint64(data[9:17], uint64(micros))
	return data
}

// processIdentityMessages dispatches a sequence with a deliberately independent server timestamp.
func processIdentityMessages(t *testing.T, w *wal, recorder *handlerRecorder, serverTime time.Time, messages ...[]byte) {
	t.Helper()
	for _, message := range messages {
		if _, err := w.processMessage(context.Background(), message, serverTime, recorder.handle); err != nil {
			t.Fatalf("processMessage(%c) error = %v", message[0], err)
		}
	}
}
