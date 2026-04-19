package main

import (
	"fmt"
	"github.com/goforj/mail"
)

func main() {
	original := mail.Message{
		To:      []mail.Recipient{{Email: "alice@example.com", Name: "Alice"}},
		Subject: "Welcome",
		Text:    "hello world",
	}
	cloned := original.Clone()
	cloned.Subject = "Changed"
	fmt.Println(original.Subject)
	// Welcome
}
