package mailsendgrid_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/goforj/mail"
	"github.com/goforj/mail/mailsendgrid"
)

func TestNewRequiresAPIKey(t *testing.T) {
	_, err := mailsendgrid.New(mailsendgrid.Config{})
	if err == nil || !strings.Contains(err.Error(), "api key is required") {
		t.Fatalf("new error = %v, want api key error", err)
	}
}

func TestDriverSendPostsExpectedPayload(t *testing.T) {
	type requestBody struct {
		From struct {
			Email string `json:"email"`
			Name  string `json:"name"`
		} `json:"from"`
		ReplyTo *struct {
			Email string `json:"email"`
			Name  string `json:"name"`
		} `json:"reply_to"`
		Subject          string `json:"subject"`
		Personalizations []struct {
			To []struct {
				Email string `json:"email"`
				Name  string `json:"name"`
			} `json:"to"`
			Cc []struct {
				Email string `json:"email"`
				Name  string `json:"name"`
			} `json:"cc"`
			Bcc []struct {
				Email string `json:"email"`
				Name  string `json:"name"`
			} `json:"bcc"`
			Headers    map[string]string `json:"headers"`
			CustomArgs map[string]string `json:"custom_args"`
		} `json:"personalizations"`
		Content []struct {
			Type  string `json:"type"`
			Value string `json:"value"`
		} `json:"content"`
		Headers    map[string]string `json:"headers"`
		Categories []string          `json:"categories"`
		CustomArgs map[string]string `json:"custom_args"`
	}

	var gotMethod string
	var gotAuth string
	var gotContentType string
	var gotBody requestBody

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotAuth = r.Header.Get("Authorization")
		gotContentType = r.Header.Get("Content-Type")

		data, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if err := json.Unmarshal(data, &gotBody); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}

		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	driver, err := mailsendgrid.New(mailsendgrid.Config{
		APIKey:     "SG.test_key",
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
			"X-App": "goforj",
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
	if gotAuth != "Bearer SG.test_key" {
		t.Fatalf("authorization = %q, want bearer api key", gotAuth)
	}
	if gotContentType != "application/json" {
		t.Fatalf("content type = %q, want application/json", gotContentType)
	}
	if gotBody.From.Email != "no-reply@example.com" || gotBody.From.Name != "Example" {
		t.Fatalf("from = %#v", gotBody.From)
	}
	if gotBody.ReplyTo == nil || gotBody.ReplyTo.Email != "support@example.com" {
		t.Fatalf("reply_to = %#v", gotBody.ReplyTo)
	}
	if gotBody.Subject != "Welcome" {
		t.Fatalf("subject = %q", gotBody.Subject)
	}
	if len(gotBody.Personalizations) != 1 {
		t.Fatalf("personalizations = %#v", gotBody.Personalizations)
	}
	if len(gotBody.Personalizations[0].To) != 1 || gotBody.Personalizations[0].To[0].Email != "alice@example.com" {
		t.Fatalf("to = %#v", gotBody.Personalizations[0].To)
	}
	if len(gotBody.Personalizations[0].Cc) != 1 || gotBody.Personalizations[0].Cc[0].Email != "manager@example.com" {
		t.Fatalf("cc = %#v", gotBody.Personalizations[0].Cc)
	}
	if len(gotBody.Personalizations[0].Bcc) != 1 || gotBody.Personalizations[0].Bcc[0].Email != "audit@example.com" {
		t.Fatalf("bcc = %#v", gotBody.Personalizations[0].Bcc)
	}
	if gotBody.Headers["X-App"] != "goforj" {
		t.Fatalf("headers = %#v", gotBody.Headers)
	}
	if gotBody.Personalizations[0].Headers["X-App"] != "goforj" {
		t.Fatalf("personalization headers = %#v", gotBody.Personalizations[0].Headers)
	}
	if len(gotBody.Categories) != 1 || gotBody.Categories[0] != "welcome" {
		t.Fatalf("categories = %#v", gotBody.Categories)
	}
	if gotBody.CustomArgs["tenant_id"] != "tenant_123" {
		t.Fatalf("custom args = %#v", gotBody.CustomArgs)
	}
	if gotBody.Personalizations[0].CustomArgs["tenant_id"] != "tenant_123" {
		t.Fatalf("personalization custom args = %#v", gotBody.Personalizations[0].CustomArgs)
	}
	if len(gotBody.Content) != 2 {
		t.Fatalf("content = %#v", gotBody.Content)
	}
}

func TestDriverSendReturnsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer server.Close()

	driver, err := mailsendgrid.New(mailsendgrid.Config{
		APIKey:     "SG.test_key",
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
