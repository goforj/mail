package maillog_test

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/goforj/mail"
	"github.com/goforj/mail/maillog"
)

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
