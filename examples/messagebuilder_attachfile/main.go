package main

import (
	"fmt"
	"github.com/goforj/mail"
	"github.com/goforj/mail/mailfake"
	"os"
)

// main keeps this example executable so API drift fails during compilation.
func main() {
	_ = os.WriteFile("report.txt", []byte("hello world"), 0o644)
	defer os.Remove("report.txt")
	msg, _ := mail.New(mailfake.New()).Message().
		From("no-reply@example.com", "Example").
		To("alice@example.com", "Alice").
		Subject("Welcome").
		Text("hello world").
		AttachFile("report.txt").
		Build()
	fmt.Println(msg.Attachments[0].Filename)
	// report.txt
}
