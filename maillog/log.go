package maillog

import (
	"context"
	"encoding/json"
	"io"
	"sync"
	"time"

	"github.com/goforj/mail"
)

// Driver writes each sent message as one JSON log record.
// @group Logging
type Driver struct {
	writer        io.Writer
	includeBodies bool
	now           func() time.Time
	mu            sync.Mutex
}

// Option customizes a log Driver during construction.
// @group Logging
type Option func(*Driver)

type entry struct {
	SentAt   time.Time         `json:"sent_at"`
	From     *mail.Recipient   `json:"from,omitempty"`
	ReplyTo  []mail.Recipient  `json:"reply_to,omitempty"`
	To       []mail.Recipient  `json:"to,omitempty"`
	Cc       []mail.Recipient  `json:"cc,omitempty"`
	Bcc      []mail.Recipient  `json:"bcc,omitempty"`
	Subject  string            `json:"subject"`
	Headers  map[string]string `json:"headers,omitempty"`
	Tags     []string          `json:"tags,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
	HTML     string            `json:"html,omitempty"`
	Text     string            `json:"text,omitempty"`
}

// New creates a log mail driver that writes one JSON record per sent message.
// @group Logging
//
// Example: log one message to a buffer
//
//	var out bytes.Buffer
//	mailer := maillog.New(&out)
//	_ = mail.New(mailer).Send(context.Background(), mail.Message{
//		From:    &mail.Recipient{Email: "no-reply@example.com"},
//		To:      []mail.Recipient{{Email: "alice@example.com"}},
//		Subject: "Welcome",
//		Text:    "hello world",
//	})
//	fmt.Println(strings.Contains(out.String(), "\"subject\":\"Welcome\""))
//	// true
func New(writer io.Writer, options ...Option) *Driver {
	driver := &Driver{
		writer: writer,
		now:    time.Now,
	}
	for _, option := range options {
		option(driver)
	}
	return driver
}

// WithBodies controls whether HTML and text bodies are included in log output.
// @group Logging
//
// Example: include bodies in log output
//
//	var out bytes.Buffer
//	mailer := maillog.New(&out, maillog.WithBodies(true))
//	_ = mail.New(mailer).Send(context.Background(), mail.Message{
//		From:    &mail.Recipient{Email: "no-reply@example.com"},
//		To:      []mail.Recipient{{Email: "alice@example.com"}},
//		Subject: "Welcome",
//		Text:    "hello world",
//	})
//	fmt.Println(strings.Contains(out.String(), "\"text\":\"hello world\""))
//	// true
func WithBodies(enabled bool) Option {
	return func(driver *Driver) {
		driver.includeBodies = enabled
	}
}

// WithNow overrides the timestamp source used by log entries.
// @group Logging
//
// Example: control the logged timestamp
//
//	var out bytes.Buffer
//	mailer := maillog.New(&out, maillog.WithNow(func() time.Time {
//		return time.Date(2026, time.April, 19, 0, 0, 0, 0, time.UTC)
//	}))
//	_ = mail.New(mailer).Send(context.Background(), mail.Message{
//		From:    &mail.Recipient{Email: "no-reply@example.com"},
//		To:      []mail.Recipient{{Email: "alice@example.com"}},
//		Subject: "Welcome",
//		Text:    "hello world",
//	})
//	fmt.Println(strings.Contains(out.String(), "2026-04-19T00:00:00Z"))
//	// true
func WithNow(now func() time.Time) Option {
	return func(driver *Driver) {
		driver.now = now
	}
}

// Send writes one JSON log record for the message.
// @group Logging
//
// Example: write one log entry directly
//
//	var out bytes.Buffer
//	_ = maillog.New(&out).Send(context.Background(), mail.Message{
//		To:      []mail.Recipient{{Email: "alice@example.com"}},
//		Subject: "Welcome",
//		Text:    "hello world",
//	})
//	fmt.Println(strings.Contains(out.String(), "\"subject\":\"Welcome\""))
//	// true
func (m *Driver) Send(_ context.Context, message mail.Message) error {
	payload := entry{
		SentAt:   m.now().UTC(),
		From:     message.From,
		ReplyTo:  append([]mail.Recipient(nil), message.ReplyTo...),
		To:       append([]mail.Recipient(nil), message.To...),
		Cc:       append([]mail.Recipient(nil), message.Cc...),
		Bcc:      append([]mail.Recipient(nil), message.Bcc...),
		Subject:  message.Subject,
		Headers:  message.Headers,
		Tags:     append([]string(nil), message.Tags...),
		Metadata: message.Metadata,
	}
	if m.includeBodies {
		payload.HTML = message.HTML
		payload.Text = message.Text
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	_, err = m.writer.Write(append(data, '\n'))
	return err
}
