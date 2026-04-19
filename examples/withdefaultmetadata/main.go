package main

import (
	"fmt"
	"github.com/goforj/mail"
	"github.com/goforj/mail/mailfake"
)

func main() {
	msg, _ := mail.New(
		mailfake.New(),
		mail.WithDefaultMetadata("tenant_id", "tenant_123"),
	).Message().
		To("alice@example.com", "Alice").
		Subject("Welcome").
		Text("hello world").
		Build()
	fmt.Println(msg.Metadata["tenant_id"])
	// tenant_123
}
