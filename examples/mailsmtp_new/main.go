package main

import (
	"fmt"
	"github.com/goforj/mail/mailsmtp"
)

func main() {
	driver, _ := mailsmtp.New(mailsmtp.Config{
		Host: "smtp.example.com",
		Port: 587,
	})
	fmt.Println(driver != nil)
	// true
}
