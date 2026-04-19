package mail

import (
	"errors"
	"net/mail"
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
)

// Recipient identifies one email recipient with an optional display name.
// @group Message Model
type Recipient struct {
	Email string
	Name  string
}

// Message is the canonical portable email envelope used by drivers.
// @group Message Model
type Message struct {
	From     *Recipient
	ReplyTo  []Recipient
	To       []Recipient
	Cc       []Recipient
	Bcc      []Recipient
	Subject  string
	HTML     string
	Text     string
	Headers  map[string]string
	Tags     []string
	Metadata map[string]string
}

// Clone returns a copy of the message safe for reuse in tests and builders.
// @group Message Model
//
// Example: clone before mutating
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
// Example: validate a complete message
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
