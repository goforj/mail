package mailsmtp_test

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/goforj/mail"
	"github.com/goforj/mail/mailsmtp"
)

type smtpCapture struct {
	mu         sync.Mutex
	mailFrom   string
	recipients []string
	data       string
	failAuth   bool
	failMail   bool
	failRCPT   bool
	failData   bool
}

var (
	sharedTLSOnce   sync.Once
	sharedTLSConfig *tls.Config
	sharedTLSCAPEM  []byte
)

// setMailFrom serializes writes from the test server before assertions inspect captured state.
func (c *smtpCapture) setMailFrom(value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.mailFrom = value
}

// addRecipient serializes writes from the test server before assertions inspect captured state.
func (c *smtpCapture) addRecipient(value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.recipients = append(c.recipients, value)
}

// setData serializes writes from the test server before assertions inspect captured state.
func (c *smtpCapture) setData(value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = value
}

// TestDriverSendOverPlainSMTP ensures the driver completes the SMTP envelope and transfers rendered MIME bytes.
func TestDriverSendOverPlainSMTP(t *testing.T) {
	capture := &smtpCapture{}
	addr := startSMTPServer(t, capture, nil)

	driver, err := mailsmtp.New(mailsmtp.Config{
		Host: "127.0.0.1",
		Port: addr.Port,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	err = driver.Send(context.Background(), mail.Message{
		From:    &mail.Recipient{Email: "no-reply@example.com", Name: "Example"},
		To:      []mail.Recipient{{Email: "alice@example.com", Name: "Alice"}},
		Cc:      []mail.Recipient{{Email: "manager@example.com", Name: "Manager"}},
		Bcc:     []mail.Recipient{{Email: "audit@example.com", Name: "Audit"}},
		Subject: "Welcome",
		Text:    "hello world",
	})
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	if capture.mailFrom != "FROM:<no-reply@example.com>" {
		t.Fatalf("mail from = %q", capture.mailFrom)
	}
	if len(capture.recipients) != 3 {
		t.Fatalf("recipients = %#v", capture.recipients)
	}
	for _, expected := range []string{
		"Subject: Welcome",
		`To: "Alice" <alice@example.com>`,
		`Cc: "Manager" <manager@example.com>`,
		"hello world",
	} {
		if !strings.Contains(capture.data, expected) {
			t.Fatalf("expected %q in smtp data\n%s", expected, capture.data)
		}
	}
}

// TestDriverSendOverImplicitTLS ensures direct TLS validates the configured server and still completes delivery.
func TestDriverSendOverImplicitTLS(t *testing.T) {
	capture := &smtpCapture{}
	tlsConfig, caPath := sharedTestTLSConfig(t)
	previous := os.Getenv("SSL_CERT_FILE")
	if err := os.Setenv("SSL_CERT_FILE", caPath); err != nil {
		t.Fatalf("set SSL_CERT_FILE: %v", err)
	}
	t.Cleanup(func() {
		if previous == "" {
			_ = os.Unsetenv("SSL_CERT_FILE")
			return
		}
		_ = os.Setenv("SSL_CERT_FILE", previous)
	})

	addr := startSMTPServer(t, capture, tlsConfig)

	driver, err := mailsmtp.New(mailsmtp.Config{
		Host:     "localhost",
		Port:     addr.Port,
		Username: "user",
		Password: "pass",
		ForceTLS: true,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	err = driver.Send(context.Background(), mail.Message{
		From:    &mail.Recipient{Email: "no-reply@example.com"},
		To:      []mail.Recipient{{Email: "alice@example.com"}},
		Subject: "Welcome",
		Text:    "hello over tls",
	})
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	if capture.mailFrom != "FROM:<no-reply@example.com>" {
		t.Fatalf("mail from = %q", capture.mailFrom)
	}
	if len(capture.recipients) != 1 || capture.recipients[0] != "TO:<alice@example.com>" {
		t.Fatalf("recipients = %#v", capture.recipients)
	}
	if !strings.Contains(capture.data, "hello over tls") {
		t.Fatalf("smtp data = %q", capture.data)
	}
}

// TestDriverSendUpgradesWithSTARTTLS ensures opportunistic plaintext is upgraded before authentication or message transfer.
func TestDriverSendUpgradesWithSTARTTLS(t *testing.T) {
	capture := &smtpCapture{}
	serverTLS, caPath := sharedTestTLSConfig(t)
	caPEM, err := os.ReadFile(caPath)
	if err != nil {
		t.Fatalf("read test CA: %v", err)
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(caPEM) {
		t.Fatal("append test CA")
	}
	addr := startSTARTTLSServer(t, capture, serverTLS)

	driver, err := mailsmtp.New(mailsmtp.Config{
		Host: "127.0.0.1",
		Port: addr.Port,
		TLSConfig: &tls.Config{
			ServerName: "localhost",
			RootCAs:    roots,
		},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	err = driver.Send(context.Background(), mail.Message{
		From:    &mail.Recipient{Email: "no-reply@example.com"},
		To:      []mail.Recipient{{Email: "alice@example.com"}},
		Subject: "Welcome",
		Text:    "hello over STARTTLS",
	})
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if capture.mailFrom != "FROM:<no-reply@example.com>" || !strings.Contains(capture.data, "hello over STARTTLS") {
		t.Fatalf("capture = %#v", capture)
	}
}

// TestDriverSendReturnsContextCancellationWhileWaitingForServer ensures stalled SMTP handshakes remain cancelable.
func TestDriverSendReturnsContextCancellationWhileWaitingForServer(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	accepted := make(chan struct{})
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		defer conn.Close()
		close(accepted)
		_, _ = io.Copy(io.Discard, conn)
	}()

	addr := listener.Addr().(*net.TCPAddr)
	driver, err := mailsmtp.New(mailsmtp.Config{Host: "127.0.0.1", Port: addr.Port})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- driver.Send(ctx, mail.Message{
			From:    &mail.Recipient{Email: "no-reply@example.com"},
			To:      []mail.Recipient{{Email: "alice@example.com"}},
			Subject: "Welcome",
			Text:    "hello world",
		})
	}()
	<-accepted
	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Send() error = %v, want %v", err, context.Canceled)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Send() did not return after cancellation")
	}
}

// TestDriverSendOverImplicitTLSFailures ensures certificate and connection failures retain actionable transport errors.
func TestDriverSendOverImplicitTLSFailures(t *testing.T) {
	tlsConfig, caPath := sharedTestTLSConfig(t)
	previous := os.Getenv("SSL_CERT_FILE")
	if err := os.Setenv("SSL_CERT_FILE", caPath); err != nil {
		t.Fatalf("set SSL_CERT_FILE: %v", err)
	}
	t.Cleanup(func() {
		if previous == "" {
			_ = os.Unsetenv("SSL_CERT_FILE")
			return
		}
		_ = os.Setenv("SSL_CERT_FILE", previous)
	})

	tests := []struct {
		name string
		set  func(*smtpCapture)
		want string
	}{
		{name: "auth", set: func(c *smtpCapture) { c.failAuth = true }, want: "535"},
		{name: "mail", set: func(c *smtpCapture) { c.failMail = true }, want: "550"},
		{name: "rcpt", set: func(c *smtpCapture) { c.failRCPT = true }, want: "550"},
		{name: "data", set: func(c *smtpCapture) { c.failData = true }, want: "554"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capture := &smtpCapture{}
			tt.set(capture)
			addr := startSMTPServer(t, capture, tlsConfig)

			driver, err := mailsmtp.New(mailsmtp.Config{
				Host:     "localhost",
				Port:     addr.Port,
				Username: "user",
				Password: "pass",
				ForceTLS: true,
			})
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}

			err = driver.Send(context.Background(), mail.Message{
				From:    &mail.Recipient{Email: "no-reply@example.com"},
				To:      []mail.Recipient{{Email: "alice@example.com"}},
				Subject: "Welcome",
				Text:    "hello over tls",
			})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Send() error = %v, want substring %q", err, tt.want)
			}
		})
	}
}

