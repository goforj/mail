package mailmailgun

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	stdmail "net/mail"
	"strings"

	"github.com/goforj/mail"
)

const defaultEndpoint = "https://api.mailgun.net"

// Config configures Mailgun delivery.
// @group Mailgun
type Config struct {
	Domain     string
	APIKey     string
	Endpoint   string
	HTTPClient *http.Client
}

// Driver sends messages through the Mailgun Messages API.
// @group Mailgun
type Driver struct {
	domain   string
	apiKey   string
	endpoint string
	client   *http.Client
}

type sendResponse struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

type apiError struct {
	StatusCode int
	Body       string
}

func (e *apiError) Error() string {
	if strings.TrimSpace(e.Body) == "" {
		return fmt.Sprintf("mailmailgun: send failed with status %d", e.StatusCode)
	}
	return fmt.Sprintf("mailmailgun: send failed with status %d: %s", e.StatusCode, e.Body)
}

// New creates a Mailgun mail driver from the given config.
// @group Mailgun
//
// Example: mailgun
//
//	driver, _ := mailmailgun.New(mailmailgun.Config{
//		Domain: "mg.example.com",
//		APIKey: "key-test",
//	})
//	fmt.Println(driver != nil)
//	// true
func New(config Config) (*Driver, error) {
	domain := strings.TrimSpace(config.Domain)
	if domain == "" {
		return nil, fmt.Errorf("mailmailgun: domain is required")
	}
	apiKey := strings.TrimSpace(config.APIKey)
	if apiKey == "" {
		return nil, fmt.Errorf("mailmailgun: api key is required")
	}
	endpoint := strings.TrimRight(strings.TrimSpace(config.Endpoint), "/")
	if endpoint == "" {
		endpoint = defaultEndpoint
	}
	client := config.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	return &Driver{
		domain:   domain,
		apiKey:   apiKey,
		endpoint: endpoint,
		client:   client,
	}, nil
}

// Send validates and transmits one message through Mailgun.
// @group Mailgun
//
// Example: send
//
//	driver, _ := mailmailgun.New(mailmailgun.Config{
//		Domain:   "mg.example.com",
//		APIKey:   "key-test",
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
		return fmt.Errorf("mailmailgun: from is required")
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	writeField := func(key, value string) error {
		if strings.TrimSpace(value) == "" {
			return nil
		}
		return writer.WriteField(key, value)
	}
	writeRecipients := func(key string, recipients []mail.Recipient) error {
		for _, recipient := range recipients {
			if err := writer.WriteField(key, formatRecipient(recipient)); err != nil {
				return err
			}
		}
		return nil
	}

	if err := writeField("from", formatRecipient(*message.From)); err != nil {
		return err
	}
	if err := writeRecipients("to", message.To); err != nil {
		return err
	}
	if err := writeRecipients("cc", message.Cc); err != nil {
		return err
	}
	if err := writeRecipients("bcc", message.Bcc); err != nil {
		return err
	}
	if err := writeField("subject", strings.TrimSpace(message.Subject)); err != nil {
		return err
	}
	if err := writeField("text", message.Text); err != nil {
		return err
	}
	if err := writeField("html", message.HTML); err != nil {
		return err
	}
	if len(message.ReplyTo) > 0 {
		replyTo := make([]string, 0, len(message.ReplyTo))
		for _, recipient := range message.ReplyTo {
			replyTo = append(replyTo, formatRecipient(recipient))
		}
		if err := writeField("h:Reply-To", strings.Join(replyTo, ",")); err != nil {
			return err
		}
	}
	for key, value := range message.Headers {
		if err := writeField("h:"+key, value); err != nil {
			return err
		}
	}
	for _, value := range message.Tags {
		if err := writeField("o:tag", value); err != nil {
			return err
		}
	}
	for _, attachment := range message.Attachments {
		part, err := writer.CreateFormFile("attachment", attachment.Filename)
		if err != nil {
			return err
		}
		if _, err := part.Write(attachment.Data); err != nil {
			return err
		}
	}
	for key, value := range message.Metadata {
		if err := writeField("v:"+key, value); err != nil {
			return err
		}
	}
	if err := writer.Close(); err != nil {
		return err
	}

	url := d.endpoint + "/v3/" + d.domain + "/messages"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &body)
	if err != nil {
		return err
	}
	req.SetBasicAuth("api", d.apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &apiError{StatusCode: resp.StatusCode, Body: strings.TrimSpace(string(respBody))}
	}

	if len(respBody) > 0 {
		var decoded sendResponse
		if err := json.Unmarshal(respBody, &decoded); err != nil {
			return err
		}
	}
	return nil
}

func formatRecipient(recipient mail.Recipient) string {
	address := strings.TrimSpace(recipient.Email)
	name := strings.TrimSpace(recipient.Name)
	if name == "" {
		return address
	}
	return (&stdmail.Address{Name: name, Address: address}).String()
}
