package main

import (
	"fmt"
	"github.com/goforj/mail"
)

func main() {
	err := (mail.Message{
		From:    &mail.Recipient{Email: "no-reply@example.com", Name: "Example"},
		To:      []mail.Recipient{{Email: "alice@example.com", Name: "Alice"}},
		Subject: "Welcome",
		Text:    "hello world",
	}).Validate()
	fmt.Println(err == nil)
	// true
}
