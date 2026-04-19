package main

import (
	"fmt"
	"github.com/goforj/mail/mailresend"
)

func main() {
	driver, _ := mailresend.New(mailresend.Config{
		APIKey: "re_test_key",
	})
	fmt.Println(driver != nil)
	// true
}
