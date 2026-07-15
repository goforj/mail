package mail_test

import (
	"errors"
	"testing"

	"github.com/goforj/mail"
)

// TestMessageValidateBranches ensures every required address, header, body, and attachment invariant is enforced.
func TestMessageValidateBranches(t *testing.T) {
	tests := []struct {
		name string
		msg  mail.Message
		want error
	}{
		{
			name: "invalid from",
			msg: mail.Message{
				From:    &mail.Recipient{Email: "bad"},
				To:      []mail.Recipient{{Email: "alice@example.com"}},
				Subject: "Welcome",
				Text:    "hello",
			},
			want: mail.ErrInvalidFrom,
		},
		{
			name: "invalid reply to",
			msg: mail.Message{
				From:    &mail.Recipient{Email: "no-reply@example.com"},
				ReplyTo: []mail.Recipient{{Email: "bad"}},
				To:      []mail.Recipient{{Email: "alice@example.com"}},
				Subject: "Welcome",
				Text:    "hello",
			},
			want: mail.ErrInvalidReplyTo,
		},
		{
			name: "invalid recipient in cc",
			msg: mail.Message{
				From:    &mail.Recipient{Email: "no-reply@example.com"},
				To:      []mail.Recipient{{Email: "alice@example.com"}},
				Cc:      []mail.Recipient{{Email: "bad"}},
				Subject: "Welcome",
				Text:    "hello",
			},
			want: mail.ErrInvalidRecipient,
		},
		{
			name: "invalid recipient in bcc",
			msg: mail.Message{
				From:    &mail.Recipient{Email: "no-reply@example.com"},
				To:      []mail.Recipient{{Email: "alice@example.com"}},
				Bcc:     []mail.Recipient{{Email: "bad"}},
				Subject: "Welcome",
				Text:    "hello",
			},
			want: mail.ErrInvalidRecipient,
		},
		{
			name: "missing subject",
			msg: mail.Message{
				From: &mail.Recipient{Email: "no-reply@example.com"},
				To:   []mail.Recipient{{Email: "alice@example.com"}},
				Text: "hello",
			},
			want: mail.ErrMissingSubject,
		},
		{
			name: "missing body",
			msg: mail.Message{
				From:    &mail.Recipient{Email: "no-reply@example.com"},
				To:      []mail.Recipient{{Email: "alice@example.com"}},
				Subject: "Welcome",
			},
			want: mail.ErrMissingBody,
		},
		{
			name: "invalid header name",
			msg: mail.Message{
				From:    &mail.Recipient{Email: "no-reply@example.com"},
				To:      []mail.Recipient{{Email: "alice@example.com"}},
				Subject: "Welcome",
				Text:    "hello",
				Headers: map[string]string{"bad:name": "value"},
			},
			want: mail.ErrInvalidHeaderName,
		},
		{
			name: "invalid attachment filename",
			msg: mail.Message{
				From:    &mail.Recipient{Email: "no-reply@example.com"},
				To:      []mail.Recipient{{Email: "alice@example.com"}},
				Subject: "Welcome",
				Text:    "hello",
				Attachments: []mail.Attachment{
					{ContentType: "text/plain", Data: []byte("hello")},
				},
			},
			want: mail.ErrInvalidAttachment,
		},
		{
			name: "invalid attachment content type",
			msg: mail.Message{
				From:    &mail.Recipient{Email: "no-reply@example.com"},
				To:      []mail.Recipient{{Email: "alice@example.com"}},
				Subject: "Welcome",
				Text:    "hello",
				Attachments: []mail.Attachment{
					{Filename: "report.txt", Data: []byte("hello")},
				},
			},
			want: mail.ErrInvalidAttachment,
		},
		{
			name: "invalid attachment nil data",
			msg: mail.Message{
				From:    &mail.Recipient{Email: "no-reply@example.com"},
				To:      []mail.Recipient{{Email: "alice@example.com"}},
				Subject: "Welcome",
				Text:    "hello",
				Attachments: []mail.Attachment{
					{Filename: "report.txt", ContentType: "text/plain"},
				},
			},
			want: mail.ErrInvalidAttachment,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.Validate()
			if !errors.Is(err, tt.want) {
				t.Fatalf("Validate() error = %v, want %v", err, tt.want)
			}
		})
	}
}

