package mailpostmark

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	stdmail "net/mail"
	"sort"
	"strings"

	"github.com/goforj/mail"
	"github.com/goforj/mail/internal/mailhttp"
)

const defaultEndpoint = "https://api.postmarkapp.com/email"

// Config configures Postmark delivery.
// @group Postmark
type Config struct {
	ServerToken   string
	Endpoint      string
	MessageStream string
	HTTPClient    *http.Client
}

// Driver sends messages through the Postmark Email API.
// @group Postmark
type Driver struct {
	serverToken   string
	endpoint      string
	messageStream string
	client        *http.Client
}

type sendRequest struct {
	From          string            `json:"From"`
	To            string            `json:"To"`
	Cc            string            `json:"Cc,omitempty"`
	Bcc           string            `json:"Bcc,omitempty"`
	ReplyTo       string            `json:"ReplyTo,omitempty"`
	Subject       string            `json:"Subject"`
	HTMLBody      string            `json:"HtmlBody,omitempty"`
	TextBody      string            `json:"TextBody,omitempty"`
	Headers       []header          `json:"Headers,omitempty"`
	Attachments   []attachment      `json:"Attachments,omitempty"`
	Tag           string            `json:"Tag,omitempty"`
	Metadata      map[string]string `json:"Metadata,omitempty"`
	MessageStream string            `json:"MessageStream,omitempty"`
}

type header struct {
	Name  string `json:"Name"`
	Value string `json:"Value"`
}

type attachment struct {
	Name        string `json:"Name"`
	Content     string `json:"Content"`
	ContentType string `json:"ContentType"`
}

type sendResponse struct {
	MessageID string `json:"MessageID"`
	ErrorCode int    `json:"ErrorCode"`
	Message   string `json:"Message"`
}

// ResponseError describes a non-success response returned by Postmark.
// @group Postmark
type ResponseError struct {
	StatusCode int
	RequestID  string
	ErrorCode  int
}

// Error formats provider codes and a safe correlation identifier without including response content.
// @group Postmark
func (e *ResponseError) Error() string {
	if e.ErrorCode != 0 {
		if e.RequestID == "" {
			return fmt.Sprintf("mailpostmark: send failed with status %d and provider code %d", e.StatusCode, e.ErrorCode)
		}
		return fmt.Sprintf("mailpostmark: send failed with status %d and provider code %d (request %s)", e.StatusCode, e.ErrorCode, e.RequestID)
	}
	if e.RequestID == "" {
		return fmt.Sprintf("mailpostmark: send failed with status %d", e.StatusCode)
	}
	return fmt.Sprintf("mailpostmark: send failed with status %d (request %s)", e.StatusCode, e.RequestID)
}

// New creates a Postmark mail driver from the given config.
// @group Postmark
//
// Example: postmark
//
//	driver, _ := mailpostmark.New(mailpostmark.Config{
//		ServerToken: "pm_test_token",
//	})
//	fmt.Println(driver != nil)
//	// true
func New(config Config) (*Driver, error) {
	serverToken := strings.TrimSpace(config.ServerToken)
	if serverToken == "" {
		return nil, fmt.Errorf("mailpostmark: server token is required")
	}
	endpoint, err := mailhttp.Endpoint("mailpostmark", config.Endpoint, defaultEndpoint)
	if err != nil {
		return nil, err
	}
	client := config.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	return &Driver{
		serverToken:   serverToken,
		endpoint:      endpoint,
		messageStream: strings.TrimSpace(config.MessageStream),
		client:        client,
	}, nil
}

// Send validates and transmits one message through Postmark.
// @group Postmark
//
// Example: send
//
//	driver, _ := mailpostmark.New(mailpostmark.Config{
//		ServerToken: "pm_test_token",
//		Endpoint:    "http://127.0.0.1:1",
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
		From:     formatRecipient(*message.From),
		To:       strings.Join(recipientStrings(message.To), ","),
		Cc:       strings.Join(recipientStrings(message.Cc), ","),
		Bcc:      strings.Join(recipientStrings(message.Bcc), ","),
		ReplyTo:  strings.Join(recipientStrings(message.ReplyTo), ","),
		Subject:  strings.TrimSpace(message.Subject),
		HTMLBody: message.HTML,
		TextBody: message.Text,
		Metadata: copyStringMap(message.Metadata),
	}
	if d.messageStream != "" {
		payload.MessageStream = d.messageStream
	}
	if len(message.Tags) > 0 {
		payload.Tag = strings.TrimSpace(message.Tags[0])
		if payload.Metadata == nil {
			payload.Metadata = map[string]string{}
		}
		nextTagIndex := 2
		for _, tag := range message.Tags[1:] {
			key := ""
			for {
				key = fmt.Sprintf("tag_%d", nextTagIndex)
				nextTagIndex++
				if _, exists := payload.Metadata[key]; !exists {
					break
				}
			}
			payload.Metadata[key] = tag
		}
	}
	if headers := buildHeaders(message.Headers); len(headers) > 0 {
		payload.Headers = headers
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
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Postmark-Server-Token", d.serverToken)

	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, bodyErr := mailhttp.ReadResponseBody(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &ResponseError{
			StatusCode: resp.StatusCode,
			RequestID:  mailhttp.RequestID(resp.Header, "X-Postmark-Request-ID"),
		}
	}
	if bodyErr != nil {
		return bodyErr
	}

	if len(body) > 0 {
		var decoded sendResponse
		if err := json.Unmarshal(body, &decoded); err != nil {
			return err
		}
		if decoded.ErrorCode != 0 {
			return &ResponseError{
				StatusCode: resp.StatusCode,
				RequestID:  mailhttp.RequestID(resp.Header, "X-Postmark-Request-ID"),
				ErrorCode:  decoded.ErrorCode,
			}
		}
	}
	return nil
}

// recipientStrings maps recipients to Postmark's comma-delimited address fields.
func recipientStrings(recipients []mail.Recipient) []string {
	if len(recipients) == 0 {
		return nil
	}
	out := make([]string, 0, len(recipients))
	for _, recipient := range recipients {
		out = append(out, formatRecipient(recipient))
	}
	return out
}

// formatRecipient keeps Postmark address fields compatible with optional display names.
func formatRecipient(recipient mail.Recipient) string {
	address := strings.TrimSpace(recipient.Email)
	name := strings.TrimSpace(recipient.Name)
	if name == "" {
		return address
	}
	return (&stdmail.Address{Name: name, Address: address}).String()
}

// buildHeaders sorts map-backed headers so equivalent messages produce stable JSON arrays.
func buildHeaders(headers map[string]string) []header {
	if len(headers) == 0 {
		return nil
	}
	keys := make([]string, 0, len(headers))
	for key := range headers {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]header, 0, len(headers))
	for _, key := range keys {
		value := headers[key]
		if strings.TrimSpace(key) == "" {
			continue
		}
		out = append(out, header{Name: key, Value: value})
	}
	return out
}

// copyStringMap detaches metadata before Postmark-specific tag enrichment.
func copyStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

// buildAttachments maps portable attachments to Postmark's base64 JSON representation.
func buildAttachments(in []mail.Attachment) []attachment {
	if len(in) == 0 {
		return nil
	}
	out := make([]attachment, 0, len(in))
	for _, value := range in {
		out = append(out, attachment{
			Name:        value.Filename,
			Content:     base64.StdEncoding.EncodeToString(value.Data),
			ContentType: value.ContentType,
		})
	}
	return out
}
