package mailses_test

import (
	"fmt"

	"github.com/goforj/mail/mailses"
)

// ExampleNew demonstrates constructing an SES driver with an injected client.
func ExampleNew() {
	driver, _ := mailses.New(mailses.Config{
		Region:          "us-east-1",
		AccessKeyID:     "test",
		SecretAccessKey: "test",
	})

	fmt.Println(driver != nil)
	// Output: true
}
