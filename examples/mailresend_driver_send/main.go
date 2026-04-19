package main

import (
	"context"
	"fmt"
	"github.com/goforj/mail"
	"github.com/goforj/mail/mailresend"
)

func main() {
	driver, _ := mailresend.New(mailresend.Config{
		APIKey:   "re_test_key",
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
