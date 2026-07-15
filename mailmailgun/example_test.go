package mailmailgun_test

import (
	"fmt"

	"github.com/goforj/mail/mailmailgun"
)

// ExampleNew demonstrates constructing a Mailgun driver from its configuration.
func ExampleNew() {
	driver, _ := mailmailgun.New(mailmailgun.Config{
		Domain: "mg.example.com",
		APIKey: "key-test",
	})

	fmt.Println(driver != nil)
	// Output: true
}
