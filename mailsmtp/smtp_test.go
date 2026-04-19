package mailsmtp_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/goforj/mail"
	"github.com/goforj/mail/mailsmtp"
)

func TestRenderMultipartAlternative(t *testing.T) {
	raw, err := mailsmtp.Render(mail.Message{
		From:    &mail.Recipient{Email: "no-reply@example.com", Name: "Example"},
		To:      []mail.Recipient{{Email: "alice@example.com", Name: "Alice"}},
		Cc:      []mail.Recipient{{Email: "bob@example.com", Name: "Bob"}},
		ReplyTo: []mail.Recipient{{Email: "support@example.com", Name: "Support"}},
		Subject: "Welcome",
		Text:    "hello text",
		HTML:    "<p>hello html</p>",
		Headers: map[string]string{"X-App": "goforj"},
	})
	if err != nil {
		t.Fatalf("render smtp message: %v", err)
	}

	rendered := string(raw)
	for _, expected := range []string{
		`From: "Example" <no-reply@example.com>`,
		`To: "Alice" <alice@example.com>`,
		`Cc: "Bob" <bob@example.com>`,
		`Reply-To: "Support" <support@example.com>`,
		`Subject: Welcome`,
		`X-App: goforj`,
		`multipart/alternative; boundary=`,
		`Content-Type: text/plain; charset="utf-8"`,
		`Content-Type: text/html; charset="utf-8"`,
		`hello text`,
		`<p>hello html</p>`,
	} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("expected %q in rendered smtp message\n%s", expected, rendered)
		}
	}
}

func TestRenderSinglePartTextMessage(t *testing.T) {
	raw, err := mailsmtp.Render(mail.Message{
		From:    &mail.Recipient{Email: "no-reply@example.com"},
		To:      []mail.Recipient{{Email: "alice@example.com"}},
		Subject: "Welcome",
		Text:    "hello text",
	})
	if err != nil {
		t.Fatalf("render text smtp message: %v", err)
	}
	if !bytes.Contains(raw, []byte(`Content-Type: text/plain; charset="utf-8"`)) {
		t.Fatalf("expected text content-type in\n%s", string(raw))
	}
}

func TestRenderMultipartMixedWithAttachment(t *testing.T) {
	raw, err := mailsmtp.Render(mail.Message{
		From:    &mail.Recipient{Email: "no-reply@example.com"},
		To:      []mail.Recipient{{Email: "alice@example.com"}},
		Subject: "Welcome",
		Text:    "hello text",
		Attachments: []mail.Attachment{
			mail.AttachmentFromBytes("report.txt", "text/plain", []byte("hello attachment")),
		},
	})
	if err != nil {
		t.Fatalf("render attachment smtp message: %v", err)
	}

	rendered := string(raw)
	for _, expected := range []string{
		`multipart/mixed; boundary=`,
		`Content-Disposition: attachment; filename="report.txt"`,
		`Content-Transfer-Encoding: base64`,
		`aGVsbG8gYXR0YWNobWVudA==`,
	} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("expected %q in rendered smtp message\n%s", expected, rendered)
		}
	}
}
