package mailresend

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/goforj/mail"
)

func TestAPIErrorAndTagHelpers(t *testing.T) {
	if got := (&apiError{StatusCode: http.StatusBadGateway}).Error(); got != "mailresend: send failed with status 502" {
		t.Fatalf("empty api error = %q", got)
	}

	got := buildTags(
		[]string{"hello world", "___"},
		map[string]string{
			"tenant id": "tenant-123",
			"___":       "skip",
		},
	)
	if len(got) != 2 {
		t.Fatalf("buildTags() len = %d, want 2", len(got))
	}
	if got[0].Name != "tenant_id" || got[0].Value != "tenant-123" {
		t.Fatalf("metadata tag = %#v", got[0])
	}
	if got[1].Name != "tag_1" || got[1].Value != "hello_world" {
		t.Fatalf("message tag = %#v", got[1])
	}

	if got := sanitizeTagToken("value", 0); got != "" {
		t.Fatalf("sanitizeTagToken(max<=0) = %q", got)
	}
}

func TestDriverSendEarlyBranches(t *testing.T) {
	driver, err := New(Config{
		APIKey:   "re_test_key",
		Endpoint: "http://127.0.0.1:1",
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
		APIKey:     "re_test_key",
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
