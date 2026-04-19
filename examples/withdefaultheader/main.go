package main

import (
	"fmt"
	"github.com/goforj/mail"
	"github.com/goforj/mail/mailfake"
)

func main() {
	msg, _ := mail.New(
		mailfake.New(),
		mail.WithDefaultHeader("X-App", "goforj"),
	).Message().
		To("alice@example.com", "Alice").
		Subject("Welcome").
		Text("hello world").
		Build()
	fmt.Println(msg.Headers["X-App"])
	// goforj
}
