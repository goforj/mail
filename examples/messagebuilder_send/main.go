package main

import (
	"context"
	"fmt"
	"github.com/goforj/mail"
	"github.com/goforj/mail/mailfake"
)

func main() {
	fake := mailfake.New()
	_ = mail.New(fake).Message().
		From("no-reply@example.com", "Example").
		To("alice@example.com", "Alice").
		Subject("Welcome").
		Text("hello world").
		Send(context.Background())
	fmt.Println(fake.SentCount())
	// 1
}
