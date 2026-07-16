package mailses

import (
	"net/http"
	"strings"
	"testing"
)

// TestNewAcceptsOptionalClientSettings ensures optional endpoint and credential settings compose when complete.
func TestNewAcceptsOptionalClientSettings(t *testing.T) {
	driver, err := New(Config{
		Region:          "us-east-1",
		AccessKeyID:     "test",
		SecretAccessKey: "test",
		SessionToken:    "session",
		Endpoint:        "http://localhost:9000",
		HTTPClient:      &http.Client{},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if driver == nil || driver.client == nil {
		t.Fatalf("driver = %#v", driver)
	}
}

// TestNewRejectsIncompleteStaticCredentials ensures partial AWS credentials fail before SDK construction.
func TestNewRejectsIncompleteStaticCredentials(t *testing.T) {
	tests := []struct {
		config Config
		want   string
	}{
		{config: Config{Region: "us-east-1", AccessKeyID: "access"}, want: "provided together"},
		{config: Config{Region: "us-east-1", SecretAccessKey: "secret"}, want: "provided together"},
		{config: Config{Region: "us-east-1", SessionToken: "session"}, want: "static credentials"},
	}
	for _, test := range tests {
		if _, err := New(test.config); err == nil || !strings.Contains(err.Error(), test.want) {
			t.Fatalf("New(%#v) error = %v, want %q", test.config, err, test.want)
		}
	}
}

// TestNewRejectsInvalidEndpoint ensures malformed SES endpoints fail during configuration.
func TestNewRejectsInvalidEndpoint(t *testing.T) {
	_, err := New(Config{
		Region:          "us-east-1",
		AccessKeyID:     "test",
		SecretAccessKey: "test",
		Endpoint:        "://bad",
	})
	if err == nil || !strings.Contains(err.Error(), "endpoint must be an absolute HTTP(S) URL") {
		t.Fatalf("New() error = %v, want endpoint error", err)
	}
}
