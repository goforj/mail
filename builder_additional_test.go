package mail_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/goforj/mail"
	"github.com/goforj/mail/mailfake"
)

func TestBuilderFluentMethodsPopulateMessage(t *testing.T) {
	msg := mail.New(mailfake.New()).Message().
		From("team@example.com", "Example Team").
		ReplyTo("support@example.com", "Support").
		To("alice@example.com", "Alice").
		Cc("manager@example.com", "Manager").
		Bcc("audit@example.com", "Audit").
		Subject("Welcome").
		HTML("<p>hello</p>").
		Text("hello").
		Header("X-Request-ID", "req_123").
		Tag("welcome").
		Metadata("tenant_id", "tenant_123").
		Message()

	if msg.From == nil || msg.From.Email != "team@example.com" {
		t.Fatalf("from = %#v", msg.From)
	}
	if len(msg.ReplyTo) != 1 || msg.ReplyTo[0].Email != "support@example.com" {
		t.Fatalf("reply-to = %#v", msg.ReplyTo)
	}
	if len(msg.To) != 1 || msg.To[0].Email != "alice@example.com" {
		t.Fatalf("to = %#v", msg.To)
	}
	if len(msg.Cc) != 1 || msg.Cc[0].Email != "manager@example.com" {
		t.Fatalf("cc = %#v", msg.Cc)
	}
	if len(msg.Bcc) != 1 || msg.Bcc[0].Email != "audit@example.com" {
		t.Fatalf("bcc = %#v", msg.Bcc)
	}
	if got, want := msg.Subject, "Welcome"; got != want {
		t.Fatalf("subject = %q, want %q", got, want)
	}
	if got, want := msg.HTML, "<p>hello</p>"; got != want {
		t.Fatalf("html = %q, want %q", got, want)
	}
	if got, want := msg.Text, "hello"; got != want {
		t.Fatalf("text = %q, want %q", got, want)
	}
	if got, want := msg.Headers["X-Request-ID"], "req_123"; got != want {
		t.Fatalf("header = %q, want %q", got, want)
	}
	if len(msg.Tags) != 1 || msg.Tags[0] != "welcome" {
		t.Fatalf("tags = %#v", msg.Tags)
	}
	if got, want := msg.Metadata["tenant_id"], "tenant_123"; got != want {
		t.Fatalf("metadata = %q, want %q", got, want)
	}
}

func TestBuilderMessageReturnsClone(t *testing.T) {
	builder := mail.New(mailfake.New()).Message().
		To("alice@example.com", "Alice").
		Subject("Welcome").
		Text("hello").
		Header("X-Test", "one").
		Tag("welcome").
		Metadata("env", "test")

	msg := builder.Message()
	msg.Subject = "Changed"
	msg.Headers["X-Test"] = "two"
	msg.Tags[0] = "mutated"
	msg.Metadata["env"] = "prod"

	next := builder.Message()
	if got, want := next.Subject, "Welcome"; got != want {
		t.Fatalf("subject = %q, want %q", got, want)
	}
	if got, want := next.Headers["X-Test"], "one"; got != want {
		t.Fatalf("header = %q, want %q", got, want)
	}
	if got, want := next.Tags[0], "welcome"; got != want {
		t.Fatalf("tag = %q, want %q", got, want)
	}
	if got, want := next.Metadata["env"], "test"; got != want {
		t.Fatalf("metadata = %q, want %q", got, want)
	}
}

func TestBuilderBuildWithoutMailerValidates(t *testing.T) {
	msg, err := (&mail.MessageBuilder{}).
		To("alice@example.com", "Alice").
		Subject("Welcome").
		Text("hello world").
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if got, want := msg.Subject, "Welcome"; got != want {
		t.Fatalf("subject = %q, want %q", got, want)
	}

	_, err = (&mail.MessageBuilder{}).
		Subject("Welcome").
		Text("hello world").
		Build()
	if !errors.Is(err, mail.ErrMissingRecipient) {
		t.Fatalf("Build() error = %v, want %v", err, mail.ErrMissingRecipient)
	}
}

func TestBuilderBuildReturnsAttachmentError(t *testing.T) {
	_, err := mail.New(mailfake.New()).Message().
		To("alice@example.com", "Alice").
		Subject("Welcome").
		Text("hello world").
		AttachFile("missing-file.txt").
		Build()
	if err == nil {
		t.Fatal("Build() should return the attachment load error")
	}
}

func TestMailerSendWithoutDriverFails(t *testing.T) {
	mailer := mail.New(nil)
	err := mailer.Send(context.Background(), mail.Message{
		To:      []mail.Recipient{{Email: "alice@example.com"}},
		Subject: "Welcome",
		Text:    "hello world",
	})
	if !errors.Is(err, mail.ErrMissingMailer) {
		t.Fatalf("Send() error = %v, want %v", err, mail.ErrMissingMailer)
	}
}

func TestAttachmentFromPathFallbackContentType(t *testing.T) {
	if err := os.WriteFile("attachment.unknownext", []byte("hello"), 0o644); err != nil {
		t.Fatalf("write temp attachment: %v", err)
	}
	defer os.Remove("attachment.unknownext")

	attachment, err := mail.AttachmentFromPath("attachment.unknownext")
	if err != nil {
		t.Fatalf("AttachmentFromPath() error = %v", err)
	}
	if got, want := attachment.ContentType, "application/octet-stream"; got != want {
		t.Fatalf("content type = %q, want %q", got, want)
	}
}
