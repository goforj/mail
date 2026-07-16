package main

import (
	"fmt"
	"github.com/goforj/mail"
	"github.com/goforj/mail/mailfake"
)

// main keeps this example executable so API drift fails during compilation.
func main() {
	message, _ := mail.New(mailfake.New()).Message().
		From("no-reply@example.com", "Example").
		To("alice@example.com", "Alice").
		Subject("Welcome").
		Text("hello world").
		Header("X-Request-ID", "req_123").
		Tag("welcome").
		Metadata("tenant_id", "tenant_123").
		Build()
	fmt.Println(message.Headers["X-Request-ID"])
	// req_123
}
