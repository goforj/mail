package main

import (
	"fmt"
	"github.com/goforj/mail"
	"github.com/goforj/mail/mailfake"
)

func main() {
	msg := mail.New(mailfake.New()).Message().
		To("alice@example.com", "Alice").
		Subject("Welcome").
		Text("hello world").
		Tag("welcome").
		Message()
	fmt.Println(msg.Tags[0])
	// welcome
}