// startSMTPServer provides a deliberately small implicit-TLS or plaintext server for transport contract tests.
func startSMTPServer(t *testing.T, capture *smtpCapture, tlsConfig *tls.Config) *net.TCPAddr {
	t.Helper()

	listen := func() (net.Listener, error) {
		if tlsConfig != nil {
			return tls.Listen("tcp", "127.0.0.1:0", tlsConfig)
		}
		return net.Listen("tcp", "127.0.0.1:0")
	}

	listener, err := listen()
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(5 * time.Second))

		reader := bufio.NewReader(conn)
		writeLine := func(line string) {
			_, _ = fmt.Fprintf(conn, "%s\r\n", line)
		}

		writeLine("220 localhost ESMTP ready")
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			line = strings.TrimRight(line, "\r\n")

			switch {
			case strings.HasPrefix(line, "EHLO "), strings.HasPrefix(line, "HELO "):
				if tlsConfig != nil {
					_, _ = fmt.Fprintf(conn, "250-localhost\r\n250 AUTH PLAIN\r\n")
				} else {
					writeLine("250 localhost")
				}
			case strings.HasPrefix(line, "AUTH PLAIN"):
				if capture.failAuth {
					writeLine("535 auth failed")
					return
				}
				writeLine("235 authenticated")
			case strings.HasPrefix(line, "MAIL "):
				if capture.failMail {
					writeLine("550 mail from rejected")
					return
				}
				capture.setMailFrom(strings.TrimPrefix(line, "MAIL "))
				writeLine("250 ok")
			case strings.HasPrefix(line, "RCPT "):
				if capture.failRCPT {
					writeLine("550 rcpt rejected")
					return
				}
				capture.addRecipient(strings.TrimPrefix(line, "RCPT "))
				writeLine("250 ok")
			case line == "DATA":
				if capture.failData {
					writeLine("554 data rejected")
					return
				}
				writeLine("354 end data with <CR><LF>.<CR><LF>")
				var data strings.Builder
				for {
					part, err := reader.ReadString('\n')
					if err != nil {
						return
					}
					if part == ".\r\n" {
						break
					}
					data.WriteString(part)
				}
				capture.setData(data.String())
				writeLine("250 queued")
			case line == "QUIT":
				writeLine("221 bye")
				return
			default:
				writeLine("250 ok")
			}
		}
	}()

	return listener.Addr().(*net.TCPAddr)
}

