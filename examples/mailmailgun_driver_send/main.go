package main

import (
	"context"
	"fmt"
	"github.com/goforj/mail"
	"github.com/goforj/mail/mailmailgun"
)

// main keeps this example executable so API drift fails during compilation.
func main() {
	driver, _ := mailmailgun.New(mailmailgun.Config{
		Domain:   "mg.example.com",
		APIKey:   "key-test",
		Endpoint: "http://127.0.0.1:1",
	})
	err := driver.Send(context.Background(), mail.Message{
		From:    &mail.Recipient{Email: "no-reply@example.com"},
		To:      []mail.Recipient{{Email: "alice@example.com"}},
		Subject: "Welcome",
		Text:    "hello world",
	})
	fmt.Println(err == nil)
	// false
}
