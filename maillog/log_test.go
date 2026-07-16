package maillog_test

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/goforj/mail"
	"github.com/goforj/mail/maillog"
)

// TestMailerWritesJSONLogEntry ensures log delivery emits structured metadata without message bodies by default.
func TestMailerWritesJSONLogEntry(t *testing.T) {
	var buffer bytes.Buffer
	now := time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC)
	mailer := maillog.New(&buffer, maillog.WithBodies(true), maillog.WithNow(func() time.Time { return now }))

	message := mail.Message{
		From:    &mail.Recipient{Email: "no-reply@example.com", Name: "Example"},
		To:      []mail.Recipient{{Email: "alice@example.com", Name: "Alice"}},
		Subject: "Welcome",
		Text:    "hello world",
		HTML:    "<p>hello world</p>",
		Metadata: map[string]string{
			"request_id": "abc123",
		},
	}
	if err := mailer.Send(context.Background(), message); err != nil {
		t.Fatalf("send log mail: %v", err)
	}

	var entry map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buffer.Bytes()), &entry); err != nil {
		t.Fatalf("unmarshal log entry: %v", err)
	}
	if entry["subject"] != "Welcome" {
		t.Fatalf("subject = %#v, want %q", entry["subject"], "Welcome")
	}
	if entry["text"] != "hello world" {
		t.Fatalf("text = %#v, want %q", entry["text"], "hello world")
	}
	if entry["html"] != "<p>hello world</p>" {
		t.Fatalf("html = %#v, want %q", entry["html"], "<p>hello world</p>")
	}
	if entry["sent_at"] != now.Format(time.RFC3339) {
		t.Fatalf("sent_at = %#v, want %q", entry["sent_at"], now.Format(time.RFC3339))
	}
}

// TestNewPanicsWithNilWriter ensures a required log destination fails fast during wiring.
func TestNewPanicsWithNilWriter(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("New(nil) should panic")
		}
	}()
	maillog.New(nil)
}

// TestNewPanicsWithTypedNilWriter ensures typed-nil log destinations cannot bypass fail-fast wiring.
func TestNewPanicsWithTypedNilWriter(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("New(typed nil) should panic")
		}
	}()
	var writer *bytes.Buffer
	maillog.New(writer)
}

// TestWithNowPanicsWithNilSource ensures deterministic clock injection remains a required collaborator when selected.
func TestWithNowPanicsWithNilSource(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("WithNow(nil) should panic")
		}
	}()
	maillog.WithNow(nil)
}

// TestDriverSerializesConcurrentWrites ensures concurrent messages cannot interleave JSON records.
func TestDriverSerializesConcurrentWrites(t *testing.T) {
	var buffer bytes.Buffer
	driver := maillog.New(&buffer)
	message := mail.Message{
		From:    &mail.Recipient{Email: "no-reply@example.com"},
		To:      []mail.Recipient{{Email: "alice@example.com"}},
		Subject: "Welcome",
		Text:    "hello world",
	}

	const sends = 50
	var wait sync.WaitGroup
	wait.Add(sends)
	for range sends {
		go func() {
			defer wait.Done()
			if err := driver.Send(context.Background(), message); err != nil {
				t.Errorf("Send() error = %v", err)
			}
		}()
	}
	wait.Wait()

	if got, want := len(strings.Split(strings.TrimSpace(buffer.String()), "\n")), sends; got != want {
		t.Fatalf("log lines = %d, want %d", got, want)
	}
}
