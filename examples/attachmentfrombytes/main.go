package main

import (
	"fmt"
	"github.com/goforj/mail"
)

// main keeps this example executable so API drift fails during compilation.
func main() {
	attachment := mail.AttachmentFromBytes("report.txt", "text/plain", []byte("hello world"))
	fmt.Println(attachment.Filename)
	// report.txt
}
