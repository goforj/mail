package main

import (
	"fmt"
	"github.com/goforj/mail/mailsendgrid"
)

// main keeps this example executable so API drift fails during compilation.
func main() {
	driver, _ := mailsendgrid.New(mailsendgrid.Config{
		APIKey: "SG.test_key",
	})
	fmt.Println(driver != nil)
	// true
}
