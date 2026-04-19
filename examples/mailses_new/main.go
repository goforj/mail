package main

import (
	"fmt"
	"github.com/goforj/mail/mailses"
)

func main() {
	driver, _ := mailses.New(mailses.Config{
		Region:          "us-east-1",
		AccessKeyID:     "test",
		SecretAccessKey: "test",
	})
	fmt.Println(driver != nil)
	// true
}
