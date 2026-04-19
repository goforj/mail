package mail_test

import (
	"context"
	"errors"
	"testing"

	"github.com/goforj/mail"
	"github.com/goforj/mail/mailfake"
)

func TestMailerFluentSendAppliesDefaults(t *testing.T) {
	fake := mailfake.New()
	mailer := mail.New(
		fake,
		mail.WithDefaultFrom("no-reply@example.com", "Example"),
		mail.WithDefaultReplyTo(mail.Recipient{Email: "support@example.com", Name: "Support"}),
		mail.WithDefaultHeader("X-App", "goforj"),
		mail.WithDefaultTag("transactional"),
		mail.WithDefaultMetadata("env", "test"),
	)

	err := mailer.
		Message().
		To("alice@example.com", "Alice").
		Subject("Welcome").
		Text("hello world").
		Metadata("request_id", "abc123").
		Send(context.Background())
	if err != nil {
		t.Fatalf("send message: %v", err)
	}

	if fake.SentCount() != 1 {
		t.Fatalf("sent count = %d, want 1", fake.SentCount())
	}

	message, ok := fake.Last()
	if !ok {
		t.Fatal("expected last message")
	}
	if message.From == nil || message.From.Email != "no-reply@example.com" {
		t.Fatalf("from = %#v, want default from", message.From)
	}
	if len(message.ReplyTo) != 1 || message.ReplyTo[0].Email != "support@example.com" {
		t.Fatalf("reply_to = %#v, want default reply-to", message.ReplyTo)
	}
	if message.Headers["X-App"] != "goforj" {
		t.Fatalf("header X-App = %q, want %q", message.Headers["X-App"], "goforj")
	}
	if len(message.Tags) != 1 || message.Tags[0] != "transactional" {
		t.Fatalf("tags = %#v, want default tag", message.Tags)
	}
	if message.Metadata["env"] != "test" || message.Metadata["request_id"] != "abc123" {
		t.Fatalf("metadata = %#v, want merged metadata", message.Metadata)
	}
}

func TestBuilderSendRequiresMailer(t *testing.T) {
	builder := (&mail.MessageBuilder{}).
		To("alice@example.com", "Alice").
		Subject("Welcome").
		Text("hello world")

	if err := builder.Send(context.Background()); !errors.Is(err, mail.ErrMissingMailer) {
		t.Fatalf("send without mailer error = %v, want %v", err, mail.ErrMissingMailer)
	}
}

func TestMessageValidation(t *testing.T) {
	message := mail.Message{
		To:      []mail.Recipient{{Email: "alice@example.com"}},
		Subject: "Welcome",
		Text:    "hello world",
	}
	if err := message.Validate(); err != nil {
		t.Fatalf("validate good message: %v", err)
	}

	message.To = nil
	if err := message.Validate(); !errors.Is(err, mail.ErrMissingRecipient) {
		t.Fatalf("validate missing recipients error = %v, want %v", err, mail.ErrMissingRecipient)
	}
}
