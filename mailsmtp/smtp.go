package mailsmtp

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	stdmail "net/mail"
	"net/smtp"
	"net/textproto"
	"strconv"
	"strings"

	"github.com/goforj/mail"
)

// Config configures SMTP delivery.
// @group SMTP
type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	Identity string
	ForceTLS bool
}

// Driver sends messages over SMTP.
// @group SMTP
type Driver struct {
	host     string
	port     int
	username string
	password string
	identity string
	forceTLS bool
}

// New creates an SMTP mail driver from the given config.
// @group SMTP
//
// Example: configure an SMTP mail driver
//
//	driver, _ := mailsmtp.New(mailsmtp.Config{
//		Host: "smtp.example.com",
//		Port: 587,
//	})
//	fmt.Println(driver != nil)
//	// true
//
// Example: configure Gmail SMTP with an app password
//
//	driver, _ := mailsmtp.New(mailsmtp.Config{
//		Host:     "smtp.gmail.com",
//		Port:     587,
//		Username: "you@gmail.com",
//		Password: "gmail-app-password",
//	})
//	fmt.Println(driver != nil)
//	// true
func New(config Config) (*Driver, error) {
	host := strings.TrimSpace(config.Host)
	if host == "" {
		return nil, fmt.Errorf("mailsmtp: host is required")
	}
	port := config.Port
	if port <= 0 {
		port = 25
	}
	return &Driver{
		host:     host,
		port:     port,
		username: strings.TrimSpace(config.Username),
		password: config.Password,
		identity: strings.TrimSpace(config.Identity),
		forceTLS: config.ForceTLS,
	}, nil
}

// Send validates and transmits one message over SMTP.
// @group SMTP
//
// Example: send one message over SMTP
//
//	driver, _ := mailsmtp.New(mailsmtp.Config{
//		Host: "smtp.example.com",
//		Port: 587,
//	})
//	err := driver.Send(context.Background(), mail.Message{
//		From:    &mail.Recipient{Email: "no-reply@example.com"},
//		To:      []mail.Recipient{{Email: "alice@example.com"}},
//		Subject: "Welcome",
//		Text:    "hello world",
//	})
//	fmt.Println(err == nil)
//	// false
func (m *Driver) Send(ctx context.Context, message mail.Message) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := message.Validate(); err != nil {
		return err
	}
	raw, err := Render(message)
	if err != nil {
		return err
	}
	recipients := collectRecipients(message)
	addr := net.JoinHostPort(m.host, strconv.Itoa(m.port))
	if m.forceTLS {
		return m.sendTLS(ctx, addr, message.From.Email, recipients, raw)
	}
	auth := m.auth()
	return smtp.SendMail(addr, auth, message.From.Email, recipients, raw)
}

