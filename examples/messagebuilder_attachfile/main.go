package main

import (
	"fmt"
	"github.com/goforj/mail"
	"github.com/goforj/mail/mailfake"
	"os"
)

func main() {
	_ = os.WriteFile("report.txt", []byte("hello world"), 0o644)
	defer os.Remove("report.txt")
	msg, _ := mail.New(mailfake.New()).Message().
		To("alice@example.com", "Alice").
		Subject("Welcome").
		Text("hello world").
		AttachFile("report.txt").
		Build()
	fmt.Println(msg.Attachments[0].Filename)
	// report.txt
}
