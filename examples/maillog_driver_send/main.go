package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/goforj/mail"
	"github.com/goforj/mail/maillog"
	"strings"
)

// main keeps this example executable so API drift fails during compilation.
func main() {
	var out bytes.Buffer
	_ = maillog.New(&out).Send(context.Background(), mail.Message{
		From:    &mail.Recipient{Email: "no-reply@example.com"},
		To:      []mail.Recipient{{Email: "alice@example.com"}},
		Subject: "Welcome",
		Text:    "hello world",
	})
	fmt.Println(strings.Contains(out.String(), "\"subject\":\"Welcome\""))
	// true
}
