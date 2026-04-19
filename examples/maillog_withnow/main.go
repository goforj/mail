package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/goforj/mail"
	"github.com/goforj/mail/maillog"
	"strings"
	"time"
)

func main() {
	var out bytes.Buffer
	mailer := maillog.New(&out, maillog.WithNow(func() time.Time {
		return time.Date(2026, time.April, 19, 0, 0, 0, 0, time.UTC)
	}))
	_ = mail.New(mailer).Send(context.Background(), mail.Message{
		From:    &mail.Recipient{Email: "no-reply@example.com"},
		To:      []mail.Recipient{{Email: "alice@example.com"}},
		Subject: "Welcome",
		Text:    "hello world",
	})
	fmt.Println(strings.Contains(out.String(), "2026-04-19T00:00:00Z"))
	// true
}
