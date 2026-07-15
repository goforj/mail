package mailsmtp_test

import (
	"bytes"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	stdmail "net/mail"
	"os"
	"strings"
	"testing"

	"github.com/goforj/mail"
	"github.com/goforj/mail/mailsmtp"
)

// TestRenderMatchesSimpleGoldenMessage ensures the stable MIME wire format remains compatible with existing consumers.
func TestRenderMatchesSimpleGoldenMessage(t *testing.T) {
	raw, err := mailsmtp.Render(mail.Message{
		From:    &mail.Recipient{Email: "no-reply@example.com", Name: "Example"},
		To:      []mail.Recipient{{Email: "alice@example.com", Name: "Alice"}},
		Subject: "Welcome",
		Text:    "hello world",
		Headers: map[string]string{"X-App": "goforj"},
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	want, err := os.ReadFile("testdata/simple.golden")
	if err != nil {
		t.Fatalf("read golden file: %v", err)
	}
	want = bytes.ReplaceAll(want, []byte("\n"), []byte("\r\n"))
	want = bytes.TrimSuffix(want, []byte("\r\n"))
	if !bytes.Equal(raw, want) {
		t.Fatalf("rendered message mismatch\n--- got ---\n%s\n--- want ---\n%s", raw, want)
	}
}

// TestRenderEncodesUnicodeAndKeepsBccOutOfHeaders ensures internationalized fields are safe without leaking blind recipients.
func TestRenderEncodesUnicodeAndKeepsBccOutOfHeaders(t *testing.T) {
	raw, err := mailsmtp.Render(mail.Message{
		From:    &mail.Recipient{Email: "no-reply@example.com", Name: "Équipe"},
		To:      []mail.Recipient{{Email: "alice@example.com", Name: "Alice"}},
		Bcc:     []mail.Recipient{{Email: "audit@example.com", Name: "Audit"}},
		Subject: "Olá, 世界",
		Text:    "  hello\nworld  ",
		Attachments: []mail.Attachment{
			mail.AttachmentFromBytes("résumé.pdf", "application/pdf", []byte("attachment")),
		},
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	message, err := stdmail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("ReadMessage() error = %v", err)
	}
	if !strings.HasPrefix(message.Header.Get("Subject"), "=?utf-8?") {
		t.Fatalf("Subject = %q, want encoded word", message.Header.Get("Subject"))
	}
	if got := message.Header.Get("Bcc"); got != "" {
		t.Fatalf("Bcc header = %q, want empty", got)
	}

	mediaType, params, err := mime.ParseMediaType(message.Header.Get("Content-Type"))
	if err != nil {
		t.Fatalf("ParseMediaType() error = %v", err)
	}
	if mediaType != "multipart/mixed" {
		t.Fatalf("media type = %q, want multipart/mixed", mediaType)
	}
	reader := multipart.NewReader(message.Body, params["boundary"])
	bodyPart, err := reader.NextRawPart()
	if err != nil {
		t.Fatalf("NextPart(body) error = %v", err)
	}
	decodedBody, err := io.ReadAll(quotedprintable.NewReader(bodyPart))
	if err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if got, want := string(decodedBody), "  hello\r\nworld  "; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
	attachmentPart, err := reader.NextRawPart()
	if err != nil {
		t.Fatalf("NextPart(attachment) error = %v", err)
	}
	if got, want := attachmentPart.FileName(), "résumé.pdf"; got != want {
		t.Fatalf("attachment filename = %q, want %q", got, want)
	}
}

// TestRenderRejectsCaseInsensitiveDuplicateHeaders ensures MIME singleton headers cannot be smuggled through casing differences.
func TestRenderRejectsCaseInsensitiveDuplicateHeaders(t *testing.T) {
	message := mail.Message{
		From:    &mail.Recipient{Email: "no-reply@example.com"},
		To:      []mail.Recipient{{Email: "alice@example.com"}},
		Subject: "Welcome",
		Text:    "hello world",
		Headers: map[string]string{"X-App": "one", "x-app": "two"},
	}
	for range 20 {
		_, err := mailsmtp.Render(message)
		if !errors.Is(err, mail.ErrDuplicateHeader) {
			t.Fatalf("Render() error = %v, want %v", err, mail.ErrDuplicateHeader)
		}
	}
}
