package main

import (
	"context"
	"fmt"
	"github.com/goforj/mail"
	"github.com/goforj/mail/mailfake"
)

func main() {
	mailer := mail.New(mailfake.New(), mail.WithDefaultFrom("no-reply@example.com", "Example"))
	err := mailer.Send(context.Background(), mail.Message{
		To:      []mail.Recipient{{Email: "alice@example.com", Name: "Alice"}},
		Subject: "Welcome",
		Text:    "hello world",
	})
	fmt.Println(err == nil)
	// true
}
