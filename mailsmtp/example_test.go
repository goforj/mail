package mailsmtp_test

import (
	"bytes"
	"fmt"

	"github.com/goforj/mail"
	"github.com/goforj/mail/mailsmtp"
)

func ExampleRender() {
	raw, _ := mailsmtp.Render(mail.Message{
		From:    &mail.Recipient{Email: "no-reply@example.com", Name: "Example"},
		To:      []mail.Recipient{{Email: "alice@example.com", Name: "Alice"}},
		Subject: "Welcome",
		Text:    "hello world",
	})

	fmt.Println(bytes.Contains(raw, []byte("Subject: Welcome")))
	// Output: true
}
