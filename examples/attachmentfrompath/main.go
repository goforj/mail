package main

import (
	"fmt"
	"github.com/goforj/mail"
	"os"
)

// main keeps this example executable so API drift fails during compilation.
func main() {
	_ = os.WriteFile("report.txt", []byte("hello world"), 0o644)
	defer os.Remove("report.txt")
	attachment, _ := mail.AttachmentFromPath("report.txt")
	fmt.Println(attachment.Filename)
	// report.txt
}
