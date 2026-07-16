package main

import (
	"fmt"
	"github.com/goforj/mail/mailpostmark"
)

// main keeps this example executable so API drift fails during compilation.
func main() {
	driver, _ := mailpostmark.New(mailpostmark.Config{
		ServerToken: "pm_test_token",
	})
	fmt.Println(driver != nil)
	// true
}
