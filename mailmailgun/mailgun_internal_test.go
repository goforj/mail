package mailmailgun

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/goforj/mail"
)

func TestAPIErrorAndHelpers(t *testing.T) {
	if got, want := (&apiError{StatusCode: http.StatusBadRequest}).Error(), "mailmailgun: send failed with status 400"; got != want {
		t.Fatalf("apiError(empty) = %q, want %q", got, want)
	}
	if got := formatRecipient(mail.Recipient{Email: "alice@example.com", Name: "Alice"}); got != `"Alice" <alice@example.com>` {
		t.Fatalf("formatRecipient() = %q", got)
	}
	if got := formatRecipient(mail.Recipient{Email: "alice@example.com"}); got != "alice@example.com" {
		t.Fatalf("formatRecipient(no name) = %q", got)
	}
}

func TestDriverSendEarlyBranches(t *testing.T) {
	driver, err := New(Config{
		Domain: "mg.example.com",
		APIKey: "key-test",
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := driver.Send(ctx, mail.Message{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("Send() error = %v, want %v", err, context.Canceled)
	}

	err = driver.Send(context.Background(), mail.Message{
		Subject: "Welcome",
		Text:    "hello world",
	})
	if !errors.Is(err, mail.ErrMissingRecipient) {
		t.Fatalf("Send() error = %v, want %v", err, mail.ErrMissingRecipient)
	}

	driver, err = New(Config{
		Domain:   "mg.example.com",
		APIKey:   "key-test",
		Endpoint: "http://127.0.0.1:1",
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	err = driver.Send(context.Background(), mail.Message{
		From:    &mail.Recipient{Email: "no-reply@example.com"},
		To:      []mail.Recipient{{Email: "alice@example.com"}},
		Subject: "Welcome",
		Text:    "hello world",
	})
	if err == nil || (!strings.Contains(err.Error(), "connect") && !strings.Contains(err.Error(), "refused")) {
		t.Fatalf("Send() transport error = %v", err)
	}

	err = driver.Send(context.Background(), mail.Message{
		To:      []mail.Recipient{{Email: "alice@example.com"}},
		Subject: "Welcome",
		Text:    "hello world",
	})
	if err == nil || !strings.Contains(err.Error(), "from is required") {
		t.Fatalf("Send() from error = %v", err)
	}
}

func TestDriverSendRejectsInvalidJSONResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("{"))
	}))
	defer server.Close()

	driver, err := New(Config{
		Domain:     "mg.example.com",
		APIKey:     "key-test",
		Endpoint:   server.URL,
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("new driver: %v", err)
	}

	err = driver.Send(context.Background(), mail.Message{
		From:    &mail.Recipient{Email: "no-reply@example.com"},
		To:      []mail.Recipient{{Email: "alice@example.com"}},
		Subject: "Welcome",
		Text:    "hello world",
	})
	if err == nil || !strings.Contains(err.Error(), "unexpected end of JSON input") {
		t.Fatalf("send error = %v, want json error", err)
	}
}
