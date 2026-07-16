package mailmailgun

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime"
	"mime/multipart"
	"net/http"
	stdmail "net/mail"
	"net/textproto"
	"net/url"
	"strings"

	"github.com/goforj/mail"
	"github.com/goforj/mail/internal/mailhttp"
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

// ResponseError describes a non-success response returned by Mailgun.
// @group Mailgun
type ResponseError struct {
	StatusCode int
	RequestID  string
}

// Error formats the provider status and safe correlation identifier without including response content.
// @group Mailgun
func (e *ResponseError) Error() string {
	if e.RequestID == "" {
		return fmt.Sprintf("mailmailgun: send failed with status %d", e.StatusCode)
	}
	return fmt.Sprintf("mailmailgun: send failed with status %d (request %s)", e.StatusCode, e.RequestID)
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
	endpoint, err := mailhttp.Endpoint("mailmailgun", strings.TrimRight(strings.TrimSpace(config.Endpoint), "/"), defaultEndpoint)
	if err != nil {
		return nil, err
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
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := message.Validate(); err != nil {
		return err
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
		partHeader := textproto.MIMEHeader{}
		partHeader.Set("Content-Disposition", mime.FormatMediaType("form-data", map[string]string{
			"filename": attachment.Filename,
			"name":     "attachment",
		}))
		partHeader.Set("Content-Type", attachment.ContentType)
		part, err := writer.CreatePart(partHeader)
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

	requestURL := d.endpoint + "/v3/" + url.PathEscape(d.domain) + "/messages"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, &body)
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

	respBody, bodyErr := mailhttp.ReadResponseBody(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &ResponseError{
			StatusCode: resp.StatusCode,
			RequestID:  mailhttp.RequestID(resp.Header, "X-Mailgun-Request-ID"),
		}
	}
	if bodyErr != nil {
		return bodyErr
	}

	if len(respBody) > 0 {
		var decoded sendResponse
		if err := json.Unmarshal(respBody, &decoded); err != nil {
			return err
		}
	}
	return nil
}

// formatRecipient keeps Mailgun form fields compatible with optional display names.
func formatRecipient(recipient mail.Recipient) string {
	address := strings.TrimSpace(recipient.Email)
	name := strings.TrimSpace(recipient.Name)
	if name == "" {
		return address
	}
	return (&stdmail.Address{Name: name, Address: address}).String()
}
