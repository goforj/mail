package main

import (
	"context"
	"fmt"
	"github.com/goforj/mail"
	"github.com/goforj/mail/mailsmtp"
)

func main() {
	driver, _ := mailsmtp.New(mailsmtp.Config{
		Host: "smtp.example.com",
		Port: 587,
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
