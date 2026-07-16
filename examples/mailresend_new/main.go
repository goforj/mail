package main

import (
	"fmt"
	"github.com/goforj/mail/mailresend"
)

// main keeps this example executable so API drift fails during compilation.
func main() {
	driver, _ := mailresend.New(mailresend.Config{
		APIKey: "re_test_key",
	})
	fmt.Println(driver != nil)
	// true
}
