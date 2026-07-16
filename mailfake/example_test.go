package mailfake_test

import (
	"context"
	"fmt"

	"github.com/goforj/mail"
	"github.com/goforj/mail/mailfake"
)

// ExampleNew demonstrates constructing a concurrency-safe recording fake.
func ExampleNew() {
	fake := mailfake.New()

	_ = fake.Send(context.Background(), mail.Message{
		From:    &mail.Recipient{Email: "no-reply@example.com"},
		To:      []mail.Recipient{{Email: "alice@example.com"}},
		Subject: "Welcome",
		Text:    "hello world",
	})

	fmt.Println(fake.SentCount())
	// Output: 1
}