// TestMessageValidateRejectsBlankRecipientAddress ensures display names cannot stand in for deliverable addresses.
func TestMessageValidateRejectsBlankRecipientAddress(t *testing.T) {
	err := (mail.Message{
		From:    &mail.Recipient{Email: "no-reply@example.com"},
		To:      []mail.Recipient{{Email: "   "}},
		Subject: "Welcome",
		Text:    "hello",
	}).Validate()
	if !errors.Is(err, mail.ErrInvalidRecipient) {
		t.Fatalf("Validate() error = %v, want %v", err, mail.ErrInvalidRecipient)
	}
}

// TestMessageValidateRejectsUnsafePortableFields ensures cross-provider fields cannot inject headers or invalid MIME metadata.
func TestMessageValidateRejectsUnsafePortableFields(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*mail.Message)
		want   error
	}{
		{
			name:   "missing from",
			mutate: func(message *mail.Message) { message.From = nil },
			want:   mail.ErrMissingFrom,
		},
		{
			name:   "display address in email field",
			mutate: func(message *mail.Message) { message.From.Email = "Example <no-reply@example.com>" },
			want:   mail.ErrInvalidFrom,
		},
		{
			name:   "display name injection",
			mutate: func(message *mail.Message) { message.From.Name = "Example\r\nBcc: audit@example.com" },
			want:   mail.ErrInvalidFrom,
		},
		{
			name:   "subject injection",
			mutate: func(message *mail.Message) { message.Subject = "Welcome\r\nBcc: audit@example.com" },
			want:   mail.ErrInvalidSubject,
		},
		{
			name:   "header name whitespace",
			mutate: func(message *mail.Message) { message.Headers = map[string]string{"Bad Header": "value"} },
			want:   mail.ErrInvalidHeaderName,
		},
		{
			name: "header value injection",
			mutate: func(message *mail.Message) {
				message.Headers = map[string]string{"X-App": "safe\r\nBcc: audit@example.com"}
			},
			want: mail.ErrInvalidHeaderValue,
		},
		{
			name:   "reserved header",
			mutate: func(message *mail.Message) { message.Headers = map[string]string{"subject": "replacement"} },
			want:   mail.ErrReservedHeader,
		},
		{
			name: "case-insensitive duplicate header",
			mutate: func(message *mail.Message) {
				message.Headers = map[string]string{"X-App": "one", "x-app": "two"}
			},
			want: mail.ErrDuplicateHeader,
		},
		{
			name: "attachment filename injection",
			mutate: func(message *mail.Message) {
				message.Attachments = []mail.Attachment{{Filename: "report.txt\r\nX-Bad: value", ContentType: "text/plain", Data: []byte("data")}}
			},
			want: mail.ErrInvalidAttachment,
		},
		{
			name: "malformed attachment media type",
			mutate: func(message *mail.Message) {
				message.Attachments = []mail.Attachment{{Filename: "report.txt", ContentType: "not a media type", Data: []byte("data")}}
			},
			want: mail.ErrInvalidAttachment,
		},
		{
			name:   "metadata key injection",
			mutate: func(message *mail.Message) { message.Metadata = map[string]string{"tenant\r\nX-Bad": "value"} },
			want:   mail.ErrInvalidMetadata,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			message := validMessage()
			test.mutate(&message)
			if err := message.Validate(); !errors.Is(err, test.want) {
				t.Fatalf("Validate() error = %v, want %v", err, test.want)
			}
		})
	}
}

// TestMessageValidateAcceptsUnicodeContentAndParameterizedMediaType ensures safe internationalized content remains portable.
func TestMessageValidateAcceptsUnicodeContentAndParameterizedMediaType(t *testing.T) {
	message := validMessage()
	message.From.Name = "Équipe"
	message.Subject = "Olá, 世界"
	message.Headers = map[string]string{"X-Trace": "café"}
	message.Metadata = map[string]string{"tenant.id": "tenant_123"}
	message.Attachments = []mail.Attachment{{
		Filename:    "résumé.pdf",
		ContentType: `application/pdf; name="résumé.pdf"`,
		Data:        []byte{},
	}}
	if err := message.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

// validMessage provides one complete portable message for focused validation mutations.
func validMessage() mail.Message {
	return mail.Message{
		From:    &mail.Recipient{Email: "no-reply@example.com", Name: "Example"},
		To:      []mail.Recipient{{Email: "alice@example.com", Name: "Alice"}},
		Subject: "Welcome",
		Text:    "hello world",
	}
}
