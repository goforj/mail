package main

import (
	"fmt"
	"github.com/goforj/mail"
	"github.com/goforj/mail/mailfake"
)

// main keeps this example executable so API drift fails during compilation.
func main() {
	msg, _ := mail.New(mailfake.New()).Message().
		From("no-reply@example.com", "Example").
		To("alice@example.com", "Alice").
		Cc("manager@example.com", "Manager").
		Subject("Welcome").
		Text("hello world").
		Build()
	fmt.Println(msg.Cc[0].Email)
	// manager@example.com
}
