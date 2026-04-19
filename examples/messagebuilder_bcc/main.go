package main

import (
	"fmt"
	"github.com/goforj/mail"
	"github.com/goforj/mail/mailfake"
)

func main() {
	msg, _ := mail.New(mailfake.New()).Message().
		To("alice@example.com", "Alice").
		Bcc("audit@example.com", "Audit").
		Subject("Welcome").
		Text("hello world").
		Build()
	fmt.Println(msg.Bcc[0].Email)
	// audit@example.com
}
