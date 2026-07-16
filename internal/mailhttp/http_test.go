package mailhttp

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

// TestEndpointResolvesAndValidatesAbsoluteHTTPURLs ensures provider paths cannot escape a valid HTTP endpoint base.
func TestEndpointResolvesAndValidatesAbsoluteHTTPURLs(t *testing.T) {
	got, err := Endpoint("driver", "", "https://api.example.com/send")
	if err != nil {
		t.Fatalf("Endpoint(default) error = %v", err)
	}
	if got != "https://api.example.com/send" {
		t.Fatalf("Endpoint(default) = %q", got)
	}

	for _, value := range []string{
		"api.example.com/send",
		"ftp://api.example.com/send",
		"https://user:secret@api.example.com/send",
		":",
	} {
		if _, err := Endpoint("driver", value, ""); err == nil {
			t.Fatalf("Endpoint(%q) should fail", value)
		}
	}
}

// TestReadResponseBodyEnforcesBound ensures hostile provider responses cannot consume unbounded memory.
func TestReadResponseBodyEnforcesBound(t *testing.T) {
	data, err := ReadResponseBody(strings.NewReader(strings.Repeat("a", int(MaxResponseBodyBytes))))
	if err != nil {
		t.Fatalf("ReadResponseBody(limit) error = %v", err)
	}
	if int64(len(data)) != MaxResponseBodyBytes {
		t.Fatalf("len(data) = %d, want %d", len(data), MaxResponseBodyBytes)
	}

	_, err = ReadResponseBody(strings.NewReader(strings.Repeat("a", int(MaxResponseBodyBytes+1))))
	if !errors.Is(err, ErrResponseTooLarge) {
		t.Fatalf("ReadResponseBody(over limit) error = %v, want %v", err, ErrResponseTooLarge)
	}
}

// TestRequestIDUsesPrintableBoundedValues ensures response correlation data is safe to log and size-bounded.
func TestRequestIDUsesPrintableBoundedValues(t *testing.T) {
	headers := http.Header{
		"X-Provider-Request-Id": []string{"provider_123"},
		"X-Request-Id":          []string{"generic_123"},
	}
	if got, want := RequestID(headers, "X-Provider-Request-ID"), "provider_123"; got != want {
		t.Fatalf("RequestID() = %q, want %q", got, want)
	}

	headers.Set("X-Provider-Request-ID", strings.Repeat("x", 201))
	if got, want := RequestID(headers, "X-Provider-Request-ID"), "generic_123"; got != want {
		t.Fatalf("RequestID(fallback) = %q, want %q", got, want)
	}
}
