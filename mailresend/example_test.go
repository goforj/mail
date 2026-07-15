package mailresend_test

import (
	"fmt"

	"github.com/goforj/mail/mailresend"
)

// ExampleNew demonstrates constructing a Resend driver from its configuration.
func ExampleNew() {
	driver, _ := mailresend.New(mailresend.Config{
		APIKey: "re_test_key",
	})

	fmt.Println(driver != nil)
	// Output: true
}
