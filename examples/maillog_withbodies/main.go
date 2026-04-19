package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/goforj/mail"
	"github.com/goforj/mail/maillog"
	"strings"
)

func main() {
	var out bytes.Buffer
	mailer := maillog.New(&out, maillog.WithBodies(true))
	_ = mail.New(mailer).Send(context.Background(), mail.Message{
		From:    &mail.Recipient{Email: "no-reply@example.com"},
		To:      []mail.Recipient{{Email: "alice@example.com"}},
		Subject: "Welcome",
		Text:    "hello world",
	})
	fmt.Println(strings.Contains(out.String(), "\"text\":\"hello world\""))
	// true
}
