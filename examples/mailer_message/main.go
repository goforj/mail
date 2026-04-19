package main

import (
	"context"
	"fmt"
	"github.com/goforj/mail"
	"github.com/goforj/mail/mailfake"
)

func main() {
	fake := mailfake.New()
	mailer := mail.New(fake, mail.WithDefaultFrom("no-reply@example.com", "Example"))
	_ = mailer.Message().
		To("alice@example.com", "Alice").
		Subject("Welcome").
		Text("hello world").
		Send(context.Background())
	fmt.Println(fake.SentCount())
	// 1
}
