package mail

import (
	"errors"
	"mime"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
)

var (
	// ErrMissingMailer indicates that no driver is configured for the attempted send.
	ErrMissingMailer = errors.New("mail: missing mailer")
	// ErrMissingRecipient indicates that a message has no to, cc, or bcc recipients.
	ErrMissingRecipient = errors.New("mail: at least one recipient is required")
	// ErrMissingSubject indicates that a message has no subject.
	ErrMissingSubject = errors.New("mail: subject is required")
	// ErrMissingBody indicates that a message has neither HTML nor text body content.
	ErrMissingBody = errors.New("mail: html or text body is required")
	// ErrInvalidRecipient indicates that one or more recipients are malformed.
	ErrInvalidRecipient = errors.New("mail: invalid recipient")
	// ErrInvalidFrom indicates that the from recipient is malformed.
	ErrInvalidFrom = errors.New("mail: invalid from recipient")
	// ErrInvalidReplyTo indicates that a reply-to recipient is malformed.
	ErrInvalidReplyTo = errors.New("mail: invalid reply-to recipient")
	// ErrInvalidHeaderName indicates that a header name is empty or malformed.
	ErrInvalidHeaderName = errors.New("mail: invalid header name")
	// ErrInvalidAttachment indicates that an attachment is missing required fields.
	ErrInvalidAttachment = errors.New("mail: invalid attachment")
)

// Recipient identifies one email recipient with an optional display name.
// @group Message Model
type Recipient struct {
	Email string
	Name  string
}

// Attachment is one portable mail attachment.
// @group Message Model
type Attachment struct {
	Filename    string
	ContentType string
	Data        []byte
}

// Message is the canonical portable email envelope used by drivers.
// @group Message Model
type Message struct {
	From        *Recipient
	ReplyTo     []Recipient
	To          []Recipient
	Cc          []Recipient
	Bcc         []Recipient
	Subject     string
	HTML        string
	Text        string
	Headers     map[string]string
	Tags        []string
	Metadata    map[string]string
	Attachments []Attachment
}

// AttachmentFromBytes creates one attachment from in-memory content.
// @group Message Model
//
// Example: from bytes
//
//	attachment := mail.AttachmentFromBytes("report.txt", "text/plain", []byte("hello world"))
//	fmt.Println(attachment.Filename)
//	// report.txt
func AttachmentFromBytes(filename, contentType string, data []byte) Attachment {
	return Attachment{
		Filename:    strings.TrimSpace(filename),
		ContentType: strings.TrimSpace(contentType),
		Data:        append([]byte(nil), data...),
	}
}

// AttachmentFromPath loads one attachment from a local file path.
// @group Message Model
//
// Example: from a file
//
//	_ = os.WriteFile("report.txt", []byte("hello world"), 0o644)
//	defer os.Remove("report.txt")
//	attachment, _ := mail.AttachmentFromPath("report.txt")
//	fmt.Println(attachment.Filename)
//	// report.txt
func AttachmentFromPath(path string) (Attachment, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Attachment{}, err
	}
	filename := filepath.Base(path)
	contentType := mime.TypeByExtension(filepath.Ext(filename))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	return Attachment{
		Filename:    filename,
		ContentType: contentType,
		Data:        data,
	}, nil
}

// Clone returns a copy of the message safe for reuse in tests and builders.
// @group Message Model
//
// Example: clone
//
//	original := mail.Message{
//		To:      []mail.Recipient{{Email: "alice@example.com", Name: "Alice"}},
//		Subject: "Welcome",
//		Text:    "hello world",
//	}
//	cloned := original.Clone()
//	cloned.Subject = "Changed"
//	fmt.Println(original.Subject)
//	// Welcome
func (m Message) Clone() Message {
	cloned := m
	if m.From != nil {
		from := *m.From
		cloned.From = &from
	}
	cloned.ReplyTo = append([]Recipient(nil), m.ReplyTo...)
	cloned.To = append([]Recipient(nil), m.To...)
	cloned.Cc = append([]Recipient(nil), m.Cc...)
	cloned.Bcc = append([]Recipient(nil), m.Bcc...)
	cloned.Tags = append([]string(nil), m.Tags...)
	if len(m.Attachments) > 0 {
		cloned.Attachments = make([]Attachment, 0, len(m.Attachments))
		for _, attachment := range m.Attachments {
			cloned.Attachments = append(cloned.Attachments, Attachment{
				Filename:    attachment.Filename,
				ContentType: attachment.ContentType,
				Data:        append([]byte(nil), attachment.Data...),
			})
		}
	}
	if len(m.Headers) > 0 {
		cloned.Headers = make(map[string]string, len(m.Headers))
		for k, v := range m.Headers {
			cloned.Headers[k] = v
		}
	}
	if len(m.Metadata) > 0 {
		cloned.Metadata = make(map[string]string, len(m.Metadata))
		for k, v := range m.Metadata {
			cloned.Metadata[k] = v
		}
	}
	return cloned
}

// Validate checks that the message has valid recipients, subject, body, and headers.
// @group Message Model
//
// Example: validate
//
//	err := (mail.Message{
//		From:    &mail.Recipient{Email: "no-reply@example.com", Name: "Example"},
//		To:      []mail.Recipient{{Email: "alice@example.com", Name: "Alice"}},
//		Subject: "Welcome",
//		Text:    "hello world",
//	}).Validate()
//	fmt.Println(err == nil)
//	// true
func (m Message) Validate() error {
	if m.From != nil {
		if err := validateRecipient(*m.From); err != nil {
			return ErrInvalidFrom
		}
	}
	for _, recipient := range m.ReplyTo {
		if err := validateRecipient(recipient); err != nil {
			return ErrInvalidReplyTo
		}
	}
	recipientCount := len(m.To) + len(m.Cc) + len(m.Bcc)
	if recipientCount == 0 {
		return ErrMissingRecipient
	}
	for _, recipient := range m.To {
		if err := validateRecipient(recipient); err != nil {
			return ErrInvalidRecipient
		}
	}
	for _, recipient := range m.Cc {
		if err := validateRecipient(recipient); err != nil {
			return ErrInvalidRecipient
		}
	}
	for _, recipient := range m.Bcc {
		if err := validateRecipient(recipient); err != nil {
			return ErrInvalidRecipient
		}
	}
	if strings.TrimSpace(m.Subject) == "" {
		return ErrMissingSubject
	}
	if strings.TrimSpace(m.HTML) == "" && strings.TrimSpace(m.Text) == "" {
		return ErrMissingBody
	}
	for name := range m.Headers {
		if strings.TrimSpace(name) == "" || strings.ContainsAny(name, "\r\n:") {
			return ErrInvalidHeaderName
		}
	}
	for _, attachment := range m.Attachments {
		if strings.TrimSpace(attachment.Filename) == "" {
			return ErrInvalidAttachment
		}
		if strings.TrimSpace(attachment.ContentType) == "" {
			return ErrInvalidAttachment
		}
		if attachment.Data == nil {
			return ErrInvalidAttachment
		}
	}
	return nil
}

func validateRecipient(recipient Recipient) error {
	address := strings.TrimSpace(recipient.Email)
	if address == "" {
		return ErrInvalidRecipient
	}
	_, err := mail.ParseAddress(formatRecipient(Recipient{Email: address, Name: recipient.Name}))
	return err
}

func formatRecipient(recipient Recipient) string {
	address := strings.TrimSpace(recipient.Email)
	name := strings.TrimSpace(recipient.Name)
	if name == "" {
		return address
	}
	return (&mail.Address{Name: name, Address: address}).String()
}
