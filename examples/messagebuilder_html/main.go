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
		HTML("<p>hello world</p>").
		Message()
	fmt.Println(msg.HTML)
	// <p>hello world</p>
}
