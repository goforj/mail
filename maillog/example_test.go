package maillog_test

import (
	"bytes"
	"context"
	"fmt"

	"github.com/goforj/mail"
	"github.com/goforj/mail/maillog"
)

func ExampleNew() {
	var buffer bytes.Buffer
	mailer := maillog.New(&buffer)

	_ = mailer.Send(context.Background(), mail.Message{
		To:      []mail.Recipient{{Email: "alice@example.com"}},
		Subject: "Welcome",
		Text:    "hello world",
	})

	fmt.Println(bytes.Contains(buffer.Bytes(), []byte(`"subject":"Welcome"`)))
	// Output: true
}