func (m *Driver) sendTLS(ctx context.Context, addr, from string, recipients []string, raw []byte) error {
	dialer := &tls.Dialer{Config: &tls.Config{ServerName: m.host, MinVersion: tls.VersionTLS12}}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, m.host)
	if err != nil {
		return err
	}
	defer client.Close()

	if m.username != "" || m.password != "" {
		if err := client.Auth(m.auth()); err != nil {
			return err
		}
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	for _, recipient := range recipients {
		if err := client.Rcpt(recipient); err != nil {
			return err
		}
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write(raw); err != nil {
		_ = writer.Close()
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return client.Quit()
}

func (m *Driver) auth() smtp.Auth {
	if m.username == "" && m.password == "" {
		return nil
	}
	return smtp.PlainAuth(m.identity, m.username, m.password, m.host)
}

// Render turns one message into an RFC 822 style SMTP payload.
// @group SMTP
//
// Example: render a text message
//
//	raw, _ := mailsmtp.Render(mail.Message{
//		From:    &mail.Recipient{Email: "no-reply@example.com", Name: "Example"},
//		To:      []mail.Recipient{{Email: "alice@example.com", Name: "Alice"}},
//		Subject: "Welcome",
//		Text:    "hello world",
//	})
//	fmt.Println(strings.Contains(string(raw), "Subject: Welcome"))
//	// true
func Render(message mail.Message) ([]byte, error) {
	if err := message.Validate(); err != nil {
		return nil, err
	}

	var buffer bytes.Buffer
	headers := textproto.MIMEHeader{}
	headers.Set("From", formatRecipients([]mail.Recipient{*message.From}))
	if len(message.To) > 0 {
		headers.Set("To", formatRecipients(message.To))
	}
	if len(message.Cc) > 0 {
		headers.Set("Cc", formatRecipients(message.Cc))
	}
	if len(message.ReplyTo) > 0 {
		headers.Set("Reply-To", formatRecipients(message.ReplyTo))
	}
	headers.Set("Subject", strings.TrimSpace(message.Subject))
	headers.Set("MIME-Version", "1.0")
	for key, value := range message.Headers {
		headers.Set(key, value)
	}

	for key, values := range headers {
		for _, value := range values {
			buffer.WriteString(key)
			buffer.WriteString(": ")
			buffer.WriteString(value)
			buffer.WriteString("\r\n")
		}
	}

	body, contentType, err := renderBody(message)
	if err != nil {
		return nil, err
	}
	buffer.WriteString("Content-Type: ")
	buffer.WriteString(contentType)
	buffer.WriteString("\r\n\r\n")
	buffer.Write(body)
	return buffer.Bytes(), nil
}

func renderBody(message mail.Message) ([]byte, string, error) {
	html := strings.TrimSpace(message.HTML)
	text := strings.TrimSpace(message.Text)
	if len(message.Attachments) == 0 {
		if html != "" && text != "" {
			return renderMultipartAlternative(text, html)
		}
		contentType := `text/plain; charset="utf-8"`
		body := text
		if html != "" {
			contentType = `text/html; charset="utf-8"`
			body = html
		}
		return []byte(body), contentType, nil
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	inlineBody, inlineType, err := renderInlineBody(text, html)
	if err != nil {
		return nil, "", err
	}
	messageHeader := textproto.MIMEHeader{}
	messageHeader.Set("Content-Type", inlineType)
	messagePart, err := writer.CreatePart(messageHeader)
	if err != nil {
		return nil, "", err
	}
	if _, err := messagePart.Write(inlineBody); err != nil {
		return nil, "", err
	}

	for _, attachment := range message.Attachments {
		partHeader := textproto.MIMEHeader{}
		partHeader.Set("Content-Type", attachment.ContentType)
		partHeader.Set("Content-Disposition", `attachment; filename="`+escapeHeaderToken(attachment.Filename)+`"`)
		partHeader.Set("Content-Transfer-Encoding", "base64")
		part, err := writer.CreatePart(partHeader)
		if err != nil {
			return nil, "", err
		}
		lineWriter := newBase64LineWriter(part)
		encoder := base64.NewEncoder(base64.StdEncoding, lineWriter)
		if _, err := encoder.Write(attachment.Data); err != nil {
			_ = encoder.Close()
			return nil, "", err
		}
		if err := encoder.Close(); err != nil {
			return nil, "", err
		}
		if err := lineWriter.Close(); err != nil {
			return nil, "", err
		}
	}

	if err := writer.Close(); err != nil {
		return nil, "", err
	}
	return body.Bytes(), `multipart/mixed; boundary="` + writer.Boundary() + `"`, nil
}

func renderMultipartAlternative(textBody, htmlBody string) ([]byte, string, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	textHeader := textproto.MIMEHeader{}
	textHeader.Set("Content-Type", `text/plain; charset="utf-8"`)
	textHeader.Set("Content-Transfer-Encoding", "8bit")
	textPart, err := writer.CreatePart(textHeader)
	if err != nil {
		return nil, "", err
	}
	if _, err := textPart.Write([]byte(textBody)); err != nil {
		return nil, "", err
	}

	htmlHeader := textproto.MIMEHeader{}
	htmlHeader.Set("Content-Type", `text/html; charset="utf-8"`)
	htmlHeader.Set("Content-Transfer-Encoding", "8bit")
	htmlPart, err := writer.CreatePart(htmlHeader)
	if err != nil {
		return nil, "", err
	}
	if _, err := htmlPart.Write([]byte(htmlBody)); err != nil {
		return nil, "", err
	}

	if err := writer.Close(); err != nil {
		return nil, "", err
	}
	return body.Bytes(), `multipart/alternative; boundary="` + writer.Boundary() + `"`, nil
}

func renderInlineBody(textBody, htmlBody string) ([]byte, string, error) {
	if strings.TrimSpace(textBody) != "" && strings.TrimSpace(htmlBody) != "" {
		return renderMultipartAlternative(textBody, htmlBody)
	}
	if strings.TrimSpace(htmlBody) != "" {
		return []byte(htmlBody), `text/html; charset="utf-8"`, nil
	}
	return []byte(textBody), `text/plain; charset="utf-8"`, nil
}

func escapeHeaderToken(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `"`, `\"`)
	return replacer.Replace(value)
}

type base64LineWriter struct {
	writer bytes.Buffer
	target io.Writer
}

func newBase64LineWriter(target io.Writer) *base64LineWriter {
	return &base64LineWriter{target: target}
}

func (w *base64LineWriter) Write(p []byte) (int, error) {
	w.writer.Write(p)
	for w.writer.Len() >= 76 {
		chunk := w.writer.Next(76)
		if _, err := w.target.Write(chunk); err != nil {
			return 0, err
		}
		if _, err := w.target.Write([]byte("\r\n")); err != nil {
			return 0, err
		}
	}
	return len(p), nil
}

func (w *base64LineWriter) Close() error {
	if w.writer.Len() == 0 {
		return nil
	}
	if _, err := w.target.Write(w.writer.Bytes()); err != nil {
		return err
	}
	_, err := w.target.Write([]byte("\r\n"))
	return err
}

func collectRecipients(message mail.Message) []string {
	recipients := make([]string, 0, len(message.To)+len(message.Cc)+len(message.Bcc))
	for _, recipient := range message.To {
		recipients = append(recipients, recipient.Email)
	}
	for _, recipient := range message.Cc {
		recipients = append(recipients, recipient.Email)
	}
	for _, recipient := range message.Bcc {
		recipients = append(recipients, recipient.Email)
	}
	return recipients
}

func formatRecipients(recipients []mail.Recipient) string {
	formatted := make([]string, 0, len(recipients))
	for _, recipient := range recipients {
		formatted = append(formatted, formatRecipient(recipient))
	}
	return strings.Join(formatted, ", ")
}

func formatRecipient(recipient mail.Recipient) string {
	address := strings.TrimSpace(recipient.Email)
	name := strings.TrimSpace(recipient.Name)
	if name == "" {
		return address
	}
	return (&stdmail.Address{Name: name, Address: address}).String()
}
