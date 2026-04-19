package main

import (
	"fmt"
	"github.com/goforj/mail/mailsendgrid"
)

func main() {
	driver, _ := mailsendgrid.New(mailsendgrid.Config{
		APIKey: "SG.test_key",
	})
	fmt.Println(driver != nil)
	// true
}
