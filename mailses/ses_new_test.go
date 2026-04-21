package mailses

import (
	"net/http"
	"testing"
)

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
