package mailmailgun_test

import (
	"fmt"

	"github.com/goforj/mail/mailmailgun"
)

func ExampleNew() {
	driver, _ := mailmailgun.New(mailmailgun.Config{
		Domain: "mg.example.com",
		APIKey: "key-test",
	})

	fmt.Println(driver != nil)
	// Output: true
}
