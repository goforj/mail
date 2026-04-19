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
		Attach("report.txt", "text/plain", []byte("hello world")).
		Message()
	fmt.Println(msg.Attachments[0].Filename)
	// report.txt
}
