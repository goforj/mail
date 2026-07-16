package mail_test

import (
	"context"
	"fmt"

	"github.com/goforj/mail"
	"github.com/goforj/mail/mailfake"
)

// ExampleNew demonstrates constructing a mailer with an injected driver.
func ExampleNew() {
	fake := mailfake.New()
	mailer := mail.New(
		fake,
		mail.WithDefaultFrom("no-reply@example.com", "Example"),
	)

	_ = mailer.Message().
		To("alice@example.com", "Alice").
		Subject("Welcome").
		Text("hello world").
		Send(context.Background())

	fmt.Println(fake.SentCount())
	// Output: 1
}

// ExampleMailer_Message demonstrates starting a fluent message from a mailer.
func ExampleMailer_Message() {
	fake := mailfake.New()
	mailer := mail.New(fake, mail.WithDefaultFrom("no-reply@example.com", "Example"))

	message, _ := mailer.Message().
		To("alice@example.com", "Alice").
		Subject("Welcome").
		Text("hello world").
		Build()

	fmt.Println(message.Subject, len(message.To))
	// Output: Welcome 1
}
