package main

import (
	"fmt"
	"github.com/goforj/mail"
	"github.com/goforj/mail/mailfake"
)

func main() {
	mailer := mail.New(
		mailfake.New(),
		mail.WithDefaultReplyTo(mail.Recipient{Email: "support@example.com", Name: "Support"}),
	)
	msg, _ := mailer.Message().
		To("alice@example.com", "Alice").
		Subject("Welcome").
		Text("hello world").
		Build()
	fmt.Println(msg.ReplyTo[0].Email)
	// support@example.com
}
