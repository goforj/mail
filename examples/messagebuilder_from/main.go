package main

import (
	"fmt"
	"github.com/goforj/mail"
	"github.com/goforj/mail/mailfake"
)

// main keeps this example executable so API drift fails during compilation.
func main() {
	msg, _ := mail.New(mailfake.New()).Message().
		From("team@example.com", "Example Team").
		To("alice@example.com", "Alice").
		Subject("Welcome").
		Text("hello world").
		Build()
	fmt.Println(msg.From.Email)
	// team@example.com
}
