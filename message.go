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
	// ErrMissingFrom indicates that a message has no sender.
	ErrMissingFrom = errors.New("mail: from recipient is required")
	// ErrMissingRecipient indicates that a message has no to, cc, or bcc recipients.
	ErrMissingRecipient = errors.New("mail: at least one recipient is required")
	// ErrMissingSubject indicates that a message has no subject.
	ErrMissingSubject = errors.New("mail: subject is required")
	// ErrInvalidSubject indicates that a subject contains characters that are unsafe in a mail header.
	ErrInvalidSubject = errors.New("mail: invalid subject")
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
	// ErrInvalidHeaderValue indicates that a header value contains prohibited control characters.
	ErrInvalidHeaderValue = errors.New("mail: invalid header value")
	// ErrDuplicateHeader indicates that custom headers contain the same name with different casing.
	ErrDuplicateHeader = errors.New("mail: duplicate header")
	// ErrReservedHeader indicates that a custom header would replace transport-owned envelope data.
	ErrReservedHeader = errors.New("mail: reserved header")
	// ErrInvalidAttachment indicates that an attachment is missing required fields.
	ErrInvalidAttachment = errors.New("mail: invalid attachment")
	// ErrInvalidMetadata indicates that a provider metadata key is empty or contains control characters.
	ErrInvalidMetadata = errors.New("mail: invalid metadata")
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
	if m.From == nil {
		return ErrMissingFrom
	}
	if err := validateRecipient(*m.From); err != nil {
		return ErrInvalidFrom
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
	if containsHeaderControl(m.Subject) {
		return ErrInvalidSubject
	}
	if strings.TrimSpace(m.HTML) == "" && strings.TrimSpace(m.Text) == "" {
		return ErrMissingBody
	}
	headerNames := make(map[string]struct{}, len(m.Headers))
	for name, value := range m.Headers {
		if !validHeaderName(name) {
			return ErrInvalidHeaderName
		}
		identity := strings.ToLower(name)
		if _, exists := headerNames[identity]; exists {
			return ErrDuplicateHeader
		}
		headerNames[identity] = struct{}{}
		if reservedHeader(name) {
			return ErrReservedHeader
		}
		if containsHeaderControl(value) {
			return ErrInvalidHeaderValue
		}
	}
	for key := range m.Metadata {
		if key == "" || key != strings.TrimSpace(key) || containsHeaderControl(key) {
			return ErrInvalidMetadata
		}
	}
	for _, attachment := range m.Attachments {
		filename := strings.TrimSpace(attachment.Filename)
		if filename == "" || containsHeaderControl(filename) {
			return ErrInvalidAttachment
		}
		contentType := strings.TrimSpace(attachment.ContentType)
		if contentType == "" || containsHeaderControl(contentType) {
			return ErrInvalidAttachment
		}
		mediaType, _, err := mime.ParseMediaType(contentType)
		if err != nil || strings.TrimSpace(mediaType) == "" {
			return ErrInvalidAttachment
		}
		if attachment.Data == nil {
			return ErrInvalidAttachment
		}
	}
	return nil
}

// validateRecipient requires the Email field to contain one bare address because provider APIs model display names separately.
func validateRecipient(recipient Recipient) error {
	address := strings.TrimSpace(recipient.Email)
	if address == "" || containsHeaderControl(recipient.Name) {
		return ErrInvalidRecipient
	}
	parsed, err := mail.ParseAddress(address)
	if err != nil || parsed.Name != "" || parsed.Address != address {
		return ErrInvalidRecipient
	}
	return nil
}

// formatRecipient delegates display-name quoting and encoded-word handling to the standard library.
func formatRecipient(recipient Recipient) string {
	address := strings.TrimSpace(recipient.Email)
	name := strings.TrimSpace(recipient.Name)
	if name == "" {
		return address
	}
	return (&mail.Address{Name: name, Address: address}).String()
}

// containsHeaderControl rejects ASCII controls before values reach MIME or multipart encoders.
func containsHeaderControl(value string) bool {
	for i := 0; i < len(value); i++ {
		if value[i] < 32 || value[i] == 127 {
			return true
		}
	}
	return false
}

// validHeaderName accepts the printable RFC field-name range while excluding the colon delimiter.
func validHeaderName(name string) bool {
	if name == "" || name != strings.TrimSpace(name) {
		return false
	}
	for i := 0; i < len(name); i++ {
		if name[i] < 33 || name[i] > 126 || name[i] == ':' {
			return false
		}
	}
	return true
}

// reservedHeader keeps the portable envelope authoritative across API and SMTP drivers.
func reservedHeader(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "bcc", "cc", "content-transfer-encoding", "content-type", "from", "mime-version", "reply-to", "subject", "to":
		return true
	default:
		return false
	}
}