// startSTARTTLSServer proves the driver performs the advertised upgrade before sending envelope data.
func startSTARTTLSServer(t *testing.T, capture *smtpCapture, tlsConfig *tls.Config) *net.TCPAddr {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
		reader := bufio.NewReader(conn)
		writeLine := func(line string) { _, _ = fmt.Fprintf(conn, "%s\r\n", line) }

		writeLine("220 localhost ESMTP ready")
		if _, readErr := reader.ReadString('\n'); readErr != nil {
			return
		}
		_, _ = fmt.Fprint(conn, "250-localhost\r\n250 STARTTLS\r\n")
		line, readErr := reader.ReadString('\n')
		if readErr != nil || strings.TrimSpace(line) != "STARTTLS" {
			return
		}
		writeLine("220 begin TLS")

		tlsConn := tls.Server(conn, tlsConfig)
		if handshakeErr := tlsConn.Handshake(); handshakeErr != nil {
			return
		}
		reader = bufio.NewReader(tlsConn)
		if _, readErr := reader.ReadString('\n'); readErr != nil {
			return
		}
		writeTLSLine := func(line string) { _, _ = fmt.Fprintf(tlsConn, "%s\r\n", line) }
		writeTLSLine("250 localhost")

		for {
			line, readErr = reader.ReadString('\n')
			if readErr != nil {
				return
			}
			line = strings.TrimRight(line, "\r\n")
			switch {
			case strings.HasPrefix(line, "MAIL "):
				capture.setMailFrom(strings.TrimPrefix(line, "MAIL "))
				writeTLSLine("250 ok")
			case strings.HasPrefix(line, "RCPT "):
				capture.addRecipient(strings.TrimPrefix(line, "RCPT "))
				writeTLSLine("250 ok")
			case line == "DATA":
				writeTLSLine("354 end data with <CR><LF>.<CR><LF>")
				var data strings.Builder
				for {
					part, partErr := reader.ReadString('\n')
					if partErr != nil {
						return
					}
					if part == ".\r\n" {
						break
					}
					data.WriteString(part)
				}
				capture.setData(data.String())
				writeTLSLine("250 queued")
			case line == "QUIT":
				writeTLSLine("221 bye")
				return
			default:
				writeTLSLine("250 ok")
			}
		}
	}()

	return listener.Addr().(*net.TCPAddr)
}

// testTLSConfig creates a private CA because verification behavior is part of the transport contract.
func testTLSConfig(t *testing.T) (*tls.Config, []byte) {
	t.Helper()

	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate ca key: %v", err)
	}
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "mail test ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		IsCA:                  true,
		BasicConstraintsValid: true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("create ca cert: %v", err)
	}

	serverKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate server key: %v", err)
	}
	serverTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"localhost"},
	}
	serverDER, err := x509.CreateCertificate(rand.Reader, serverTemplate, caTemplate, &serverKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("create server cert: %v", err)
	}

	serverCert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: serverDER})
	serverKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(serverKey)})
	cert, err := tls.X509KeyPair(serverCert, serverKeyPEM)
	if err != nil {
		t.Fatalf("x509 key pair: %v", err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})
}

// sharedTestTLSConfig amortizes key generation without weakening certificate verification in individual tests.
func sharedTestTLSConfig(t *testing.T) (*tls.Config, string) {
	t.Helper()
	sharedTLSOnce.Do(func() {
		sharedTLSConfig, sharedTLSCAPEM = testTLSConfig(t)
	})
	caPath := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(caPath, sharedTLSCAPEM, 0o644); err != nil {
		t.Fatalf("write ca cert: %v", err)
	}
	return sharedTLSConfig, caPath
}
