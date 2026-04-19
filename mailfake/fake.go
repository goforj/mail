package mailfake

import (
	"context"
	"sync"

	"github.com/goforj/mail"
)

// Driver is an in-memory fake driver for tests and examples.
// @group Testing
type Driver struct {
	mu       sync.Mutex
	messages []mail.Message
	err      error
}

// New creates an in-memory fake mail driver for tests.
// @group Testing
//
// Example: record one sent message
//
//	fake := mailfake.New()
//	_ = mail.New(fake).Send(context.Background(), mail.Message{
//		From:    &mail.Recipient{Email: "no-reply@example.com"},
//		To:      []mail.Recipient{{Email: "alice@example.com"}},
//		Subject: "Welcome",
//		Text:    "hello world",
//	})
//	fmt.Println(fake.SentCount())
//	// 1
func New() *Driver {
	return &Driver{}
}

// Send records the message and returns the configured error when set.
// @group Testing
//
// Example: record a sent message directly
//
//	fake := mailfake.New()
//	_ = fake.Send(context.Background(), mail.Message{
//		To:      []mail.Recipient{{Email: "alice@example.com"}},
//		Subject: "Welcome",
//		Text:    "hello world",
//	})
//	fmt.Println(fake.SentCount())
//	// 1
func (m *Driver) Send(_ context.Context, message mail.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.messages = append(m.messages, message.Clone())
	return nil
}

// SetError configures the error returned by future sends.
// @group Testing
//
// Example: force sends to fail
//
//	fake := mailfake.New()
//	fake.SetError(errors.New("boom"))
//	err := fake.Send(context.Background(), mail.Message{
//		To:      []mail.Recipient{{Email: "alice@example.com"}},
//		Subject: "Welcome",
//		Text:    "hello world",
//	})
//	fmt.Println(err != nil)
//	// true
func (m *Driver) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
}

// Reset clears recorded messages and any configured send error.
// @group Testing
//
// Example: clear recorded state
//
//	fake := mailfake.New()
//	_ = fake.Send(context.Background(), mail.Message{
//		To:      []mail.Recipient{{Email: "alice@example.com"}},
//		Subject: "Welcome",
//		Text:    "hello world",
//	})
//	fake.Reset()
//	fmt.Println(fake.SentCount())
//	// 0
func (m *Driver) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = nil
	m.err = nil
}

// Messages returns a copy of every recorded message.
// @group Testing
//
// Example: inspect recorded messages
//
//	fake := mailfake.New()
//	_ = mail.New(fake).Send(context.Background(), mail.Message{
//		From:    &mail.Recipient{Email: "no-reply@example.com"},
//		To:      []mail.Recipient{{Email: "alice@example.com"}},
//		Subject: "Welcome",
//		Text:    "hello world",
//	})
//	fmt.Println(len(fake.Messages()))
//	// 1
func (m *Driver) Messages() []mail.Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]mail.Message, 0, len(m.messages))
	for _, message := range m.messages {
		out = append(out, message.Clone())
	}
	return out
}

// SentCount reports the number of recorded messages.
// @group Testing
//
// Example: count recorded messages
//
//	fake := mailfake.New()
//	_ = fake.Send(context.Background(), mail.Message{
//		To:      []mail.Recipient{{Email: "alice@example.com"}},
//		Subject: "Welcome",
//		Text:    "hello world",
//	})
//	fmt.Println(fake.SentCount())
//	// 1
func (m *Driver) SentCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.messages)
}

// Last returns the last recorded message when one exists.
// @group Testing
//
// Example: read the last recorded subject
//
//	fake := mailfake.New()
//	_ = mail.New(fake).Send(context.Background(), mail.Message{
//		From:    &mail.Recipient{Email: "no-reply@example.com"},
//		To:      []mail.Recipient{{Email: "alice@example.com"}},
//		Subject: "Welcome",
//		Text:    "hello world",
//	})
//	last, _ := fake.Last()
//	fmt.Println(last.Subject)
//	// Welcome
func (m *Driver) Last() (mail.Message, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.messages) == 0 {
		return mail.Message{}, false
	}
	return m.messages[len(m.messages)-1].Clone(), true
}
