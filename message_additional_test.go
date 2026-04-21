package mail_test

import (
	"errors"
	"testing"

	"github.com/goforj/mail"
)

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
				To:   []mail.Recipient{{Email: "alice@example.com"}},
				Text: "hello",
			},
			want: mail.ErrMissingSubject,
		},
		{
			name: "missing body",
			msg: mail.Message{
				To:      []mail.Recipient{{Email: "alice@example.com"}},
				Subject: "Welcome",
			},
			want: mail.ErrMissingBody,
		},
		{
			name: "invalid header name",
			msg: mail.Message{
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

func TestMessageValidateRejectsBlankRecipientAddress(t *testing.T) {
	err := (mail.Message{
		To:      []mail.Recipient{{Email: "   "}},
		Subject: "Welcome",
		Text:    "hello",
	}).Validate()
	if !errors.Is(err, mail.ErrInvalidRecipient) {
		t.Fatalf("Validate() error = %v, want %v", err, mail.ErrInvalidRecipient)
	}
}
