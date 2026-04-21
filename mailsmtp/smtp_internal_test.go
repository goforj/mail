package mailsmtp

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/goforj/mail"
)

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
}

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
		Subject: "Welcome",
		Text:    "hello world",
	})
	if !errors.Is(err, mail.ErrMissingRecipient) {
		t.Fatalf("Send() error = %v, want %v", err, mail.ErrMissingRecipient)
	}
}

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

	body, contentType, err := renderInlineBody("hello text", "")
	if err != nil {
		t.Fatalf("renderInlineBody(text): %v", err)
	}
	if got, want := string(body), "hello text"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
	if got, want := contentType, `text/plain; charset="utf-8"`; got != want {
		t.Fatalf("content type = %q, want %q", got, want)
	}

	body, contentType, err = renderInlineBody("", "<p>hello</p>")
	if err != nil {
		t.Fatalf("renderInlineBody(html): %v", err)
	}
	if got, want := string(body), "<p>hello</p>"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
	if got, want := contentType, `text/html; charset="utf-8"`; got != want {
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
	if got, want := escapeHeaderToken(`report "Q1"\draft.txt`), `report \"Q1\"\\draft.txt`; got != want {
		t.Fatalf("escapeHeaderToken() = %q, want %q", got, want)
	}
}

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

func TestRenderBodyVariants(t *testing.T) {
	body, contentType, err := renderBody(mail.Message{HTML: "<p>hello</p>"})
	if err != nil {
		t.Fatalf("renderBody(html): %v", err)
	}
	if got, want := string(body), "<p>hello</p>"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
	if got, want := contentType, `text/html; charset="utf-8"`; got != want {
		t.Fatalf("content type = %q, want %q", got, want)
	}

	body, contentType, err = renderBody(mail.Message{
		Text: "hello text",
		HTML: "<p>hello</p>",
		Attachments: []mail.Attachment{
			mail.AttachmentFromBytes("report.txt", "text/plain", []byte("attachment body")),
		},
	})
	if err != nil {
		t.Fatalf("renderBody(attachment): %v", err)
	}
	rendered := string(body)
	for _, expected := range []string{
		`multipart/alternative; boundary="`,
		`Content-Disposition: attachment; filename="report.txt"`,
		`YXR0YWNobWVudCBib2R5`,
	} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("expected %q in rendered body\n%s", expected, rendered)
		}
	}
	if !strings.HasPrefix(contentType, `multipart/mixed; boundary="`) {
		t.Fatalf("content type = %q", contentType)
	}
}

func TestRenderRejectsInvalidMessage(t *testing.T) {
	_, err := Render(mail.Message{
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

func (w errWriter) Write([]byte) (int, error) {
	return 0, w.err
}

var _ io.Writer = errWriter{}
