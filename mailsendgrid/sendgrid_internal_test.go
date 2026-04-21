package mailsendgrid

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/goforj/mail"
)

func TestAPIErrorAndEarlyBranches(t *testing.T) {
	if got := (&apiError{StatusCode: http.StatusBadGateway}).Error(); got != "mailsendgrid: send failed with status 502" {
		t.Fatalf("empty api error = %q", got)
	}

	driver, err := New(Config{APIKey: "SG.test_key"})
	if err != nil {
		t.Fatalf("new driver: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := driver.Send(ctx, mail.Message{}); err == nil || err != context.Canceled {
		t.Fatalf("Send canceled error = %v", err)
	}

	if err := driver.Send(context.Background(), mail.Message{}); err == nil {
		t.Fatal("expected validation error")
	}

	if err := driver.Send(context.Background(), mail.Message{
		To:      []mail.Recipient{{Email: "alice@example.com"}},
		Subject: "Welcome",
		Text:    "hello world",
	}); err == nil || !strings.Contains(err.Error(), "from is required") {
		t.Fatalf("from error = %v", err)
	}
}

func TestDriverSendAdditionalTransportBranches(t *testing.T) {
	driver := &Driver{
		apiKey:   "SG.test_key",
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
		apiKey:   "SG.test_key",
		endpoint: "http://example.com",
		client: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusAccepted,
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
