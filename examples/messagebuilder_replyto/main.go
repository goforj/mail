package main

import (
	"fmt"
	"github.com/goforj/mail"
	"github.com/goforj/mail/mailfake"
)

func main() {
	msg, _ := mail.New(mailfake.New()).Message().
		To("alice@example.com", "Alice").
		ReplyTo("support@example.com", "Support").
		Subject("Welcome").
		Text("hello world").
		Build()
	fmt.Println(msg.ReplyTo[0].Email)
	// support@example.com
}
