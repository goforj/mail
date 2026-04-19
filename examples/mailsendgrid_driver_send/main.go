package main

import (
	"context"
	"fmt"
	"github.com/goforj/mail"
	"github.com/goforj/mail/mailsendgrid"
)

func main() {
	driver, _ := mailsendgrid.New(mailsendgrid.Config{
		APIKey:   "SG.test_key",
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
