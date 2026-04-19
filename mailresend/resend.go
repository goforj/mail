package mailresend

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	stdmail "net/mail"
	"strconv"
	"strings"

	"github.com/goforj/mail"
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

type apiError struct {
	StatusCode int
	Body       string
}

func (e *apiError) Error() string {
	if strings.TrimSpace(e.Body) == "" {
		return fmt.Sprintf("mailresend: send failed with status %d", e.StatusCode)
	}
	return fmt.Sprintf("mailresend: send failed with status %d: %s", e.StatusCode, e.Body)
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
	endpoint := strings.TrimSpace(config.Endpoint)
	if endpoint == "" {
		endpoint = defaultEndpoint
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
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := message.Validate(); err != nil {
		return err
	}
	if message.From == nil {
		return fmt.Errorf("mailresend: from is required")
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
		delete(headers, "Idempotency-Key")
		delete(headers, "idempotency-key")
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &apiError{
			StatusCode: resp.StatusCode,
			Body:       strings.TrimSpace(string(body)),
		}
	}

	var decoded sendResponse
	if len(body) > 0 {
		if err := json.Unmarshal(body, &decoded); err != nil {
			return err
		}
	}
	return nil
}

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

func formatRecipient(recipient mail.Recipient) string {
	address := strings.TrimSpace(recipient.Email)
	name := strings.TrimSpace(recipient.Name)
	if name == "" {
		return address
	}
	return (&stdmail.Address{Name: name, Address: address}).String()
}

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

func buildTags(tags []string, metadata map[string]string) []tag {
	out := make([]tag, 0, len(tags)+len(metadata))
	for key, value := range metadata {
		name := sanitizeTagToken(key, 256)
		tagValue := sanitizeTagToken(value, 256)
		if name == "" || tagValue == "" {
			continue
		}
		out = append(out, tag{Name: name, Value: tagValue})
	}
	for i, value := range tags {
		name := "tag_" + strconv.Itoa(i+1)
		tagValue := sanitizeTagToken(value, 256)
		if tagValue == "" {
			continue
		}
		out = append(out, tag{Name: name, Value: tagValue})
	}
	return out
}

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
