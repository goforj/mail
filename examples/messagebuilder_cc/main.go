package main

import (
	"fmt"
	"github.com/goforj/mail"
	"github.com/goforj/mail/mailfake"
)

func main() {
	msg, _ := mail.New(mailfake.New()).Message().
		To("alice@example.com", "Alice").
		Cc("manager@example.com", "Manager").
		Subject("Welcome").
		Text("hello world").
		Build()
	fmt.Println(msg.Cc[0].Email)
	// manager@example.com
}
