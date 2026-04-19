package mail

import (
	"context"
)

// Driver delivers one validated message through a concrete backend.
// @group Delivery
type Driver interface {
	Send(context.Context, Message) error
}

// Mailer composes messages, applies defaults, and delegates final delivery to a driver.
// @group Composition
type Mailer struct {
	driver   Driver
	defaults Message
}

// Option customizes a Mailer during construction.
// @group Defaults
type Option func(*Mailer)

// New creates a Mailer backed by the provided driver.
// @group Construction
//
// Example: with a default from
//
//	fake := mailfake.New()
//	mailer := mail.New(fake, mail.WithDefaultFrom("no-reply@example.com", "Example"))
//	fmt.Println(mailer != nil)
//	// true
func New(driver Driver, options ...Option) *Mailer {
	mailer := &Mailer{driver: driver}
	for _, option := range options {
		option(mailer)
	}
	return mailer
}

// WithDefaultFrom configures the default from recipient applied when a message omits one.
// @group Defaults
//
// Example: default from
//
//	mailer := mail.New(
//		mailfake.New(),
//		mail.WithDefaultFrom("no-reply@example.com", "Example"),
//	)
//	fmt.Println(mailer != nil)
//	// true
func WithDefaultFrom(email, name string) Option {
	return func(mailer *Mailer) {
		mailer.defaults.From = &Recipient{Email: email, Name: name}
	}
}

// WithDefaultReplyTo configures the default reply-to recipients applied when a message omits them.
// @group Defaults
//
// Example: default reply-to
//
//	mailer := mail.New(
//		mailfake.New(),
//		mail.WithDefaultReplyTo(mail.Recipient{Email: "support@example.com", Name: "Support"}),
//	)
//	msg, _ := mailer.Message().
//		To("alice@example.com", "Alice").
//		Subject("Welcome").
//		Text("hello world").
//		Build()
//	fmt.Println(msg.ReplyTo[0].Email)
//	// support@example.com
func WithDefaultReplyTo(recipients ...Recipient) Option {
	return func(mailer *Mailer) {
		mailer.defaults.ReplyTo = append([]Recipient(nil), recipients...)
	}
}

// WithDefaultHeader configures a header applied when a message omits that header key.
// @group Defaults
//
// Example: default header
//
//	msg, _ := mail.New(
//		mailfake.New(),
//		mail.WithDefaultHeader("X-App", "goforj"),
//	).Message().
//		To("alice@example.com", "Alice").
//		Subject("Welcome").
//		Text("hello world").
//		Build()
//	fmt.Println(msg.Headers["X-App"])
//	// goforj
func WithDefaultHeader(key, value string) Option {
	return func(mailer *Mailer) {
		if mailer.defaults.Headers == nil {
			mailer.defaults.Headers = map[string]string{}
		}
		mailer.defaults.Headers[key] = value
	}
}

// WithDefaultTag configures a tag prepended to every message sent by the mailer.
// @group Defaults
//
// Example: default tag
//
//	msg, _ := mail.New(
//		mailfake.New(),
//		mail.WithDefaultTag("transactional"),
//	).Message().
//		To("alice@example.com", "Alice").
//		Subject("Welcome").
//		Text("hello world").
//		Build()
//	fmt.Println(msg.Tags[0])
//	// transactional
func WithDefaultTag(tag string) Option {
	return func(mailer *Mailer) {
		mailer.defaults.Tags = append(mailer.defaults.Tags, tag)
	}
}

// WithDefaultMetadata configures metadata applied when a message omits that metadata key.
// @group Defaults
//
// Example: default metadata
//
//	msg, _ := mail.New(
//		mailfake.New(),
//		mail.WithDefaultMetadata("tenant_id", "tenant_123"),
//	).Message().
//		To("alice@example.com", "Alice").
//		Subject("Welcome").
//		Text("hello world").
//		Build()
//	fmt.Println(msg.Metadata["tenant_id"])
//	// tenant_123
func WithDefaultMetadata(key, value string) Option {
	return func(mailer *Mailer) {
		if mailer.defaults.Metadata == nil {
			mailer.defaults.Metadata = map[string]string{}
		}
		mailer.defaults.Metadata[key] = value
	}
}

// Message starts a new fluent message builder bound to this mailer.
// @group Composition
//
// Example: send one message
//
//	fake := mailfake.New()
//	mailer := mail.New(fake, mail.WithDefaultFrom("no-reply@example.com", "Example"))
//	_ = mailer.Message().
//		To("alice@example.com", "Alice").
//		Subject("Welcome").
//		Text("hello world").
//		Send(context.Background())
//	fmt.Println(fake.SentCount())
//	// 1
func (m *Mailer) Message() *MessageBuilder {
	return &MessageBuilder{
		mailer:  m,
		message: Message{},
	}
}

// Send validates the message, applies defaults, and delegates delivery to the driver.
// @group Delivery
//
// Example: send a message
//
//	mailer := mail.New(mailfake.New(), mail.WithDefaultFrom("no-reply@example.com", "Example"))
//	err := mailer.Send(context.Background(), mail.Message{
//		To:      []mail.Recipient{{Email: "alice@example.com", Name: "Alice"}},
//		Subject: "Welcome",
//		Text:    "hello world",
//	})
//	fmt.Println(err == nil)
//	// true
func (m *Mailer) Send(ctx context.Context, msg Message) error {
	if m.driver == nil {
		return ErrMissingMailer
	}
	resolved := m.applyDefaults(msg)
	if err := resolved.Validate(); err != nil {
		return err
	}
	return m.driver.Send(ctx, resolved)
}

func (m *Mailer) applyDefaults(msg Message) Message {
	resolved := msg.Clone()
	if resolved.From == nil && m.defaults.From != nil {
		from := *m.defaults.From
		resolved.From = &from
	}
	if len(resolved.ReplyTo) == 0 && len(m.defaults.ReplyTo) > 0 {
		resolved.ReplyTo = append([]Recipient(nil), m.defaults.ReplyTo...)
	}
	if len(m.defaults.Headers) > 0 {
		if resolved.Headers == nil {
			resolved.Headers = map[string]string{}
		}
		for key, value := range m.defaults.Headers {
			if _, exists := resolved.Headers[key]; !exists {
				resolved.Headers[key] = value
			}
		}
	}
	if len(m.defaults.Tags) > 0 {
		resolved.Tags = append(append([]string(nil), m.defaults.Tags...), resolved.Tags...)
	}
	if len(m.defaults.Metadata) > 0 {
		if resolved.Metadata == nil {
			resolved.Metadata = map[string]string{}
		}
		for key, value := range m.defaults.Metadata {
			if _, exists := resolved.Metadata[key]; !exists {
				resolved.Metadata[key] = value
			}
		}
	}
	return resolved
}
