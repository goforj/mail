package main

import (
	"fmt"
	"github.com/goforj/mail"
	"github.com/goforj/mail/mailsmtp"
	"strings"
)

func main() {
	raw, _ := mailsmtp.Render(mail.Message{
		From:    &mail.Recipient{Email: "no-reply@example.com", Name: "Example"},
		To:      []mail.Recipient{{Email: "alice@example.com", Name: "Alice"}},
		Subject: "Welcome",
		Text:    "hello world",
	})
	fmt.Println(strings.Contains(string(raw), "Subject: Welcome"))
	// true
}
