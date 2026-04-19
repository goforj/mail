package mail

import "context"

// MessageBuilder composes one message fluently before build or send.
// @group Composition
type MessageBuilder struct {
	mailer  *Mailer
	message Message
	err     error
}

// From sets the from recipient.
// @group Composition
//
// Example: from
//
//	msg, _ := mail.New(mailfake.New()).Message().
//		From("team@example.com", "Example Team").
//		To("alice@example.com", "Alice").
//		Subject("Welcome").
//		Text("hello world").
//		Build()
//	fmt.Println(msg.From.Email)
//	// team@example.com
func (b *MessageBuilder) From(email, name string) *MessageBuilder {
	b.message.From = &Recipient{Email: email, Name: name}
	return b
}

// ReplyTo appends one reply-to recipient.
// @group Composition
//
// Example: reply-to
//
//	msg, _ := mail.New(mailfake.New()).Message().
//		To("alice@example.com", "Alice").
//		ReplyTo("support@example.com", "Support").
//		Subject("Welcome").
//		Text("hello world").
//		Build()
//	fmt.Println(msg.ReplyTo[0].Email)
//	// support@example.com
func (b *MessageBuilder) ReplyTo(email, name string) *MessageBuilder {
	b.message.ReplyTo = append(b.message.ReplyTo, Recipient{Email: email, Name: name})
	return b
}

// To appends one primary recipient.
// @group Composition
//
// Example: to
//
//	msg, _ := mail.New(mailfake.New()).Message().
//		To("alice@example.com", "Alice").
//		Subject("Welcome").
//		Text("hello world").
//		Build()
//	fmt.Println(len(msg.To))
//	// 1
func (b *MessageBuilder) To(email, name string) *MessageBuilder {
	b.message.To = append(b.message.To, Recipient{Email: email, Name: name})
	return b
}

// Cc appends one carbon-copy recipient.
// @group Composition
//
// Example: cc
//
//	msg, _ := mail.New(mailfake.New()).Message().
//		To("alice@example.com", "Alice").
//		Cc("manager@example.com", "Manager").
//		Subject("Welcome").
//		Text("hello world").
//		Build()
//	fmt.Println(msg.Cc[0].Email)
//	// manager@example.com
func (b *MessageBuilder) Cc(email, name string) *MessageBuilder {
	b.message.Cc = append(b.message.Cc, Recipient{Email: email, Name: name})
	return b
}

// Bcc appends one blind-carbon-copy recipient.
// @group Composition
//
// Example: bcc
//
//	msg, _ := mail.New(mailfake.New()).Message().
//		To("alice@example.com", "Alice").
//		Bcc("audit@example.com", "Audit").
//		Subject("Welcome").
//		Text("hello world").
//		Build()
//	fmt.Println(msg.Bcc[0].Email)
//	// audit@example.com
func (b *MessageBuilder) Bcc(email, name string) *MessageBuilder {
	b.message.Bcc = append(b.message.Bcc, Recipient{Email: email, Name: name})
	return b
}

// Subject sets the message subject.
// @group Content
//
// Example: subject
//
//	msg := mail.New(mailfake.New()).Message().
//		To("alice@example.com", "Alice").
//		Subject("Welcome").
//		Text("hello world").
//		Message()
//	fmt.Println(msg.Subject)
//	// Welcome
func (b *MessageBuilder) Subject(value string) *MessageBuilder {
	b.message.Subject = value
	return b
}

// HTML sets the HTML body.
// @group Content
//
// Example: html
//
//	msg := mail.New(mailfake.New()).Message().
//		To("alice@example.com", "Alice").
//		Subject("Welcome").
//		HTML("<p>hello world</p>").
//		Message()
//	fmt.Println(msg.HTML)
//	// <p>hello world</p>
func (b *MessageBuilder) HTML(value string) *MessageBuilder {
	b.message.HTML = value
	return b
}

// Text sets the plain text body.
// @group Content
//
// Example: text
//
//	msg := mail.New(mailfake.New()).Message().
//		To("alice@example.com", "Alice").
//		Subject("Welcome").
//		Text("hello world").
//		Message()
//	fmt.Println(msg.Text)
//	// hello world
func (b *MessageBuilder) Text(value string) *MessageBuilder {
	b.message.Text = value
	return b
}

// Header sets or replaces one message header.
// @group Content
//
// Example: header, tag, metadata
//
//	message, _ := mail.New(mailfake.New()).Message().
//		To("alice@example.com", "Alice").
//		Subject("Welcome").
//		Text("hello world").
//		Header("X-Request-ID", "req_123").
//		Tag("welcome").
//		Metadata("tenant_id", "tenant_123").
//		Build()
//	fmt.Println(message.Headers["X-Request-ID"])
//	// req_123
func (b *MessageBuilder) Header(key, value string) *MessageBuilder {
	if b.message.Headers == nil {
		b.message.Headers = map[string]string{}
	}
	b.message.Headers[key] = value
	return b
}

