package mailsmtp

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	stdmail "net/mail"
	"net/smtp"
	"net/textproto"
	"sort"
	"strconv"
	"strings"

	"github.com/goforj/mail"
)

// Config configures SMTP delivery.
// @group SMTP
type Config struct {
	// Host is the SMTP server hostname and is also the default TLS ServerName.
	Host string
	// Port is the SMTP server port; zero defaults to 25.
	Port int
	// Username is the optional SMTP authentication username.
	Username string
	// Password is the optional SMTP authentication password.
	Password string
	// Identity is the optional PLAIN authentication identity.
	Identity string
	// ForceTLS selects implicit TLS; otherwise the driver upgrades with STARTTLS when advertised.
	ForceTLS bool
	// TLSConfig customizes TLS verification and is cloned during construction.
	TLSConfig *tls.Config
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
	tls      *tls.Config
}

// New creates an SMTP mail driver from the given config.
// TLS defaults to the configured host for ServerName, a minimum of TLS 1.2, and normal certificate verification.
// @group SMTP
//
// Example: smtp
//
//	driver, _ := mailsmtp.New(mailsmtp.Config{
//		Host: "smtp.example.com",
//		Port: 587,
//	})
//	fmt.Println(driver != nil)
//	// true
//
// Example: gmail
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
	if port == 0 {
		port = 25
	}
	if port < 1 || port > 65535 {
		return nil, fmt.Errorf("mailsmtp: port must be between 1 and 65535")
	}
	tlsConfig := config.TLSConfig
	if tlsConfig == nil {
		tlsConfig = &tls.Config{}
	} else {
		tlsConfig = tlsConfig.Clone()
	}
	if tlsConfig.ServerName == "" {
		tlsConfig.ServerName = host
	}
	if tlsConfig.MinVersion == 0 {
		tlsConfig.MinVersion = tls.VersionTLS12
	}
	return &Driver{
		host:     host,
		port:     port,
		username: strings.TrimSpace(config.Username),
		password: config.Password,
		identity: strings.TrimSpace(config.Identity),
		forceTLS: config.ForceTLS,
		tls:      tlsConfig,
	}, nil
}

// Send validates and transmits one message over SMTP.
// @group SMTP
//
// Example: send
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
	if ctx == nil {
		ctx = context.Background()
	}
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
	return m.send(ctx, addr, strings.TrimSpace(message.From.Email), recipients, raw)
}

// send owns the SMTP session so context cancellation applies to plain, STARTTLS, and implicit-TLS connections alike.
func (m *Driver) send(ctx context.Context, addr, from string, recipients []string, raw []byte) (sendErr error) {
	defer func() {
		if sendErr != nil && ctx.Err() != nil {
			sendErr = ctx.Err()
		}
	}()

	var conn net.Conn
	var err error
	if m.forceTLS {
		dialer := &tls.Dialer{Config: m.tls.Clone()}
		conn, err = dialer.DialContext(ctx, "tcp", addr)
	} else {
		dialer := &net.Dialer{}
		conn, err = dialer.DialContext(ctx, "tcp", addr)
	}
	if err != nil {
		return err
	}
	defer conn.Close()
	stopWatchingContext := context.AfterFunc(ctx, func() {
		_ = conn.Close()
	})
	defer stopWatchingContext()

	client, err := smtp.NewClient(conn, m.host)
	if err != nil {
		return err
	}
	defer client.Close()

	if !m.forceTLS {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(m.tls.Clone()); err != nil {
				return err
			}
		}
	}
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

// auth returns nil for anonymous SMTP and otherwise defers credential policy to net/smtp.
func (m *Driver) auth() smtp.Auth {
	if m.username == "" && m.password == "" {
		return nil
	}
	return smtp.PlainAuth(m.identity, m.username, m.password, m.host)
}

// Render turns one message into an RFC 822 style SMTP payload.
// @group SMTP
//
// Example: render
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

	body, err := renderBody(message)
	if err != nil {
		return nil, err
	}

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
	headers.Set("Subject", mime.QEncoding.Encode("utf-8", strings.TrimSpace(message.Subject)))
	headers.Set("MIME-Version", "1.0")
	for key, value := range message.Headers {
		headers.Set(key, value)
	}
	headers.Set("Content-Type", body.contentType)
	if body.transferEncoding != "" {
		headers.Set("Content-Transfer-Encoding", body.transferEncoding)
	}

	keys := make([]string, 0, len(headers))
	for key := range headers {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var buffer bytes.Buffer
	for _, key := range keys {
		values := headers[key]
		for _, value := range values {
			buffer.WriteString(key)
			buffer.WriteString(": ")
			buffer.WriteString(value)
			buffer.WriteString("\r\n")
		}
	}
	buffer.WriteString("\r\n")
	buffer.Write(body.data)
	return buffer.Bytes(), nil
}

// renderedPart carries the MIME metadata needed by both top-level and nested parts.
type renderedPart struct {
	data             []byte
	contentType      string
	transferEncoding string
}

