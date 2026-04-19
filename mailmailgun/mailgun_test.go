package mailmailgun_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/goforj/mail"
	"github.com/goforj/mail/mailmailgun"
)

func TestNewRequiresDomainAndAPIKey(t *testing.T) {
	_, err := mailmailgun.New(mailmailgun.Config{APIKey: "key"})
	if err == nil || !strings.Contains(err.Error(), "domain is required") {
		t.Fatalf("new error = %v, want domain error", err)
	}
	_, err = mailmailgun.New(mailmailgun.Config{Domain: "mg.example.com"})
	if err == nil || !strings.Contains(err.Error(), "api key is required") {
		t.Fatalf("new error = %v, want api key error", err)
	}
}

func TestDriverSendPostsExpectedPayload(t *testing.T) {
	var gotAuth string
	var gotPath string
	var values map[string][]string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		username, password, _ := r.BasicAuth()
		gotAuth = username + ":" + password

		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("parse multipart: %v", err)
		}
		values = r.MultipartForm.Value

		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":"mailgun_123","message":"Queued. Thank you."}`)
	}))
	defer server.Close()

	driver, err := mailmailgun.New(mailmailgun.Config{
		Domain:     "mg.example.com",
		APIKey:     "key-test",
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
		Headers: map[string]string{"X-App": "goforj"},
		Tags:    []string{"welcome", "transactional"},
		Metadata: map[string]string{
			"tenant_id": "tenant_123",
		},
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}

	if gotPath != "/v3/mg.example.com/messages" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotAuth != "api:key-test" {
		t.Fatalf("auth = %q", gotAuth)
	}
	if values["from"][0] != `"Example" <no-reply@example.com>` {
		t.Fatalf("from = %#v", values["from"])
	}
	if values["to"][0] != `"Alice" <alice@example.com>` {
		t.Fatalf("to = %#v", values["to"])
	}
	if values["cc"][0] != `"Manager" <manager@example.com>` {
		t.Fatalf("cc = %#v", values["cc"])
	}
	if values["bcc"][0] != `"Audit" <audit@example.com>` {
		t.Fatalf("bcc = %#v", values["bcc"])
	}
	if values["h:Reply-To"][0] != `"Support" <support@example.com>` {
		t.Fatalf("reply-to = %#v", values["h:Reply-To"])
	}
	if values["h:X-App"][0] != "goforj" {
		t.Fatalf("headers = %#v", values)
	}
	if len(values["o:tag"]) != 2 {
		t.Fatalf("tags = %#v", values["o:tag"])
	}
	if values["v:tenant_id"][0] != "tenant_123" {
		t.Fatalf("metadata = %#v", values)
	}
}

func TestDriverSendReturnsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer server.Close()

	driver, err := mailmailgun.New(mailmailgun.Config{
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
	if err == nil || !strings.Contains(err.Error(), "status 400") {
		t.Fatalf("send error = %v, want api error", err)
	}
}
