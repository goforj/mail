package mailmailgun

import (
	"context"
	"errors"
	"io"
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

func TestDriverSendAdditionalTransportBranches(t *testing.T) {
	driver := &Driver{
		domain:   "mg.example.com",
		apiKey:   "key-test",
		endpoint: ":",
		client:   http.DefaultClient,
	}
	err := driver.Send(context.Background(), mail.Message{
		From:    &mail.Recipient{Email: "no-reply@example.com"},
		To:      []mail.Recipient{{Email: "alice@example.com"}},
		Subject: "Welcome",
		Text:    "hello world",
	})
	if err == nil || !strings.Contains(err.Error(), "missing protocol scheme") {
		t.Fatalf("invalid endpoint error = %v", err)
	}

	driver = &Driver{
		domain:   "mg.example.com",
		apiKey:   "key-test",
		endpoint: "http://example.com",
		client: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(errReader{err: errors.New("read failed")}),
				Header:     make(http.Header),
			}, nil
		})},
	}
	err = driver.Send(context.Background(), mail.Message{
		From:    &mail.Recipient{Email: "no-reply@example.com"},
		To:      []mail.Recipient{{Email: "alice@example.com"}},
		Subject: "Welcome",
		Text:    "hello world",
	})
	if err == nil || !strings.Contains(err.Error(), "read failed") {
		t.Fatalf("read body error = %v", err)
	}

	driver = &Driver{
		domain:   "mg.example.com",
		apiKey:   "key-test",
		endpoint: "http://example.com",
		client: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("")),
				Header:     make(http.Header),
			}, nil
		})},
	}
	err = driver.Send(context.Background(), mail.Message{
		From:    &mail.Recipient{Email: "no-reply@example.com"},
		To:      []mail.Recipient{{Email: "alice@example.com"}},
		Subject: "Welcome",
		Text:    "hello world",
	})
	if err != nil {
		t.Fatalf("empty success body error = %v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type errReader struct {
	err error
}

func (r errReader) Read([]byte) (int, error) {
	return 0, r.err
}
