package main

import (
	"fmt"
	"github.com/goforj/mail/mailpostmark"
)

func main() {
	driver, _ := mailpostmark.New(mailpostmark.Config{
		ServerToken: "pm_test_token",
	})
	fmt.Println(driver != nil)
	// true
}
