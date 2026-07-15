package main

import (
	"fmt"
	"github.com/goforj/mail/mailses"
)

// main keeps this example executable so API drift fails during compilation.
func main() {
	driver, _ := mailses.New(mailses.Config{
		Region:          "us-east-1",
		AccessKeyID:     "test",
		SecretAccessKey: "test",
	})
	fmt.Println(driver != nil)
	// true
}
