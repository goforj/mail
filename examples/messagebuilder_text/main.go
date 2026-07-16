package main

import (
	"fmt"
	"github.com/goforj/mail"
	"github.com/goforj/mail/mailfake"
)

// main keeps this example executable so API drift fails during compilation.
func main() {
	msg := mail.New(mailfake.New()).Message().
		To("alice@example.com", "Alice").
		Subject("Welcome").
		Text("hello world").
		Message()
	fmt.Println(msg.Text)
	// hello world
}
