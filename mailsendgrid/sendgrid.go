package mailsendgrid

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/goforj/mail"
)

const defaultEndpoint = "https://api.sendgrid.com/v3/mail/send"

// Config configures SendGrid delivery.
// @group SendGrid
type Config struct {
	APIKey     string
	Endpoint   string
	HTTPClient *http.Client
}

// Driver sends messages through the SendGrid Mail Send API.
// @group SendGrid
type Driver struct {
	apiKey   string
	endpoint string
	client   *http.Client
}

type sendRequest struct {
	From             sender            `json:"from"`
	ReplyTo          *sender           `json:"reply_to,omitempty"`
	Subject          string            `json:"subject"`
	Personalizations []personalization `json:"personalizations"`
	Content          []contentBlock    `json:"content"`
	Attachments      []attachment      `json:"attachments,omitempty"`
	Headers          map[string]string `json:"headers,omitempty"`
	Categories       []string          `json:"categories,omitempty"`
	CustomArgs       map[string]string `json:"custom_args,omitempty"`
}

type sender struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

type personalization struct {
	To         []sender          `json:"to"`
	Cc         []sender          `json:"cc,omitempty"`
	Bcc        []sender          `json:"bcc,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
	CustomArgs map[string]string `json:"custom_args,omitempty"`
}

type contentBlock struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type attachment struct {
	Content     string `json:"content"`
	Type        string `json:"type,omitempty"`
	Filename    string `json:"filename"`
	Disposition string `json:"disposition,omitempty"`
}

type apiError struct {
	StatusCode int
	Body       string
}

func (e *apiError) Error() string {
	if strings.TrimSpace(e.Body) == "" {
		return fmt.Sprintf("mailsendgrid: send failed with status %d", e.StatusCode)
	}
	return fmt.Sprintf("mailsendgrid: send failed with status %d: %s", e.StatusCode, e.Body)
}

// New creates a SendGrid mail driver from the given config.
// @group SendGrid
//
// Example: configure a SendGrid mail driver
//
//	driver, _ := mailsendgrid.New(mailsendgrid.Config{
//		APIKey: "SG.test_key",
//	})
//	fmt.Println(driver != nil)
//	// true
func New(config Config) (*Driver, error) {
	apiKey := strings.TrimSpace(config.APIKey)
	if apiKey == "" {
		return nil, fmt.Errorf("mailsendgrid: api key is required")
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

// Send validates and transmits one message through SendGrid.
// @group SendGrid
//
// Example: send one message through SendGrid
//
//	driver, _ := mailsendgrid.New(mailsendgrid.Config{
//		APIKey:   "SG.test_key",
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
		return fmt.Errorf("mailsendgrid: from is required")
	}

	contents := buildContent(message)
	payload := sendRequest{
		From:             toSender(*message.From),
		Subject:          strings.TrimSpace(message.Subject),
		Personalizations: []personalization{buildPersonalization(message)},
		Content:          contents,
	}
	if len(message.ReplyTo) > 0 {
		replyTo := message.ReplyTo[0]
		payload.ReplyTo = &sender{
			Email: strings.TrimSpace(replyTo.Email),
			Name:  strings.TrimSpace(replyTo.Name),
		}
	}
	if headers := copyStringMap(message.Headers); len(headers) > 0 {
		payload.Headers = headers
	}
	if len(message.Tags) > 0 {
		payload.Categories = append([]string(nil), message.Tags...)
	}
	if metadata := copyStringMap(message.Metadata); len(metadata) > 0 {
		payload.CustomArgs = metadata
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
	return nil
}

func buildPersonalization(message mail.Message) personalization {
	p := personalization{
		To:         recipientsToSenders(message.To),
		Cc:         recipientsToSenders(message.Cc),
		Bcc:        recipientsToSenders(message.Bcc),
		Headers:    copyStringMap(message.Headers),
		CustomArgs: copyStringMap(message.Metadata),
	}
	if len(p.Headers) == 0 {
		p.Headers = nil
	}
	if len(p.CustomArgs) == 0 {
		p.CustomArgs = nil
	}
	return p
}

func buildContent(message mail.Message) []contentBlock {
	out := make([]contentBlock, 0, 2)
	if text := strings.TrimSpace(message.Text); text != "" {
		out = append(out, contentBlock{
			Type:  "text/plain",
			Value: message.Text,
		})
	}
	if html := strings.TrimSpace(message.HTML); html != "" {
		out = append(out, contentBlock{
			Type:  "text/html",
			Value: message.HTML,
		})
	}
	return out
}

func recipientsToSenders(recipients []mail.Recipient) []sender {
	if len(recipients) == 0 {
		return nil
	}
	out := make([]sender, 0, len(recipients))
	for _, recipient := range recipients {
		out = append(out, toSender(recipient))
	}
	return out
}

func toSender(recipient mail.Recipient) sender {
	return sender{
		Email: strings.TrimSpace(recipient.Email),
		Name:  strings.TrimSpace(recipient.Name),
	}
}

func copyStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]string, len(values))
	for key, value := range values {
		out[key] = value
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
			Content:     base64.StdEncoding.EncodeToString(value.Data),
			Type:        value.ContentType,
			Filename:    value.Filename,
			Disposition: "attachment",
		})
	}
	return out
}
