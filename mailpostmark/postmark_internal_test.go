package mailpostmark

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/goforj/mail"
)

func TestAPIErrorAndHelpers(t *testing.T) {
	if got := (&apiError{StatusCode: http.StatusBadGateway}).Error(); got != "mailpostmark: send failed with status 502" {
		t.Fatalf("empty api error = %q", got)
	}

	headers := buildHeaders(map[string]string{
		"":      "skip",
		"X-App": "goforj",
	})
	if len(headers) != 1 || headers[0].Name != "X-App" || headers[0].Value != "goforj" {
		t.Fatalf("buildHeaders() = %#v", headers)
	}
}

func TestDriverSendEarlyBranches(t *testing.T) {
	driver, err := New(Config{
		ServerToken: "pm_test_token",
		Endpoint:    "http://127.0.0.1:1",
	})
	if err != nil {
		t.Fatalf("new driver: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := driver.Send(ctx, mail.Message{}); err == nil || err != context.Canceled {
		t.Fatalf("Send canceled error = %v, want context canceled", err)
	}

	if err := driver.Send(context.Background(), mail.Message{}); err == nil || !strings.Contains(err.Error(), "at least one recipient is required") {
		t.Fatalf("Send validate error = %v", err)
	}

	if err := driver.Send(context.Background(), mail.Message{
		From:    &mail.Recipient{Email: "no-reply@example.com"},
		To:      []mail.Recipient{{Email: "alice@example.com"}},
		Subject: "Welcome",
		Text:    "hello world",
	}); err == nil {
		t.Fatal("expected transport error")
	}

	if err := driver.Send(context.Background(), mail.Message{
		To:      []mail.Recipient{{Email: "alice@example.com"}},
		Subject: "Welcome",
		Text:    "hello world",
	}); err == nil || !strings.Contains(err.Error(), "from is required") {
		t.Fatalf("Send from error = %v", err)
	}
}

func TestDriverSendRejectsInvalidJSONResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("{"))
	}))
	defer server.Close()

	driver, err := New(Config{
		ServerToken: "pm_test_token",
		Endpoint:    server.URL,
		HTTPClient:  server.Client(),
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
