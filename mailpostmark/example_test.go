package mailpostmark_test

import (
	"fmt"

	"github.com/goforj/mail/mailpostmark"
)

func ExampleNew() {
	driver, _ := mailpostmark.New(mailpostmark.Config{
		ServerToken: "pm_test_token",
	})

	fmt.Println(driver != nil)
	// Output: true
}
