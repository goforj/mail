package mailsmtp

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/goforj/mail"
)

// TestNewDefaultsAndValidation ensures SMTP construction applies secure defaults and rejects incomplete transport settings.
func TestNewDefaultsAndValidation(t *testing.T) {
	driver, err := New(Config{
		Host: " smtp.example.com ",
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if got, want := driver.host, "smtp.example.com"; got != want {
		t.Fatalf("host = %q, want %q", got, want)
	}
	if got, want := driver.port, 25; got != want {
		t.Fatalf("port = %d, want %d", got, want)
	}
	if got, want := driver.tls.ServerName, "smtp.example.com"; got != want {
		t.Fatalf("TLS ServerName = %q, want %q", got, want)
	}
	if got, want := driver.tls.MinVersion, uint16(tls.VersionTLS12); got != want {
		t.Fatalf("TLS MinVersion = %d, want %d", got, want)
	}
	if driver.tls.InsecureSkipVerify {
		t.Fatal("TLS verification should be enabled by default")
	}

	driver, err = New(Config{
		Host:     "smtp.example.com",
		Port:     587,
		Username: " user ",
		Password: "pass",
		Identity: " ident ",
		ForceTLS: true,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if got, want := driver.username, "user"; got != want {
		t.Fatalf("username = %q, want %q", got, want)
	}
	if got, want := driver.identity, "ident"; got != want {
		t.Fatalf("identity = %q, want %q", got, want)
	}
	if !driver.forceTLS {
		t.Fatal("forceTLS should be true")
	}

	if _, err := New(Config{}); err == nil {
		t.Fatal("New() should reject an empty host")
	}
	for _, port := range []int{-1, 65536} {
		if _, err := New(Config{Host: "smtp.example.com", Port: port}); err == nil {
			t.Fatalf("New() should reject port %d", port)
		}
	}

	customTLS := &tls.Config{ServerName: "mail.internal", MinVersion: tls.VersionTLS13}
	driver, err = New(Config{Host: "smtp.example.com", TLSConfig: customTLS})
	if err != nil {
		t.Fatalf("New(custom TLS) error = %v", err)
	}
	if driver.tls == customTLS {
		t.Fatal("New() should clone caller TLS config")
	}
	customTLS.ServerName = "mutated.example.com"
	if got, want := driver.tls.ServerName, "mail.internal"; got != want {
		t.Fatalf("cloned TLS ServerName = %q, want %q", got, want)
	}
}

// TestDriverSendEarlyReturns ensures invalid messages and canceled contexts fail before dialing SMTP.
func TestDriverSendEarlyReturns(t *testing.T) {
	driver, err := New(Config{Host: "smtp.example.com"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := driver.Send(ctx, mail.Message{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("Send() error = %v, want %v", err, context.Canceled)
	}

	err = driver.Send(context.Background(), mail.Message{
		From:    &mail.Recipient{Email: "no-reply@example.com"},
		Subject: "Welcome",
		Text:    "hello world",
	})
	if !errors.Is(err, mail.ErrMissingRecipient) {
		t.Fatalf("Send() error = %v, want %v", err, mail.ErrMissingRecipient)
	}
}

// TestAuthRenderHelpersAndRecipients ensures authentication, envelope recipients, and MIME helpers preserve their separate responsibilities.
func TestAuthRenderHelpersAndRecipients(t *testing.T) {
	driver, err := New(Config{Host: "smtp.example.com"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if got := driver.auth(); got != nil {
		t.Fatalf("auth() = %#v, want nil", got)
	}

	driver, err = New(Config{Host: "smtp.example.com", Username: "user", Password: "pass"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if got := driver.auth(); got == nil {
		t.Fatal("auth() should return smtp auth when credentials exist")
	}

	part, err := renderInlineBody("hello text", "")
	if err != nil {
		t.Fatalf("renderInlineBody(text): %v", err)
	}
	if got, want := string(part.data), "hello text"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
	if got, want := part.contentType, `text/plain; charset="utf-8"`; got != want {
		t.Fatalf("content type = %q, want %q", got, want)
	}
	if got, want := part.transferEncoding, "quoted-printable"; got != want {
		t.Fatalf("transfer encoding = %q, want %q", got, want)
	}

	part, err = renderInlineBody("", "<p>hello</p>")
	if err != nil {
		t.Fatalf("renderInlineBody(html): %v", err)
	}
	if got, want := string(part.data), "<p>hello</p>"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
	if got, want := part.contentType, `text/html; charset="utf-8"`; got != want {
		t.Fatalf("content type = %q, want %q", got, want)
	}

	recipients := collectRecipients(mail.Message{
		To:  []mail.Recipient{{Email: "alice@example.com"}},
		Cc:  []mail.Recipient{{Email: "manager@example.com"}},
		Bcc: []mail.Recipient{{Email: "audit@example.com"}},
	})
	if got, want := strings.Join(recipients, ","), "alice@example.com,manager@example.com,audit@example.com"; got != want {
		t.Fatalf("recipients = %q, want %q", got, want)
	}

	formatted := formatRecipients([]mail.Recipient{
		{Email: "alice@example.com", Name: "Alice"},
		{Email: "bob@example.com"},
	})
	if got, want := formatted, `"Alice" <alice@example.com>, bob@example.com`; got != want {
		t.Fatalf("formatRecipients() = %q, want %q", got, want)
	}
}

// TestBase64LineWriterWrapsAndCloses ensures attachment encoding obeys MIME line limits and flushes its tail.
func TestBase64LineWriterWrapsAndCloses(t *testing.T) {
	var out bytes.Buffer
	writer := newBase64LineWriter(&out)

	payload := strings.Repeat("a", 80)
	if _, err := writer.Write([]byte(payload)); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	rendered := out.String()
	lines := strings.Split(strings.TrimSuffix(rendered, "\r\n"), "\r\n")
	if len(lines) != 2 {
		t.Fatalf("wrapped lines = %#v", lines)
	}
	if got := len(lines[0]); got != 76 {
		t.Fatalf("first line len = %d, want 76", got)
	}
	if got := len(lines[1]); got != 4 {
		t.Fatalf("second line len = %d, want 4", got)
	}

	var empty bytes.Buffer
	emptyWriter := newBase64LineWriter(&empty)
	if err := emptyWriter.Close(); err != nil {
		t.Fatalf("Close(empty) error = %v", err)
	}
	if empty.Len() != 0 {
		t.Fatalf("empty writer output = %q", empty.String())
	}
}

// TestBase64LineWriterPropagatesTargetErrors ensures MIME encoding cannot hide downstream write failures.
func TestBase64LineWriterPropagatesTargetErrors(t *testing.T) {
	writeErr := errors.New("write failed")
	writer := newBase64LineWriter(errWriter{err: writeErr})
	if _, err := writer.Write([]byte(strings.Repeat("a", 76))); !errors.Is(err, writeErr) {
		t.Fatalf("Write() error = %v, want %v", err, writeErr)
	}

	closeErr := errors.New("close failed")
	writer = newBase64LineWriter(errWriter{err: closeErr})
	if _, err := writer.Write([]byte("abcd")); err != nil {
		t.Fatalf("Write() unexpected error = %v", err)
	}
	if err := writer.Close(); !errors.Is(err, closeErr) {
		t.Fatalf("Close() error = %v, want %v", err, closeErr)
	}
}

// TestRenderBodyVariants ensures text, HTML, and attachment combinations select valid MIME structures.
func TestRenderBodyVariants(t *testing.T) {
	part, err := renderBody(mail.Message{HTML: "<p>hello</p>"})
	if err != nil {
		t.Fatalf("renderBody(html): %v", err)
	}
	if got, want := string(part.data), "<p>hello</p>"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
	if got, want := part.contentType, `text/html; charset="utf-8"`; got != want {
		t.Fatalf("content type = %q, want %q", got, want)
	}

	part, err = renderBody(mail.Message{
		Text: "hello text",
		HTML: "<p>hello</p>",
		Attachments: []mail.Attachment{
			mail.AttachmentFromBytes("report.txt", "text/plain", []byte("attachment body")),
		},
	})
	if err != nil {
		t.Fatalf("renderBody(attachment): %v", err)
	}
	rendered := string(part.data)
	for _, expected := range []string{
		`multipart/alternative; boundary=`,
		`Content-Disposition: attachment; filename=report.txt`,
		`YXR0YWNobWVudCBib2R5`,
	} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("expected %q in rendered body\n%s", expected, rendered)
		}
	}
	if !strings.HasPrefix(part.contentType, `multipart/mixed; boundary=`) {
		t.Fatalf("content type = %q", part.contentType)
	}
}

// TestRenderRejectsInvalidMessage ensures malformed messages fail before any SMTP bytes are produced.
func TestRenderRejectsInvalidMessage(t *testing.T) {
	_, err := Render(mail.Message{
		From:    &mail.Recipient{Email: "no-reply@example.com"},
		Subject: "Welcome",
		Text:    "hello world",
	})
	if !errors.Is(err, mail.ErrMissingRecipient) {
		t.Fatalf("Render() error = %v, want %v", err, mail.ErrMissingRecipient)
	}
}

type errWriter struct {
	err error
}

// Write injects the configured output failure into MIME rendering tests.
func (w errWriter) Write([]byte) (int, error) {
	return 0, w.err
}

var _ io.Writer = errWriter{}
