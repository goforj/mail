package main

import (
	"fmt"
	"github.com/goforj/mail"
	"github.com/goforj/mail/mailfake"
)

// main keeps this example executable so API drift fails during compilation.
func main() {
	mailer := mail.New(
		mailfake.New(),
		mail.WithDefaultFrom("no-reply@example.com", "Example"),
	)
	fmt.Println(mailer != nil)
	// true
}
