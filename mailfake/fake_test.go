package mailfake_test

import (
	"context"
	"errors"
	"testing"

	"github.com/goforj/mail"
	"github.com/goforj/mail/mailfake"
)

func TestDriverSetErrorResetMessagesAndLast(t *testing.T) {
	fake := mailfake.New()

	if _, ok := fake.Last(); ok {
		t.Fatal("Last() should report false before any sends")
	}
	if got := len(fake.Messages()); got != 0 {
		t.Fatalf("Messages() len = %d, want 0", got)
	}

	wantErr := errors.New("boom")
	fake.SetError(wantErr)
	err := fake.Send(context.Background(), mail.Message{
		To:      []mail.Recipient{{Email: "alice@example.com"}},
		Subject: "Welcome",
		Text:    "hello world",
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Send() error = %v, want %v", err, wantErr)
	}
	if got := fake.SentCount(); got != 0 {
		t.Fatalf("SentCount() = %d, want 0", got)
	}

	fake.Reset()
	if got := fake.SentCount(); got != 0 {
		t.Fatalf("SentCount() after reset = %d, want 0", got)
	}

	original := mail.Message{
		To:       []mail.Recipient{{Email: "alice@example.com", Name: "Alice"}},
		Subject:  "Welcome",
		Text:     "hello world",
		Headers:  map[string]string{"X-Test": "one"},
		Metadata: map[string]string{"env": "test"},
		Tags:     []string{"welcome"},
	}
	if err := fake.Send(context.Background(), original); err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	original.Subject = "Changed"
	original.Headers["X-Test"] = "two"
	original.Metadata["env"] = "prod"
	original.Tags[0] = "mutated"

	messages := fake.Messages()
	if got := len(messages); got != 1 {
		t.Fatalf("Messages() len = %d, want 1", got)
	}
	if got, want := messages[0].Subject, "Welcome"; got != want {
		t.Fatalf("subject = %q, want %q", got, want)
	}
	if got, want := messages[0].Headers["X-Test"], "one"; got != want {
		t.Fatalf("header = %q, want %q", got, want)
	}
	if got, want := messages[0].Metadata["env"], "test"; got != want {
		t.Fatalf("metadata = %q, want %q", got, want)
	}
	if got, want := messages[0].Tags[0], "welcome"; got != want {
		t.Fatalf("tag = %q, want %q", got, want)
	}

	last, ok := fake.Last()
	if !ok {
		t.Fatal("Last() should return the last sent message")
	}
	if got, want := last.Subject, "Welcome"; got != want {
		t.Fatalf("last subject = %q, want %q", got, want)
	}
}
