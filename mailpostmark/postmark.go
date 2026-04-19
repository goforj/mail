package mailpostmark

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	stdmail "net/mail"
	"strings"

	"github.com/goforj/mail"
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
	Tag           string            `json:"Tag,omitempty"`
	Metadata      map[string]string `json:"Metadata,omitempty"`
	MessageStream string            `json:"MessageStream,omitempty"`
}

type header struct {
	Name  string `json:"Name"`
	Value string `json:"Value"`
}

type sendResponse struct {
	MessageID string `json:"MessageID"`
	ErrorCode int    `json:"ErrorCode"`
	Message   string `json:"Message"`
}

type apiError struct {
	StatusCode int
	Body       string
}

func (e *apiError) Error() string {
	if strings.TrimSpace(e.Body) == "" {
		return fmt.Sprintf("mailpostmark: send failed with status %d", e.StatusCode)
	}
	return fmt.Sprintf("mailpostmark: send failed with status %d: %s", e.StatusCode, e.Body)
}

// New creates a Postmark mail driver from the given config.
// @group Postmark
//
// Example: configure a Postmark mail driver
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
	endpoint := strings.TrimSpace(config.Endpoint)
	if endpoint == "" {
		endpoint = defaultEndpoint
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
// Example: send one message through Postmark
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
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := message.Validate(); err != nil {
		return err
	}
	if message.From == nil {
		return fmt.Errorf("mailpostmark: from is required")
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
		for i, tag := range message.Tags[1:] {
			payload.Metadata[fmt.Sprintf("tag_%d", i+2)] = tag
		}
	}
	if headers := buildHeaders(message.Headers); len(headers) > 0 {
		payload.Headers = headers
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &apiError{StatusCode: resp.StatusCode, Body: strings.TrimSpace(string(body))}
	}

	if len(body) > 0 {
		var decoded sendResponse
		if err := json.Unmarshal(body, &decoded); err != nil {
			return err
		}
	}
	return nil
}

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

func formatRecipient(recipient mail.Recipient) string {
	address := strings.TrimSpace(recipient.Email)
	name := strings.TrimSpace(recipient.Name)
	if name == "" {
		return address
	}
	return (&stdmail.Address{Name: name, Address: address}).String()
}

func buildHeaders(headers map[string]string) []header {
	if len(headers) == 0 {
		return nil
	}
	out := make([]header, 0, len(headers))
	for key, value := range headers {
		if strings.TrimSpace(key) == "" {
			continue
		}
		out = append(out, header{Name: key, Value: value})
	}
	return out
}

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
