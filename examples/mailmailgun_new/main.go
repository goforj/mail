package main

import (
	"fmt"
	"github.com/goforj/mail/mailmailgun"
)

func main() {
	driver, _ := mailmailgun.New(mailmailgun.Config{
		Domain: "mg.example.com",
		APIKey: "key-test",
	})
	fmt.Println(driver != nil)
	// true
}
