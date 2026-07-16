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

// TestBuilderFluentMethodsPopulateMessage ensures fluent setters populate every portable message field.
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

// TestBuilderMessageReturnsClone ensures callers cannot mutate builder state through returned messages.
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

// TestBuilderBuildWithoutMailerValidates ensures message construction remains useful without a delivery dependency.
func TestBuilderBuildWithoutMailerValidates(t *testing.T) {
	msg, err := (&mail.MessageBuilder{}).
		From("no-reply@example.com", "Example").
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
		From("no-reply@example.com", "Example").
		Subject("Welcome").
		Text("hello world").
		Build()
	if !errors.Is(err, mail.ErrMissingRecipient) {
		t.Fatalf("Build() error = %v, want %v", err, mail.ErrMissingRecipient)
	}
}

// TestBuilderBuildReturnsAttachmentError ensures file-loading failures survive until build.
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

// TestBuilderSkipsSubsequentAttachFileAfterError ensures the first attachment failure remains authoritative.
func TestBuilderSkipsSubsequentAttachFileAfterError(t *testing.T) {
	builder := mail.New(mailfake.New()).Message().
		To("alice@example.com", "Alice").
		Subject("Welcome").
		Text("hello world").
		AttachFile("missing-file.txt").
		AttachFile("still-missing.txt")

	if _, err := builder.Build(); err == nil {
		t.Fatal("Build() should preserve the first attachment error")
	}
}

// TestBuilderSendReturnsBuildErrorEarly ensures invalid builder state prevents driver invocation.
func TestBuilderSendReturnsBuildErrorEarly(t *testing.T) {
	err := mail.New(mailfake.New()).Message().
		To("alice@example.com", "Alice").
		Subject("Welcome").
		Text("hello world").
		AttachFile("missing-file.txt").
		Send(context.Background())
	if err == nil {
		t.Fatal("Send() should return the build error")
	}
}

// TestBuilderBuildWithMailerReturnsValidationError ensures mailer defaults do not bypass final validation.
func TestBuilderBuildWithMailerReturnsValidationError(t *testing.T) {
	_, err := mail.New(mailfake.New()).
		Message().
		From("no-reply@example.com", "Example").
		Subject("Welcome").
		Text("hello world").
		Build()
	if !errors.Is(err, mail.ErrMissingRecipient) {
		t.Fatalf("Build() error = %v, want %v", err, mail.ErrMissingRecipient)
	}
}

// TestNewPanicsWithoutDriver ensures a required delivery driver fails fast during wiring.
func TestNewPanicsWithoutDriver(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("New(nil) should panic")
		}
	}()
	_ = mail.New(nil)
}

// TestNewPanicsWithTypedNilDriver ensures typed-nil delivery drivers cannot bypass fail-fast wiring.
func TestNewPanicsWithTypedNilDriver(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("New(typed nil) should panic")
		}
	}()
	var driver *typedNilDriver
	_ = mail.New(driver)
}

// TestZeroValueMailerReportsMissingDriver ensures accidental zero-value use returns a clear wiring error.
func TestZeroValueMailerReportsMissingDriver(t *testing.T) {
	var mailer mail.Mailer
	err := mailer.Send(context.Background(), validMessage())
	if !errors.Is(err, mail.ErrMissingMailer) {
		t.Fatalf("Send() error = %v, want %v", err, mail.ErrMissingMailer)
	}
}

type typedNilDriver struct{}

// Send satisfies mail.Driver for typed-nil construction coverage.
func (*typedNilDriver) Send(context.Context, mail.Message) error {
	return nil
}

// TestAttachmentFromPathFallbackContentType ensures unknown file types use a portable binary media type.
func TestAttachmentFromPathFallbackContentType(t *testing.T) {
	attachmentPath := filepath.Join(t.TempDir(), "attachment.unknownext")
	if err := os.WriteFile(attachmentPath, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write temp attachment: %v", err)
	}

	attachment, err := mail.AttachmentFromPath(attachmentPath)
	if err != nil {
		t.Fatalf("AttachmentFromPath() error = %v", err)
	}
	if got, want := attachment.ContentType, "application/octet-stream"; got != want {
		t.Fatalf("content type = %q, want %q", got, want)
	}
}