// Tag appends one provider-facing message tag.
// @group Content
//
// Example: tag
//
//	msg := mail.New(mailfake.New()).Message().
//		To("alice@example.com", "Alice").
//		Subject("Welcome").
//		Text("hello world").
//		Tag("welcome").
//		Message()
//	fmt.Println(msg.Tags[0])
//	// welcome
func (b *MessageBuilder) Tag(value string) *MessageBuilder {
	b.message.Tags = append(b.message.Tags, value)
	return b
}

// Metadata sets one provider-facing metadata key/value pair.
// @group Content
//
// Example: metadata
//
//	msg := mail.New(mailfake.New()).Message().
//		To("alice@example.com", "Alice").
//		Subject("Welcome").
//		Text("hello world").
//		Metadata("tenant_id", "tenant_123").
//		Message()
//	fmt.Println(msg.Metadata["tenant_id"])
//	// tenant_123
func (b *MessageBuilder) Metadata(key, value string) *MessageBuilder {
	if b.message.Metadata == nil {
		b.message.Metadata = map[string]string{}
	}
	b.message.Metadata[key] = value
	return b
}

// Attach appends one in-memory attachment.
// @group Content
//
// Example: attach
//
//	msg := mail.New(mailfake.New()).Message().
//		To("alice@example.com", "Alice").
//		Subject("Welcome").
//		Text("hello world").
//		Attach("report.txt", "text/plain", []byte("hello world")).
//		Message()
//	fmt.Println(msg.Attachments[0].Filename)
//	// report.txt
func (b *MessageBuilder) Attach(filename, contentType string, data []byte) *MessageBuilder {
	b.message.Attachments = append(b.message.Attachments, AttachmentFromBytes(filename, contentType, data))
	return b
}

// AttachFile loads one attachment from disk and appends it to the message.
// @group Content
//
// Example: attach file
//
//	_ = os.WriteFile("report.txt", []byte("hello world"), 0o644)
//	defer os.Remove("report.txt")
//	msg, _ := mail.New(mailfake.New()).Message().
//		To("alice@example.com", "Alice").
//		Subject("Welcome").
//		Text("hello world").
//		AttachFile("report.txt").
//		Build()
//	fmt.Println(msg.Attachments[0].Filename)
//	// report.txt
func (b *MessageBuilder) AttachFile(path string) *MessageBuilder {
	if b.err != nil {
		return b
	}
	attachment, err := AttachmentFromPath(path)
	if err != nil {
		b.err = err
		return b
	}
	b.message.Attachments = append(b.message.Attachments, attachment)
	return b
}

// Message returns the currently composed message without applying mailer defaults.
// @group Composition
//
// Example: current message
//
//	msg := mail.New(mailfake.New()).Message().
//		To("alice@example.com", "Alice").
//		Subject("Welcome").
//		Text("hello world").
//		Message()
//	fmt.Println(msg.Subject)
//	// Welcome
func (b *MessageBuilder) Message() Message {
	return b.message.Clone()
}

// Build applies defaults, validates, and returns the composed message without sending it.
// @group Delivery
//
// Example: build
//
//	msg, _ := mail.New(
//		mailfake.New(),
//		mail.WithDefaultFrom("no-reply@example.com", "Example"),
//	).Message().
//		To("alice@example.com", "Alice").
//		Subject("Welcome").
//		Text("hello world").
//		Build()
//	fmt.Println(msg.From.Email)
//	// no-reply@example.com
func (b *MessageBuilder) Build() (Message, error) {
	if b.err != nil {
		return Message{}, b.err
	}
	if b.mailer == nil {
		msg := b.message.Clone()
		if err := msg.Validate(); err != nil {
			return Message{}, err
		}
		return msg, nil
	}
	msg := b.mailer.applyDefaults(b.message)
	if err := msg.Validate(); err != nil {
		return Message{}, err
	}
	return msg, nil
}

// Send delegates the composed message to the bound mailer.
// @group Delivery
//
// Example: send
//
//	fake := mailfake.New()
//	_ = mail.New(fake).Message().
//		From("no-reply@example.com", "Example").
//		To("alice@example.com", "Alice").
//		Subject("Welcome").
//		Text("hello world").
//		Send(context.Background())
//	fmt.Println(fake.SentCount())
//	// 1
func (b *MessageBuilder) Send(ctx context.Context) error {
	if b.err != nil {
		return b.err
	}
	if b.mailer == nil {
		return ErrMissingMailer
	}
	return b.mailer.Send(ctx, b.message)
}
