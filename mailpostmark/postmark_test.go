package mailpostmark_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/goforj/mail"
	"github.com/goforj/mail/mailpostmark"
)

func TestNewRequiresServerToken(t *testing.T) {
	_, err := mailpostmark.New(mailpostmark.Config{})
	if err == nil || !strings.Contains(err.Error(), "server token is required") {
		t.Fatalf("new error = %v, want server token error", err)
	}
}

func TestDriverSendPostsExpectedPayload(t *testing.T) {
	type payload struct {
		From          string            `json:"From"`
		To            string            `json:"To"`
		Cc            string            `json:"Cc"`
		Bcc           string            `json:"Bcc"`
		ReplyTo       string            `json:"ReplyTo"`
		Subject       string            `json:"Subject"`
		HTMLBody      string            `json:"HtmlBody"`
		TextBody      string            `json:"TextBody"`
		Headers       []map[string]any  `json:"Headers"`
		Attachments   []map[string]any  `json:"Attachments"`
		Tag           string            `json:"Tag"`
		Metadata      map[string]string `json:"Metadata"`
		MessageStream string            `json:"MessageStream"`
	}

	var gotAuth string
	var gotAccept string
	var gotContentType string
	var gotBody payload

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("X-Postmark-Server-Token")
		gotAccept = r.Header.Get("Accept")
		gotContentType = r.Header.Get("Content-Type")

		data, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if err := json.Unmarshal(data, &gotBody); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"MessageID":"msg_123","ErrorCode":0,"Message":"OK"}`))
	}))
	defer server.Close()

	driver, err := mailpostmark.New(mailpostmark.Config{
		ServerToken:   "pm_test_token",
		Endpoint:      server.URL,
		MessageStream: "outbound",
		HTTPClient:    server.Client(),
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
		Attachments: []mail.Attachment{
			mail.AttachmentFromBytes("report.txt", "text/plain", []byte("hello attachment")),
		},
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}

	if gotAuth != "pm_test_token" {
		t.Fatalf("server token = %q", gotAuth)
	}
	if gotAccept != "application/json" {
		t.Fatalf("accept = %q", gotAccept)
	}
	if gotContentType != "application/json" {
		t.Fatalf("content type = %q", gotContentType)
	}
	if gotBody.From != `"Example" <no-reply@example.com>` {
		t.Fatalf("from = %q", gotBody.From)
	}
	if gotBody.To != `"Alice" <alice@example.com>` {
		t.Fatalf("to = %q", gotBody.To)
	}
	if gotBody.Cc != `"Manager" <manager@example.com>` {
		t.Fatalf("cc = %q", gotBody.Cc)
	}
	if gotBody.Bcc != `"Audit" <audit@example.com>` {
		t.Fatalf("bcc = %q", gotBody.Bcc)
	}
	if gotBody.ReplyTo != `"Support" <support@example.com>` {
		t.Fatalf("reply_to = %q", gotBody.ReplyTo)
	}
	if gotBody.Tag != "welcome" {
		t.Fatalf("tag = %q", gotBody.Tag)
	}
	if gotBody.Metadata["tenant_id"] != "tenant_123" || gotBody.Metadata["tag_2"] != "transactional" {
		t.Fatalf("metadata = %#v", gotBody.Metadata)
	}
	if gotBody.MessageStream != "outbound" {
		t.Fatalf("message stream = %q", gotBody.MessageStream)
	}
	if len(gotBody.Attachments) != 1 || gotBody.Attachments[0]["Name"] != "report.txt" {
		t.Fatalf("attachments = %#v", gotBody.Attachments)
	}
}

func TestDriverSendReturnsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer server.Close()

	driver, err := mailpostmark.New(mailpostmark.Config{
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
	if err == nil || !strings.Contains(err.Error(), "status 400") {
		t.Fatalf("send error = %v, want api error", err)
	}
}
