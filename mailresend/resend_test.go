package mailresend_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/goforj/mail"
	"github.com/goforj/mail/mailresend"
)

func TestNewRequiresAPIKey(t *testing.T) {
	_, err := mailresend.New(mailresend.Config{})
	if err == nil || !strings.Contains(err.Error(), "api key is required") {
		t.Fatalf("new error = %v, want api key error", err)
	}
}

func TestDriverSendPostsExpectedPayload(t *testing.T) {
	type requestBody struct {
		From    string            `json:"from"`
		To      []string          `json:"to"`
		Cc      []string          `json:"cc"`
		Bcc     []string          `json:"bcc"`
		ReplyTo []string          `json:"reply_to"`
		Subject string            `json:"subject"`
		HTML    string            `json:"html"`
		Text    string            `json:"text"`
		Headers map[string]string `json:"headers"`
		Tags    []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"tags"`
	}

	var gotMethod string
	var gotAuth string
	var gotContentType string
	var gotIdempotency string
	var gotBody requestBody

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotAuth = r.Header.Get("Authorization")
		gotContentType = r.Header.Get("Content-Type")
		gotIdempotency = r.Header.Get("Idempotency-Key")

		data, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if err := json.Unmarshal(data, &gotBody); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"email_123"}`))
	}))
	defer server.Close()

	driver, err := mailresend.New(mailresend.Config{
		APIKey:     "re_test_key",
		Endpoint:   server.URL,
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("new driver: %v", err)
	}

	err = driver.Send(context.Background(), mail.Message{
		From:    &mail.Recipient{Email: "no-reply@example.com", Name: "Example"},
		ReplyTo: []mail.Recipient{{Email: "support@example.com", Name: "Support"}},
		To:      []mail.Recipient{{Email: "alice@example.com", Name: "Alice"}},
		Cc:      []mail.Recipient{{Email: "manager@example.com", Name: "Manager"}},
		Bcc:     []mail.Recipient{{Email: "audit@example.com", Name: "Audit"}},
		Subject: "Welcome",
		HTML:    "<p>hello world</p>",
		Text:    "hello world",
		Headers: map[string]string{
			"X-App":           "goforj",
			"Idempotency-Key": "req_123",
		},
		Tags: []string{"welcome"},
		Metadata: map[string]string{
			"tenant_id": "tenant_123",
		},
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("method = %q, want POST", gotMethod)
	}
	if gotAuth != "Bearer re_test_key" {
		t.Fatalf("authorization = %q, want bearer api key", gotAuth)
	}
	if gotContentType != "application/json" {
		t.Fatalf("content type = %q, want application/json", gotContentType)
	}
	if gotIdempotency != "req_123" {
		t.Fatalf("idempotency key = %q, want req_123", gotIdempotency)
	}
	if gotBody.From != `"Example" <no-reply@example.com>` {
		t.Fatalf("from = %q", gotBody.From)
	}
	if len(gotBody.To) != 1 || gotBody.To[0] != "alice@example.com" {
		t.Fatalf("to = %#v", gotBody.To)
	}
	if len(gotBody.Cc) != 1 || gotBody.Cc[0] != "manager@example.com" {
		t.Fatalf("cc = %#v", gotBody.Cc)
	}
	if len(gotBody.Bcc) != 1 || gotBody.Bcc[0] != "audit@example.com" {
		t.Fatalf("bcc = %#v", gotBody.Bcc)
	}
	if len(gotBody.ReplyTo) != 1 || gotBody.ReplyTo[0] != "support@example.com" {
		t.Fatalf("reply_to = %#v", gotBody.ReplyTo)
	}
	if gotBody.Headers["X-App"] != "goforj" {
		t.Fatalf("headers = %#v", gotBody.Headers)
	}
	if _, ok := gotBody.Headers["Idempotency-Key"]; ok {
		t.Fatalf("payload headers should not include idempotency key: %#v", gotBody.Headers)
	}
	if len(gotBody.Tags) != 2 {
		t.Fatalf("tags = %#v", gotBody.Tags)
	}
}

func TestDriverSendReturnsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer server.Close()

	driver, err := mailresend.New(mailresend.Config{
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
	if err == nil || !strings.Contains(err.Error(), "status 400") {
		t.Fatalf("send error = %v, want api error", err)
	}
}
