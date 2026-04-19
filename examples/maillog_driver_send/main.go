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
	_ = maillog.New(&out).Send(context.Background(), mail.Message{
		To:      []mail.Recipient{{Email: "alice@example.com"}},
		Subject: "Welcome",
		Text:    "hello world",
	})
	fmt.Println(strings.Contains(out.String(), "\"subject\":\"Welcome\""))
	// true
}
