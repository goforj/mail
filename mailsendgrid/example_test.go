package mailsendgrid_test

import (
	"fmt"

	"github.com/goforj/mail/mailsendgrid"
)

// ExampleNew demonstrates constructing a SendGrid driver from its configuration.
func ExampleNew() {
	driver, _ := mailsendgrid.New(mailsendgrid.Config{
		APIKey: "SG.test_key",
	})

	fmt.Println(driver != nil)
	// Output: true
}
