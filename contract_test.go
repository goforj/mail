package mail_test

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/goforj/mail"
	"github.com/goforj/mail/mailfake"
	"github.com/goforj/mail/maillog"
	"github.com/goforj/mail/mailmailgun"
	"github.com/goforj/mail/mailpostmark"
	"github.com/goforj/mail/mailresend"
	"github.com/goforj/mail/mailsendgrid"
	"github.com/goforj/mail/mailsmtp"
)

// TestDriversShareValidationContract ensures every provider rejects unsafe messages before external I/O.
func TestDriversShareValidationContract(t *testing.T) {
	var logOutput bytes.Buffer
	drivers := []struct {
		name   string
		driver mail.Driver
	}{
		{name: "fake", driver: mailfake.New()},
		{name: "log", driver: maillog.New(&logOutput)},
		{name: "mailgun", driver: mustMailgunDriver(t)},
		{name: "postmark", driver: mustPostmarkDriver(t)},
		{name: "resend", driver: mustResendDriver(t)},
		{name: "sendgrid", driver: mustSendGridDriver(t)},
		{name: "smtp", driver: mustSMTPDriver(t)},
	}

	message := mail.Message{
		To:      []mail.Recipient{{Email: "alice@example.com"}},
		Subject: "Welcome",
		Text:    "hello world",
	}
	for _, item := range drivers {
		t.Run(item.name, func(t *testing.T) {
			err := item.driver.Send(context.Background(), message)
			if !errors.Is(err, mail.ErrMissingFrom) {
				t.Fatalf("Send() error = %v, want %v", err, mail.ErrMissingFrom)
			}

			duplicateHeaders := validMessage()
			duplicateHeaders.Headers = map[string]string{"X-App": "one", "x-app": "two"}
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			err = item.driver.Send(ctx, duplicateHeaders)
			if !errors.Is(err, mail.ErrDuplicateHeader) {
				t.Fatalf("Send(duplicate headers) error = %v, want %v", err, mail.ErrDuplicateHeader)
			}
		})
	}
}

// TestLocalDriversShareContextContract ensures fake and log transports honor cancellation like remote drivers.
func TestLocalDriversShareContextContract(t *testing.T) {
	var logOutput bytes.Buffer
	drivers := []mail.Driver{mailfake.New(), maillog.New(&logOutput)}
	message := mail.Message{
		From:    &mail.Recipient{Email: "no-reply@example.com"},
		To:      []mail.Recipient{{Email: "alice@example.com"}},
		Subject: "Welcome",
		Text:    "hello world",
	}

	for _, driver := range drivers {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if err := driver.Send(ctx, message); !errors.Is(err, context.Canceled) {
			t.Fatalf("Send() error = %v, want %v", err, context.Canceled)
		}
	}
}

// mustMailgunDriver keeps provider construction noise out of the shared contract table.
func mustMailgunDriver(t *testing.T) mail.Driver {
	t.Helper()
	driver, err := mailmailgun.New(mailmailgun.Config{Domain: "mg.example.com", APIKey: "key"})
	if err != nil {
		t.Fatalf("mailmailgun.New() error = %v", err)
	}
	return driver
}

// mustPostmarkDriver keeps provider construction noise out of the shared contract table.
func mustPostmarkDriver(t *testing.T) mail.Driver {
	t.Helper()
	driver, err := mailpostmark.New(mailpostmark.Config{ServerToken: "token"})
	if err != nil {
		t.Fatalf("mailpostmark.New() error = %v", err)
	}
	return driver
}

// mustResendDriver keeps provider construction noise out of the shared contract table.
func mustResendDriver(t *testing.T) mail.Driver {
	t.Helper()
	driver, err := mailresend.New(mailresend.Config{APIKey: "key"})
	if err != nil {
		t.Fatalf("mailresend.New() error = %v", err)
	}
	return driver
}

// mustSendGridDriver keeps provider construction noise out of the shared contract table.
func mustSendGridDriver(t *testing.T) mail.Driver {
	t.Helper()
	driver, err := mailsendgrid.New(mailsendgrid.Config{APIKey: "key"})
	if err != nil {
		t.Fatalf("mailsendgrid.New() error = %v", err)
	}
	return driver
}

// mustSMTPDriver keeps provider construction noise out of the shared contract table.
func mustSMTPDriver(t *testing.T) mail.Driver {
	t.Helper()
	driver, err := mailsmtp.New(mailsmtp.Config{Host: "smtp.example.com"})
	if err != nil {
		t.Fatalf("mailsmtp.New() error = %v", err)
	}
	return driver
}
