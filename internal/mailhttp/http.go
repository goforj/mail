// Package mailhttp centralizes the safety contract shared by API-backed mail drivers.
package mailhttp

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	// MaxResponseBodyBytes caps provider responses so an upstream cannot force unbounded allocation.
	MaxResponseBodyBytes int64 = 1 << 20
)

var (
	// ErrResponseTooLarge indicates that a provider response exceeded MaxResponseBodyBytes.
	ErrResponseTooLarge = errors.New("mail: provider response exceeds 1 MiB")
)

// Endpoint resolves and validates an HTTP endpoint before a driver becomes usable.
func Endpoint(driver, value, fallback string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		value = fallback
	}
	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.User != nil {
		return "", fmt.Errorf("%s: endpoint must be an absolute HTTP(S) URL", driver)
	}
	return value, nil
}

// ReadResponseBody reads a bounded provider response and reports oversized payloads explicitly.
func ReadResponseBody(reader io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(reader, MaxResponseBodyBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > MaxResponseBodyBytes {
		return nil, ErrResponseTooLarge
	}
	return data, nil
}

// RequestID extracts a bounded correlation identifier without reflecting arbitrary response data into errors.
func RequestID(headers http.Header, providerNames ...string) string {
	names := append(append([]string(nil), providerNames...), "X-Request-ID", "Request-ID")
	for _, name := range names {
		value := strings.TrimSpace(headers.Get(name))
		if value == "" || len(value) > 200 {
			continue
		}
		valid := true
		for i := 0; i < len(value); i++ {
			if value[i] < 33 || value[i] > 126 {
				valid = false
				break
			}
		}
		if valid {
			return value
		}
	}
	return ""
}
