package maillog_test

import (
	"bytes"
	"context"
	"fmt"

	"github.com/goforj/mail"
	"github.com/goforj/mail/maillog"
)

// ExampleNew demonstrates constructing a structured logging mail driver.
func ExampleNew() {
	var buffer bytes.Buffer
	mailer := maillog.New(&buffer)

	_ = mailer.Send(context.Background(), mail.Message{
		From:    &mail.Recipient{Email: "no-reply@example.com"},
		To:      []mail.Recipient{{Email: "alice@example.com"}},
		Subject: "Welcome",
		Text:    "hello world",
	})

	fmt.Println(bytes.Contains(buffer.Bytes(), []byte(`"subject":"Welcome"`)))
	// Output: true
}
