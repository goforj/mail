package main

import (
	"fmt"
	"github.com/goforj/mail"
	"github.com/goforj/mail/mailfake"
)

func main() {
	fake := mailfake.New()
	mailer := mail.New(fake, mail.WithDefaultFrom("no-reply@example.com", "Example"))
	fmt.Println(mailer != nil)
	// true
}
