package main

import (
	"fmt"
	"github.com/goforj/mail"
	"github.com/goforj/mail/mailfake"
)

func main() {
	msg := mail.New(mailfake.New()).Message().
		To("alice@example.com", "Alice").
		Subject("Welcome").
		Text("hello world").
		Metadata("tenant_id", "tenant_123").
		Message()
	fmt.Println(msg.Metadata["tenant_id"])
	// tenant_123
}
