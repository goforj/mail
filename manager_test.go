package mail_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/goforj/mail"
	"github.com/goforj/mail/mailfake"
)

// TestMailerFluentSendAppliesDefaults ensures fluent delivery merges defaults before validation and send.
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

// TestBuilderSendRequiresMailer ensures standalone builders cannot silently discard a message.
func TestBuilderSendRequiresMailer(t *testing.T) {
	builder := (&mail.MessageBuilder{}).
		To("alice@example.com", "Alice").
		Subject("Welcome").
		Text("hello world")

	if err := builder.Send(context.Background()); !errors.Is(err, mail.ErrMissingMailer) {
		t.Fatalf("send without mailer error = %v, want %v", err, mail.ErrMissingMailer)
	}
}

// TestMailerSendReturnsValidationErrorAfterDefaults ensures incomplete defaults still fail before the driver.
func TestMailerSendReturnsValidationErrorAfterDefaults(t *testing.T) {
	fake := mailfake.New()
	mailer := mail.New(
		fake,
		mail.WithDefaultFrom("no-reply@example.com", "Example"),
		mail.WithDefaultMetadata("env", "test"),
	)

	err := mailer.Send(context.Background(), mail.Message{
		Subject: "Welcome",
		Text:    "hello world",
	})
	if !errors.Is(err, mail.ErrMissingRecipient) {
		t.Fatalf("Send() error = %v, want %v", err, mail.ErrMissingRecipient)
	}
}

// TestMailerApplyDefaultsCreatesMetadataMap ensures default metadata can merge into an initially nil message map.
func TestMailerApplyDefaultsCreatesMetadataMap(t *testing.T) {
	msg, err := mail.New(
		mailfake.New(),
		mail.WithDefaultFrom("no-reply@example.com", "Example"),
		mail.WithDefaultMetadata("env", "test"),
	).Message().
		To("alice@example.com", "Alice").
		Subject("Welcome").
		Text("hello world").
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if got, want := msg.Metadata["env"], "test"; got != want {
		t.Fatalf("metadata = %q, want %q", got, want)
	}
}

// TestMailerDefaultHeadersUseCaseInsensitiveIdentity ensures header defaults cannot duplicate an explicitly cased header.
func TestMailerDefaultHeadersUseCaseInsensitiveIdentity(t *testing.T) {
	message, err := mail.New(
		mailfake.New(),
		mail.WithDefaultFrom("no-reply@example.com", "Example"),
		mail.WithDefaultHeader("X-App", "default"),
	).Message().
		To("alice@example.com", "Alice").
		Subject("Welcome").
		Text("hello world").
		Header("x-app", "message").
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if got, want := len(message.Headers), 1; got != want {
		t.Fatalf("len(headers) = %d, want %d", got, want)
	}
	if got, want := message.Headers["x-app"], "message"; got != want {
		t.Fatalf("header = %q, want %q", got, want)
	}
}

// TestMessageValidation ensures the core required-field contract remains enforced through Mailer.
func TestMessageValidation(t *testing.T) {
	message := mail.Message{
		From:    &mail.Recipient{Email: "no-reply@example.com"},
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

// TestBuilderAttachments ensures in-memory and file attachments retain content and metadata.
func TestBuilderAttachments(t *testing.T) {
	attachmentPath := filepath.Join(t.TempDir(), "test-attachment.txt")
	if err := os.WriteFile(attachmentPath, []byte("hello attachment"), 0o644); err != nil {
		t.Fatalf("write temp attachment: %v", err)
	}

	message, err := mail.New(mailfake.New()).
		Message().
		From("no-reply@example.com", "Example").
		To("alice@example.com", "Alice").
		Subject("Welcome").
		Text("hello world").
		Attach("inline.txt", "text/plain", []byte("hello inline")).
		AttachFile(attachmentPath).
		Build()
	if err != nil {
		t.Fatalf("build message with attachments: %v", err)
	}

	if len(message.Attachments) != 2 {
		t.Fatalf("attachments = %#v, want 2", message.Attachments)
	}
	if message.Attachments[0].Filename != "inline.txt" {
		t.Fatalf("first attachment = %#v", message.Attachments[0])
	}
	if message.Attachments[1].Filename != "test-attachment.txt" {
		t.Fatalf("second attachment = %#v", message.Attachments[1])
	}
}

// TestAttachmentFromPathLoadsFile ensures filesystem attachments capture bytes and inferred media type at build time.
func TestAttachmentFromPathLoadsFile(t *testing.T) {
	attachmentPath := filepath.Join(t.TempDir(), "path-attachment.txt")
	if err := os.WriteFile(attachmentPath, []byte("hello path"), 0o644); err != nil {
		t.Fatalf("write temp attachment: %v", err)
	}

	attachment, err := mail.AttachmentFromPath(attachmentPath)
	if err != nil {
		t.Fatalf("attachment from path: %v", err)
	}
	if attachment.Filename != "path-attachment.txt" {
		t.Fatalf("filename = %q", attachment.Filename)
	}
	if len(attachment.Data) == 0 {
		t.Fatalf("expected attachment data")
	}
}
