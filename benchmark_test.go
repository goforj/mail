package mail_test

import (
	"testing"

	"github.com/goforj/mail"
	"github.com/goforj/mail/mailfake"
	"github.com/goforj/mail/mailsmtp"
)

var (
	benchmarkMessage mail.Message
	benchmarkRaw     []byte
)

// BenchmarkMessageClone measures ownership-preserving copies of a representative message.
func BenchmarkMessageClone(b *testing.B) {
	message := benchmarkFixture()
	b.ReportAllocs()
	for b.Loop() {
		benchmarkMessage = message.Clone()
	}
}

// BenchmarkMessageBuild measures fluent construction and validation of a common message.
func BenchmarkMessageBuild(b *testing.B) {
	mailer := mail.New(mailfake.New(), mail.WithDefaultFrom("no-reply@example.com", "Example"))
	b.ReportAllocs()
	for b.Loop() {
		message, err := mailer.Message().
			To("alice@example.com", "Alice").
			Subject("Welcome").
			Text("hello world").
			Header("X-App", "goforj").
			Build()
		if err != nil {
			b.Fatal(err)
		}
		benchmarkMessage = message
	}
}

// BenchmarkSMTPRender measures deterministic MIME rendering with both bodies and an attachment.
func BenchmarkSMTPRender(b *testing.B) {
	message := benchmarkFixture()
	b.ReportAllocs()
	for b.Loop() {
		raw, err := mailsmtp.Render(message)
		if err != nil {
			b.Fatal(err)
		}
		benchmarkRaw = raw
	}
}

// benchmarkFixture exercises maps, dual bodies, and attachment copying without external I/O.
func benchmarkFixture() mail.Message {
	return mail.Message{
		From:    &mail.Recipient{Email: "no-reply@example.com", Name: "Example"},
		To:      []mail.Recipient{{Email: "alice@example.com", Name: "Alice"}},
		Subject: "Welcome",
		Text:    "hello world",
		HTML:    "<p>hello world</p>",
		Headers: map[string]string{"X-App": "goforj"},
		Metadata: map[string]string{
			"tenant_id": "tenant_123",
		},
		Attachments: []mail.Attachment{
			mail.AttachmentFromBytes("report.bin", "application/octet-stream", make([]byte, 4096)),
		},
	}
}
