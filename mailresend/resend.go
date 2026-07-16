package mailresend

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	stdmail "net/mail"
	"sort"
	"strconv"
	"strings"

	"github.com/goforj/mail"
	"github.com/goforj/mail/internal/mailhttp"
)

const defaultEndpoint = "https://api.resend.com/emails"

// Config configures Resend delivery.
// @group Resend
type Config struct {
	APIKey     string
	Endpoint   string
	HTTPClient *http.Client
}

// Driver sends messages through the Resend Email API.
// @group Resend
type Driver struct {
	apiKey   string
	endpoint string
	client   *http.Client
}

type sendRequest struct {
	From        string            `json:"from"`
	To          []string          `json:"to"`
	Cc          []string          `json:"cc,omitempty"`
	Bcc         []string          `json:"bcc,omitempty"`
	ReplyTo     []string          `json:"reply_to,omitempty"`
	Subject     string            `json:"subject"`
	HTML        string            `json:"html,omitempty"`
	Text        string            `json:"text,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Tags        []tag             `json:"tags,omitempty"`
	Attachments []attachment      `json:"attachments,omitempty"`
}

type tag struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type attachment struct {
	Filename    string `json:"filename"`
	Content     string `json:"content"`
	ContentType string `json:"content_type,omitempty"`
}

type sendResponse struct {
	ID string `json:"id"`
}

// ResponseError describes a non-success response returned by Resend.
// @group Resend
type ResponseError struct {
	StatusCode int
	RequestID  string
}

// Error formats the provider status and safe correlation identifier without including response content.
// @group Resend
func (e *ResponseError) Error() string {
	if e.RequestID == "" {
		return fmt.Sprintf("mailresend: send failed with status %d", e.StatusCode)
	}
	return fmt.Sprintf("mailresend: send failed with status %d (request %s)", e.StatusCode, e.RequestID)
}

// New creates a Resend mail driver from the given config.
// @group Resend
//
// Example: resend
//
//	driver, _ := mailresend.New(mailresend.Config{
//		APIKey: "re_test_key",
//	})
//	fmt.Println(driver != nil)
//	// true
func New(config Config) (*Driver, error) {
	apiKey := strings.TrimSpace(config.APIKey)
	if apiKey == "" {
		return nil, fmt.Errorf("mailresend: api key is required")
	}
	endpoint, err := mailhttp.Endpoint("mailresend", config.Endpoint, defaultEndpoint)
	if err != nil {
		return nil, err
	}

	client := config.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}

	return &Driver{
		apiKey:   apiKey,
		endpoint: endpoint,
		client:   client,
	}, nil
}

// Send validates and transmits one message through Resend.
// @group Resend
//
// Example: send
//
//	driver, _ := mailresend.New(mailresend.Config{
//		APIKey:   "re_test_key",
//		Endpoint: "http://127.0.0.1:1",
//	})
//	err := driver.Send(context.Background(), mail.Message{
//		From:    &mail.Recipient{Email: "no-reply@example.com"},
//		To:      []mail.Recipient{{Email: "alice@example.com"}},
//		Subject: "Welcome",
//		Text:    "hello world",
//	})
//	fmt.Println(err == nil)
//	// false
func (d *Driver) Send(ctx context.Context, message mail.Message) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := message.Validate(); err != nil {
		return err
	}

	payload := sendRequest{
		From:    formatRecipient(*message.From),
		To:      recipientEmails(message.To),
		Cc:      recipientEmails(message.Cc),
		Bcc:     recipientEmails(message.Bcc),
		ReplyTo: recipientEmails(message.ReplyTo),
		Subject: strings.TrimSpace(message.Subject),
		HTML:    message.HTML,
		Text:    message.Text,
	}

	headers := copyHeaders(message.Headers)
	if len(headers) > 0 {
		deleteHeaderFold(headers, "Idempotency-Key")
		if len(headers) > 0 {
			payload.Headers = headers
		}
	}

	if tags := buildTags(message.Tags, message.Metadata); len(tags) > 0 {
		payload.Tags = tags
	}
	if attachments := buildAttachments(message.Attachments); len(attachments) > 0 {
		payload.Attachments = attachments
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.endpoint, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+d.apiKey)
	req.Header.Set("Content-Type", "application/json")

	if key, ok := idempotencyKey(message.Headers); ok {
		req.Header.Set("Idempotency-Key", key)
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, bodyErr := mailhttp.ReadResponseBody(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &ResponseError{
			StatusCode: resp.StatusCode,
			RequestID:  mailhttp.RequestID(resp.Header, "X-Resend-Request-ID"),
		}
	}
	if bodyErr != nil {
		return bodyErr
	}

	var decoded sendResponse
	if len(body) > 0 {
		if err := json.Unmarshal(body, &decoded); err != nil {
			return err
		}
	}
	return nil
}

// recipientEmails maps the portable recipient model to Resend's bare-address arrays.
func recipientEmails(recipients []mail.Recipient) []string {
	if len(recipients) == 0 {
		return nil
	}
	out := make([]string, 0, len(recipients))
	for _, recipient := range recipients {
		out = append(out, strings.TrimSpace(recipient.Email))
	}
	return out
}

// formatRecipient keeps Resend's sender field compatible with optional display names.
func formatRecipient(recipient mail.Recipient) string {
	address := strings.TrimSpace(recipient.Email)
	name := strings.TrimSpace(recipient.Name)
	if name == "" {
		return address
	}
	return (&stdmail.Address{Name: name, Address: address}).String()
}

// copyHeaders detaches caller-owned header maps before provider-specific filtering.
func copyHeaders(headers map[string]string) map[string]string {
	if len(headers) == 0 {
		return nil
	}
	out := make(map[string]string, len(headers))
	for key, value := range headers {
		out[key] = value
	}
	return out
}

// deleteHeaderFold removes a header using the case-insensitive identity defined by mail protocols.
func deleteHeaderFold(headers map[string]string, name string) {
	for key := range headers {
		if strings.EqualFold(strings.TrimSpace(key), name) {
			delete(headers, key)
		}
	}
}

// idempotencyKey extracts Resend's transport header without forwarding it as message content.
func idempotencyKey(headers map[string]string) (string, bool) {
	for key, value := range headers {
		if strings.EqualFold(strings.TrimSpace(key), "Idempotency-Key") {
			trimmed := strings.TrimSpace(value)
			if trimmed != "" {
				return trimmed, true
			}
		}
	}
	return "", false
}

// buildTags produces deterministic provider tags so equivalent messages serialize identically.
func buildTags(tags []string, metadata map[string]string) []tag {
	out := make([]tag, 0, len(tags)+len(metadata))
	usedNames := make(map[string]struct{}, len(tags)+len(metadata))
	keys := make([]string, 0, len(metadata))
	for key := range metadata {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := metadata[key]
		name := sanitizeTagToken(key, 256)
		tagValue := sanitizeTagToken(value, 256)
		if name == "" || tagValue == "" {
			continue
		}
		if _, exists := usedNames[name]; exists {
			continue
		}
		usedNames[name] = struct{}{}
		out = append(out, tag{Name: name, Value: tagValue})
	}
	nextTagIndex := 1
	for _, value := range tags {
		tagValue := sanitizeTagToken(value, 256)
		if tagValue == "" {
			continue
		}
		name := ""
		for {
			name = "tag_" + strconv.Itoa(nextTagIndex)
			nextTagIndex++
			if _, exists := usedNames[name]; !exists {
				break
			}
		}
		usedNames[name] = struct{}{}
		out = append(out, tag{Name: name, Value: tagValue})
	}
	return out
}

// buildAttachments maps portable attachments to Resend's base64 JSON representation.
func buildAttachments(values []mail.Attachment) []attachment {
	if len(values) == 0 {
		return nil
	}
	out := make([]attachment, 0, len(values))
	for _, value := range values {
		out = append(out, attachment{
			Filename:    value.Filename,
			Content:     base64.StdEncoding.EncodeToString(value.Data),
			ContentType: value.ContentType,
		})
	}
	return out
}

// sanitizeTagToken limits tags to the portable token subset accepted by Resend.
func sanitizeTagToken(value string, max int) string {
	if max <= 0 {
		return ""
	}
	var builder strings.Builder
	builder.Grow(len(value))
	for _, r := range strings.TrimSpace(value) {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			builder.WriteRune(r)
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		case r == '_' || r == '-':
			builder.WriteRune(r)
		default:
			builder.WriteRune('_')
		}
		if builder.Len() >= max {
			break
		}
	}
	return strings.Trim(builder.String(), "_-")
}