// renderBody selects the smallest MIME tree that preserves the message's body variants and attachments.
func renderBody(message mail.Message) (renderedPart, error) {
	hasHTML := strings.TrimSpace(message.HTML) != ""
	hasText := strings.TrimSpace(message.Text) != ""
	if len(message.Attachments) == 0 {
		if hasHTML && hasText {
			return renderMultipartAlternative(message.Text, message.HTML)
		}
		contentType := `text/plain; charset="utf-8"`
		body := message.Text
		if hasHTML {
			contentType = `text/html; charset="utf-8"`
			body = message.HTML
		}
		return renderTextPart(contentType, body)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	inlinePart, err := renderInlineBody(message.Text, message.HTML)
	if err != nil {
		return renderedPart{}, err
	}
	if err := writeMIMEPart(writer, inlinePart); err != nil {
		return renderedPart{}, err
	}

	for _, attachment := range message.Attachments {
		mediaType, params, _ := mime.ParseMediaType(attachment.ContentType)
		partHeader := textproto.MIMEHeader{}
		partHeader.Set("Content-Type", mime.FormatMediaType(mediaType, params))
		partHeader.Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": attachment.Filename}))
		partHeader.Set("Content-Transfer-Encoding", "base64")
		part, err := writer.CreatePart(partHeader)
		if err != nil {
			return renderedPart{}, err
		}
		lineWriter := newBase64LineWriter(part)
		encoder := base64.NewEncoder(base64.StdEncoding, lineWriter)
		if _, err := encoder.Write(attachment.Data); err != nil {
			_ = encoder.Close()
			return renderedPart{}, err
		}
		if err := encoder.Close(); err != nil {
			return renderedPart{}, err
		}
		if err := lineWriter.Close(); err != nil {
			return renderedPart{}, err
		}
	}

	if err := writer.Close(); err != nil {
		return renderedPart{}, err
	}
	return renderedPart{
		data:        body.Bytes(),
		contentType: mime.FormatMediaType("multipart/mixed", map[string]string{"boundary": writer.Boundary()}),
	}, nil
}

// renderMultipartAlternative gives clients both portable representations without preferring one during transport.
func renderMultipartAlternative(textBody, htmlBody string) (renderedPart, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	textPart, err := renderTextPart(`text/plain; charset="utf-8"`, textBody)
	if err != nil {
		return renderedPart{}, err
	}
	if err := writeMIMEPart(writer, textPart); err != nil {
		return renderedPart{}, err
	}

	htmlPart, err := renderTextPart(`text/html; charset="utf-8"`, htmlBody)
	if err != nil {
		return renderedPart{}, err
	}
	if err := writeMIMEPart(writer, htmlPart); err != nil {
		return renderedPart{}, err
	}

	if err := writer.Close(); err != nil {
		return renderedPart{}, err
	}
	return renderedPart{
		data:        body.Bytes(),
		contentType: mime.FormatMediaType("multipart/alternative", map[string]string{"boundary": writer.Boundary()}),
	}, nil
}

// renderInlineBody prepares the body node nested inside a multipart/mixed message.
func renderInlineBody(textBody, htmlBody string) (renderedPart, error) {
	if strings.TrimSpace(textBody) != "" && strings.TrimSpace(htmlBody) != "" {
		return renderMultipartAlternative(textBody, htmlBody)
	}
	if strings.TrimSpace(htmlBody) != "" {
		return renderTextPart(`text/html; charset="utf-8"`, htmlBody)
	}
	return renderTextPart(`text/plain; charset="utf-8"`, textBody)
}

// renderTextPart uses quoted-printable so SMTP transport does not depend on an 8BITMIME extension.
func renderTextPart(contentType, body string) (renderedPart, error) {
	var encoded bytes.Buffer
	writer := quotedprintable.NewWriter(&encoded)
	if _, err := writer.Write([]byte(body)); err != nil {
		return renderedPart{}, err
	}
	if err := writer.Close(); err != nil {
		return renderedPart{}, err
	}
	return renderedPart{
		data:             encoded.Bytes(),
		contentType:      contentType,
		transferEncoding: "quoted-printable",
	}, nil
}

// writeMIMEPart emits metadata before data so every nested part follows the same encoding contract.
func writeMIMEPart(writer *multipart.Writer, rendered renderedPart) error {
	header := textproto.MIMEHeader{}
	header.Set("Content-Type", rendered.contentType)
	if rendered.transferEncoding != "" {
		header.Set("Content-Transfer-Encoding", rendered.transferEncoding)
	}
	part, err := writer.CreatePart(header)
	if err != nil {
		return err
	}
	_, err = part.Write(rendered.data)
	return err
}

type base64LineWriter struct {
	writer bytes.Buffer
	target io.Writer
}

// newBase64LineWriter wraps attachment output at the 76-character MIME line limit.
func newBase64LineWriter(target io.Writer) *base64LineWriter {
	return &base64LineWriter{target: target}
}

// Write buffers encoded data until complete MIME-width lines can be emitted.
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

// Close flushes the final partial MIME line with a CRLF terminator.
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

// collectRecipients returns transport-envelope addresses without leaking Bcc into rendered headers.
func collectRecipients(message mail.Message) []string {
	recipients := make([]string, 0, len(message.To)+len(message.Cc)+len(message.Bcc))
	for _, recipient := range message.To {
		recipients = append(recipients, strings.TrimSpace(recipient.Email))
	}
	for _, recipient := range message.Cc {
		recipients = append(recipients, strings.TrimSpace(recipient.Email))
	}
	for _, recipient := range message.Bcc {
		recipients = append(recipients, strings.TrimSpace(recipient.Email))
	}
	return recipients
}

// formatRecipients joins independently encoded mailbox values for one address header.
func formatRecipients(recipients []mail.Recipient) string {
	formatted := make([]string, 0, len(recipients))
	for _, recipient := range recipients {
		formatted = append(formatted, formatRecipient(recipient))
	}
	return strings.Join(formatted, ", ")
}

// formatRecipient delegates display-name quoting and encoded-word handling to the standard library.
func formatRecipient(recipient mail.Recipient) string {
	address := strings.TrimSpace(recipient.Email)
	name := strings.TrimSpace(recipient.Name)
	if name == "" {
		return address
	}
	return (&stdmail.Address{Name: name, Address: address}).String()
}
