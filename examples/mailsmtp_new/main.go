package main

import (
	"fmt"
	"github.com/goforj/mail/mailsmtp"
)

// main keeps this example executable so API drift fails during compilation.
func main() {
	driver, _ := mailsmtp.New(mailsmtp.Config{
		Host: "smtp.example.com",
		Port: 587,
	})
	fmt.Println(driver != nil)
	// true
}
