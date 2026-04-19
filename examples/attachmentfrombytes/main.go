package main

import (
	"fmt"
	"github.com/goforj/mail"
)

func main() {
	attachment := mail.AttachmentFromBytes("report.txt", "text/plain", []byte("hello world"))
	fmt.Println(attachment.Filename)
	// report.txt
}
