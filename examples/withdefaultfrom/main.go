package main

import (
	"fmt"
	"github.com/goforj/mail"
	"github.com/goforj/mail/mailfake"
)

func main() {
	mailer := mail.New(
		mailfake.New(),
		mail.WithDefaultFrom("no-reply@example.com", "Example"),
	)
	fmt.Println(mailer != nil)
	// true
}
