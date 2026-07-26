<p align="center">
  <img src="./docs/assets/logo.png?v=1" width="300" alt="mail logo">
</p>

<p align="center">
  Fluent email composition and pluggable delivery for GoForj packages and apps.
</p>

<p align="center">
  <a href="https://pkg.go.dev/github.com/goforj/mail"><img src="https://pkg.go.dev/badge/github.com/goforj/mail.svg" alt="Go Reference"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License: MIT"></a>
  <a href="https://github.com/goforj/mail/actions/workflows/ci.yml"><img src="https://github.com/goforj/mail/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://golang.org"><img src="https://img.shields.io/badge/go-1.24+-blue?logo=go" alt="Go version"></a>
  <img src="https://img.shields.io/github/v/tag/goforj/mail?label=version&sort=semver&filter=%21mailses%2A" alt="Latest tag">
  <a href="https://goreportcard.com/report/github.com/goforj/mail"><img src="https://goreportcard.com/badge/github.com/goforj/mail" alt="Go Report Card"></a>
  <a href="https://codecov.io/gh/goforj/mail"><img src="https://codecov.io/gh/goforj/mail/graph/badge.svg?token=PJJVA66P5X" alt="Codecov"></a>
<!-- test-count:embed:start -->
<img src="https://img.shields.io/badge/unit_tests-118-brightgreen" alt="Unit tests (executed count)">
<!-- test-count:embed:end -->
<!-- package-coverage:embed:start -->
<br>
<img src="https://img.shields.io/badge/mail-95.2%25-4c9a2a" alt="mail coverage">
<img src="https://img.shields.io/badge/mailfake-91.2%25-4c9a2a" alt="mailfake coverage">
<img src="https://img.shields.io/badge/maillog-86.1%25-4c9a2a" alt="maillog coverage">
<img src="https://img.shields.io/badge/mailmailgun-83.7%25-4c9a2a" alt="mailmailgun coverage">
<img src="https://img.shields.io/badge/mailpostmark-91.3%25-4c9a2a" alt="mailpostmark coverage">
<img src="https://img.shields.io/badge/mailresend-94.6%25-4c9a2a" alt="mailresend coverage">
<img src="https://img.shields.io/badge/mailsendgrid-96.1%25-4c9a2a" alt="mailsendgrid coverage">
<img src="https://img.shields.io/badge/mailses-93.0%25-4c9a2a" alt="mailses coverage">
<img src="https://img.shields.io/badge/mailsmtp-88.4%25-4c9a2a" alt="mailsmtp coverage">
<!-- package-coverage:embed:end -->
</p>

## Installation

```bash
go get github.com/goforj/mail
```

## Quick Start

```go
package main

import (
	"context"
	"log"

	"github.com/goforj/mail"
	"github.com/goforj/mail/mailsmtp"
)

func main() {
	driver, err := mailsmtp.New(mailsmtp.Config{
		Host:     "smtp.example.com",
		Port:     587,
		Username: "smtp-user",
		Password: "smtp-password",
	})
	if err != nil {
		log.Fatal(err)
	}

	mailer := mail.New(
		driver,
		mail.WithDefaultFrom("no-reply@example.com", "Example"),
	)

	err = mailer.Message().
		To("alice@example.com", "Alice").
		Subject("Welcome").
		Text("hello world").
		Send(context.Background())
	if err != nil {
		log.Fatal(err)
	}
}
```

## Gmail via SMTP

Gmail does not need its own driver. Use `mailsmtp` with Gmail's SMTP host and an app password:

```go
driver, err := mailsmtp.New(mailsmtp.Config{
	Host:     "smtp.gmail.com",
	Port:     587,
	Username: "you@gmail.com",
	Password: "gmail-app-password",
})
```

Notes:

- Use a Google app password, not your normal account password.
- `587` is the usual STARTTLS port. Use `465` with `ForceTLS: true` if you explicitly want implicit TLS.
- Gmail is fine for personal or low-volume transactional sending, but a dedicated provider like Resend, Postmark, Mailgun, or SendGrid is usually a better production default.

## Driver Capabilities

| Driver | HTML/Text | Headers | Tags | Metadata | Attachments | Notes |
|:--|:--:|:--:|:--:|:--:|:--:|:--|
| mailsmtp | ✓ | ✓ | x | x | ✓ | Covers Gmail and other SMTP providers. |
| mailresend | ✓ | ✓ | ✓ | ✓ | ✓ | API-backed transactional delivery. |
| mailpostmark | ✓ | ✓ | ✓ | ✓ | ✓ | First tag is native; additional tags are mapped into metadata. |
| mailmailgun | ✓ | ✓ | ✓ | ✓ | ✓ | Uses Mailgun multipart message uploads. |
| mailsendgrid | ✓ | ✓ | ✓ | ✓ | ✓ | Maps tags to categories and metadata to custom args. |
| mailses | ✓ | ✓ | ✓ | ✓ | ✓ | Uses SES raw email with the same MIME rendering as SMTP. |
| maillog | ✓ | ✓ | x | x | ✓ | Local/dev inspection only; logs the composed message. |
| mailfake | ✓ | ✓ | ✓ | ✓ | ✓ | Test helper; captures the full portable message. |

## Delivery Contract

Every message is validated before a bundled driver performs I/O. A deliverable message requires:

- a `From` recipient (either explicit or supplied by `WithDefaultFrom`);
- at least one `To`, `Cc`, or `Bcc` recipient;
- a subject and at least one text or HTML body;
- single-address recipient fields, safe custom headers, valid provider metadata keys, and valid attachment MIME metadata.

`mail.New` requires a driver and panics for a nil or typed-nil driver. This is a construction invariant: use `mailfake.New()` when a test needs a harmless driver. The zero-value `MessageBuilder` remains useful for standalone composition and `Build`; calling `Send` on an unbound builder returns `mail.ErrMissingMailer`.

Custom headers cannot replace envelope-owned fields such as `From`, `To`, `Subject`, `Content-Type`, or `MIME-Version`, and names must be unique without regard to case. Header, subject, attachment, and metadata validation rejects control characters before data reaches SMTP or multipart encoders.

API-backed drivers return an exported, provider-specific `ResponseError` for non-2xx responses. It exposes the HTTP status and a safe request ID when the provider supplies one. Provider response bodies are bounded and deliberately omitted from error strings so logs cannot accidentally capture credentials or message content.

## SMTP TLS Policy

`mailsmtp` uses opportunistic STARTTLS when the server advertises it. Set `ForceTLS: true` for implicit TLS, commonly used on port 465.

TLS defaults are explicit:

- certificate and hostname verification are enabled;
- `ServerName` defaults to `Config.Host`;
- the minimum protocol version defaults to TLS 1.2;
- a supplied `TLSConfig` is cloned during construction, so later caller mutation cannot change a running driver.

Use `TLSConfig` for a private CA or a stricter minimum version. Disabling verification is not recommended.

## Unreleased Compatibility Notes

This quality pass deliberately tightens the pre-v1 contract:

- `Message.Validate` now returns `ErrMissingFrom` when no sender remains after defaults;
- `mail.New(nil)` and `maillog.New(nil)` now fail fast by panicking;
- direct `mailfake` and `maillog` sends now honor cancellation and use the same validation contract as network drivers;
- malformed custom HTTP endpoints fail during driver construction instead of the first send;
- SMTP rejects negative or out-of-range ports, preserves body whitespace, safely encodes Unicode subjects and filenames, and emits quoted-printable text bodies;
- SendGrid headers and custom arguments are emitted only inside `personalizations`, matching its API schema.
- The root module and `mailses` are staged as a coordinated v0.3.0 release; publish the root tag before `mailses/v0.3.0` so the SES module never depends on a local replacement. The checksum-staging sequence is documented in [scripts/RELEASE.md](scripts/RELEASE.md).

GoForj-generated mailers already configure `WithDefaultFrom`, so their normal send shape is unchanged.

## API

<!-- api:embed:start -->
## API Index

| Group | Functions |
|------:|:-----------|
| **Composition** | [Mailer.Message](#mailer-message) · [MessageBuilder.Bcc](#messagebuilder-bcc) · [MessageBuilder.Cc](#messagebuilder-cc) · [MessageBuilder.From](#messagebuilder-from) · [MessageBuilder.Message](#messagebuilder-message) · [MessageBuilder.ReplyTo](#messagebuilder-replyto) · [MessageBuilder.To](#messagebuilder-to) |
| **Construction** | [New](#new) |
| **Content** | [MessageBuilder.Attach](#messagebuilder-attach) · [MessageBuilder.AttachFile](#messagebuilder-attachfile) · [MessageBuilder.HTML](#messagebuilder-html) · [MessageBuilder.Header](#messagebuilder-header) · [MessageBuilder.Metadata](#messagebuilder-metadata) · [MessageBuilder.Subject](#messagebuilder-subject) · [MessageBuilder.Tag](#messagebuilder-tag) · [MessageBuilder.Text](#messagebuilder-text) |
| **Defaults** | [WithDefaultFrom](#withdefaultfrom) · [WithDefaultHeader](#withdefaultheader) · [WithDefaultMetadata](#withdefaultmetadata) · [WithDefaultReplyTo](#withdefaultreplyto) · [WithDefaultTag](#withdefaulttag) |
| **Delivery** | [Mailer.Send](#mailer-send) · [MessageBuilder.Build](#messagebuilder-build) · [MessageBuilder.Send](#messagebuilder-send) |
| **Logging** | [maillog.Driver.Send](#maillog-driver-send) · [maillog.New](#maillog-new) · [maillog.WithBodies](#maillog-withbodies) · [maillog.WithNow](#maillog-withnow) |
| **Mailgun** | [mailmailgun.Driver.Send](#mailmailgun-driver-send) · [mailmailgun.New](#mailmailgun-new) · [mailmailgun.ResponseError.Error](#mailmailgun-responseerror-error) |
| **Message Model** | [AttachmentFromBytes](#attachmentfrombytes) · [AttachmentFromPath](#attachmentfrompath) · [Message.Clone](#message-clone) · [Message.Validate](#message-validate) |
| **Postmark** | [mailpostmark.Driver.Send](#mailpostmark-driver-send) · [mailpostmark.New](#mailpostmark-new) · [mailpostmark.ResponseError.Error](#mailpostmark-responseerror-error) |
| **Resend** | [mailresend.Driver.Send](#mailresend-driver-send) · [mailresend.New](#mailresend-new) · [mailresend.ResponseError.Error](#mailresend-responseerror-error) |
| **SES** | [mailses.Driver.Send](#mailses-driver-send) · [mailses.New](#mailses-new) |
| **SMTP** | [mailsmtp.Driver.Send](#mailsmtp-driver-send) · [mailsmtp.New](#mailsmtp-new) · [mailsmtp.Render](#mailsmtp-render) |
| **SendGrid** | [mailsendgrid.Driver.Send](#mailsendgrid-driver-send) · [mailsendgrid.New](#mailsendgrid-new) · [mailsendgrid.ResponseError.Error](#mailsendgrid-responseerror-error) |
| **Testing** | [mailfake.Driver.Last](#mailfake-driver-last) · [mailfake.Driver.Messages](#mailfake-driver-messages) · [mailfake.Driver.Reset](#mailfake-driver-reset) · [mailfake.Driver.Send](#mailfake-driver-send) · [mailfake.Driver.SentCount](#mailfake-driver-sentcount) · [mailfake.Driver.SetError](#mailfake-driver-seterror) · [mailfake.New](#mailfake-new) |


## API Reference

_Generated from public API comments and examples._

### Composition

#### <a id="mailer-message"></a>Mailer.Message

Message starts a new fluent message builder bound to this mailer.

```go
fake := mailfake.New()
mailer := mail.New(fake, mail.WithDefaultFrom("no-reply@example.com", "Example"))
_ = mailer.Message().
	To("alice@example.com", "Alice").
	Subject("Welcome").
	Text("hello world").
	Send(context.Background())
fmt.Println(fake.SentCount())
// 1
```

#### <a id="messagebuilder-bcc"></a>MessageBuilder.Bcc

Bcc appends one blind-carbon-copy recipient.

```go
msg, _ := mail.New(mailfake.New()).Message().
	From("no-reply@example.com", "Example").
	To("alice@example.com", "Alice").
	Bcc("audit@example.com", "Audit").
	Subject("Welcome").
	Text("hello world").
	Build()
fmt.Println(msg.Bcc[0].Email)
// audit@example.com
```

#### <a id="messagebuilder-cc"></a>MessageBuilder.Cc

Cc appends one carbon-copy recipient.

```go
msg, _ := mail.New(mailfake.New()).Message().
	From("no-reply@example.com", "Example").
	To("alice@example.com", "Alice").
	Cc("manager@example.com", "Manager").
	Subject("Welcome").
	Text("hello world").
	Build()
fmt.Println(msg.Cc[0].Email)
// manager@example.com
```

#### <a id="messagebuilder-from"></a>MessageBuilder.From

From sets the from recipient.

```go
msg, _ := mail.New(mailfake.New()).Message().
	From("team@example.com", "Example Team").
	To("alice@example.com", "Alice").
	Subject("Welcome").
	Text("hello world").
	Build()
fmt.Println(msg.From.Email)
// team@example.com
```

#### <a id="messagebuilder-message"></a>MessageBuilder.Message

Message returns the currently composed message without applying mailer defaults.

```go
msg := mail.New(mailfake.New()).Message().
	To("alice@example.com", "Alice").
	Subject("Welcome").
	Text("hello world").
	Message()
fmt.Println(msg.Subject)
// Welcome
```

#### <a id="messagebuilder-replyto"></a>MessageBuilder.ReplyTo

ReplyTo appends one reply-to recipient.

```go
msg, _ := mail.New(mailfake.New()).Message().
	From("no-reply@example.com", "Example").
	To("alice@example.com", "Alice").
	ReplyTo("support@example.com", "Support").
	Subject("Welcome").
	Text("hello world").
	Build()
fmt.Println(msg.ReplyTo[0].Email)
// support@example.com
```

#### <a id="messagebuilder-to"></a>MessageBuilder.To

To appends one primary recipient.

```go
msg, _ := mail.New(mailfake.New()).Message().
	From("no-reply@example.com", "Example").
	To("alice@example.com", "Alice").
	Subject("Welcome").
	Text("hello world").
	Build()
fmt.Println(len(msg.To))
// 1
```

### Construction

#### <a id="new"></a>New

New creates a Mailer backed by the provided driver.
New panics when driver is nil because a Mailer cannot deliver without its required collaborator.

```go
fake := mailfake.New()
mailer := mail.New(fake, mail.WithDefaultFrom("no-reply@example.com", "Example"))
fmt.Println(mailer != nil)
// true
```

### Content

#### <a id="messagebuilder-attach"></a>MessageBuilder.Attach

Attach appends one in-memory attachment.

```go
msg := mail.New(mailfake.New()).Message().
	To("alice@example.com", "Alice").
	Subject("Welcome").
	Text("hello world").
	Attach("report.txt", "text/plain", []byte("hello world")).
	Message()
fmt.Println(msg.Attachments[0].Filename)
// report.txt
```

#### <a id="messagebuilder-attachfile"></a>MessageBuilder.AttachFile

AttachFile loads one attachment from disk and appends it to the message.

```go
_ = os.WriteFile("report.txt", []byte("hello world"), 0o644)
defer os.Remove("report.txt")
msg, _ := mail.New(mailfake.New()).Message().
	From("no-reply@example.com", "Example").
	To("alice@example.com", "Alice").
	Subject("Welcome").
	Text("hello world").
	AttachFile("report.txt").
	Build()
fmt.Println(msg.Attachments[0].Filename)
// report.txt
```

#### <a id="messagebuilder-html"></a>MessageBuilder.HTML

HTML sets the HTML body.

```go
msg := mail.New(mailfake.New()).Message().
	To("alice@example.com", "Alice").
	Subject("Welcome").
	HTML("<p>hello world</p>").
	Message()
fmt.Println(msg.HTML)
// <p>hello world</p>
```

#### <a id="messagebuilder-header"></a>MessageBuilder.Header

Header sets or replaces one message header.

```go
message, _ := mail.New(mailfake.New()).Message().
	From("no-reply@example.com", "Example").
	To("alice@example.com", "Alice").
	Subject("Welcome").
	Text("hello world").
	Header("X-Request-ID", "req_123").
	Tag("welcome").
	Metadata("tenant_id", "tenant_123").
	Build()
fmt.Println(message.Headers["X-Request-ID"])
// req_123
```

#### <a id="messagebuilder-metadata"></a>MessageBuilder.Metadata

Metadata sets one provider-facing metadata key/value pair.

```go
msg := mail.New(mailfake.New()).Message().
	To("alice@example.com", "Alice").
	Subject("Welcome").
	Text("hello world").
	Metadata("tenant_id", "tenant_123").
	Message()
fmt.Println(msg.Metadata["tenant_id"])
// tenant_123
```

#### <a id="messagebuilder-subject"></a>MessageBuilder.Subject

Subject sets the message subject.

```go
msg := mail.New(mailfake.New()).Message().
	To("alice@example.com", "Alice").
	Subject("Welcome").
	Text("hello world").
	Message()
fmt.Println(msg.Subject)
// Welcome
```

#### <a id="messagebuilder-tag"></a>MessageBuilder.Tag

Tag appends one provider-facing message tag.

```go
msg := mail.New(mailfake.New()).Message().
	To("alice@example.com", "Alice").
	Subject("Welcome").
	Text("hello world").
	Tag("welcome").
	Message()
fmt.Println(msg.Tags[0])
// welcome
```

#### <a id="messagebuilder-text"></a>MessageBuilder.Text

Text sets the plain text body.

```go
msg := mail.New(mailfake.New()).Message().
	To("alice@example.com", "Alice").
	Subject("Welcome").
	Text("hello world").
	Message()
fmt.Println(msg.Text)
// hello world
```

### Defaults

#### <a id="withdefaultfrom"></a>WithDefaultFrom

WithDefaultFrom configures the default from recipient applied when a message omits one.

```go
mailer := mail.New(
	mailfake.New(),
	mail.WithDefaultFrom("no-reply@example.com", "Example"),
)
fmt.Println(mailer != nil)
// true
```

#### <a id="withdefaultheader"></a>WithDefaultHeader

WithDefaultHeader configures a header applied when a message omits that header key.

```go
msg, _ := mail.New(
	mailfake.New(),
	mail.WithDefaultFrom("no-reply@example.com", "Example"),
	mail.WithDefaultHeader("X-App", "goforj"),
).Message().
	To("alice@example.com", "Alice").
	Subject("Welcome").
	Text("hello world").
	Build()
fmt.Println(msg.Headers["X-App"])
// goforj
```

#### <a id="withdefaultmetadata"></a>WithDefaultMetadata

WithDefaultMetadata configures metadata applied when a message omits that metadata key.

```go
msg, _ := mail.New(
	mailfake.New(),
	mail.WithDefaultFrom("no-reply@example.com", "Example"),
	mail.WithDefaultMetadata("tenant_id", "tenant_123"),
).Message().
	To("alice@example.com", "Alice").
	Subject("Welcome").
	Text("hello world").
	Build()
fmt.Println(msg.Metadata["tenant_id"])
// tenant_123
```

#### <a id="withdefaultreplyto"></a>WithDefaultReplyTo

WithDefaultReplyTo configures the default reply-to recipients applied when a message omits them.

```go
mailer := mail.New(
	mailfake.New(),
	mail.WithDefaultFrom("no-reply@example.com", "Example"),
	mail.WithDefaultReplyTo(mail.Recipient{Email: "support@example.com", Name: "Support"}),
)
msg, _ := mailer.Message().
	To("alice@example.com", "Alice").
	Subject("Welcome").
	Text("hello world").
	Build()
fmt.Println(msg.ReplyTo[0].Email)
// support@example.com
```

#### <a id="withdefaulttag"></a>WithDefaultTag

WithDefaultTag configures a tag prepended to every message sent by the mailer.

```go
msg, _ := mail.New(
	mailfake.New(),
	mail.WithDefaultFrom("no-reply@example.com", "Example"),
	mail.WithDefaultTag("transactional"),
).Message().
	To("alice@example.com", "Alice").
	Subject("Welcome").
	Text("hello world").
	Build()
fmt.Println(msg.Tags[0])
// transactional
```

### Delivery

#### <a id="mailer-send"></a>Mailer.Send

Send validates the message, applies defaults, and delegates delivery to the driver.

```go
mailer := mail.New(mailfake.New(), mail.WithDefaultFrom("no-reply@example.com", "Example"))
err := mailer.Send(context.Background(), mail.Message{
	To:      []mail.Recipient{{Email: "alice@example.com", Name: "Alice"}},
	Subject: "Welcome",
	Text:    "hello world",
})
fmt.Println(err == nil)
// true
```

#### <a id="messagebuilder-build"></a>MessageBuilder.Build

Build applies defaults, validates, and returns the composed message without sending it.

```go
msg, _ := mail.New(
	mailfake.New(),
	mail.WithDefaultFrom("no-reply@example.com", "Example"),
).Message().
	To("alice@example.com", "Alice").
	Subject("Welcome").
	Text("hello world").
	Build()
fmt.Println(msg.From.Email)
// no-reply@example.com
```

#### <a id="messagebuilder-send"></a>MessageBuilder.Send

Send delegates the composed message to the bound mailer.

```go
fake := mailfake.New()
_ = mail.New(fake).Message().
	From("no-reply@example.com", "Example").
	To("alice@example.com", "Alice").
	Subject("Welcome").
	Text("hello world").
	Send(context.Background())
fmt.Println(fake.SentCount())
// 1
```

### Logging

#### <a id="maillog-driver-send"></a>maillog.Driver.Send

Send validates the message and writes one JSON log record.

```go
var out bytes.Buffer
_ = maillog.New(&out).Send(context.Background(), mail.Message{
	From:    &mail.Recipient{Email: "no-reply@example.com"},
	To:      []mail.Recipient{{Email: "alice@example.com"}},
	Subject: "Welcome",
	Text:    "hello world",
})
fmt.Println(strings.Contains(out.String(), "\"subject\":\"Welcome\""))
// true
```

#### <a id="maillog-new"></a>maillog.New

New creates a log mail driver that writes one JSON record per sent message.
New panics when writer is nil because output is the driver's required collaborator.

```go
var out bytes.Buffer
mailer := maillog.New(&out)
_ = mail.New(mailer).Send(context.Background(), mail.Message{
	From:    &mail.Recipient{Email: "no-reply@example.com"},
	To:      []mail.Recipient{{Email: "alice@example.com"}},
	Subject: "Welcome",
	Text:    "hello world",
})
fmt.Println(strings.Contains(out.String(), "\"subject\":\"Welcome\""))
// true
```

#### <a id="maillog-withbodies"></a>maillog.WithBodies

WithBodies controls whether HTML and text bodies are included in log output.

```go
var out bytes.Buffer
mailer := maillog.New(&out, maillog.WithBodies(true))
_ = mail.New(mailer).Send(context.Background(), mail.Message{
	From:    &mail.Recipient{Email: "no-reply@example.com"},
	To:      []mail.Recipient{{Email: "alice@example.com"}},
	Subject: "Welcome",
	Text:    "hello world",
})
fmt.Println(strings.Contains(out.String(), "\"text\":\"hello world\""))
// true
```

#### <a id="maillog-withnow"></a>maillog.WithNow

WithNow overrides the timestamp source used by log entries.

```go
var out bytes.Buffer
mailer := maillog.New(&out, maillog.WithNow(func() time.Time {
	return time.Date(2026, time.April, 19, 0, 0, 0, 0, time.UTC)
}))
_ = mail.New(mailer).Send(context.Background(), mail.Message{
	From:    &mail.Recipient{Email: "no-reply@example.com"},
	To:      []mail.Recipient{{Email: "alice@example.com"}},
	Subject: "Welcome",
	Text:    "hello world",
})
fmt.Println(strings.Contains(out.String(), "2026-04-19T00:00:00Z"))
// true
```

### Mailgun

#### <a id="mailmailgun-driver-send"></a>mailmailgun.Driver.Send

Send validates and transmits one message through Mailgun.

```go
driver, _ := mailmailgun.New(mailmailgun.Config{
	Domain:   "mg.example.com",
	APIKey:   "key-test",
	Endpoint: "http://127.0.0.1:1",
})
err := driver.Send(context.Background(), mail.Message{
	From:    &mail.Recipient{Email: "no-reply@example.com"},
	To:      []mail.Recipient{{Email: "alice@example.com"}},
	Subject: "Welcome",
	Text:    "hello world",
})
fmt.Println(err == nil)
// false
```

#### <a id="mailmailgun-new"></a>mailmailgun.New

New creates a Mailgun mail driver from the given config.

```go
driver, _ := mailmailgun.New(mailmailgun.Config{
	Domain: "mg.example.com",
	APIKey: "key-test",
})
fmt.Println(driver != nil)
// true
```

#### <a id="mailmailgun-responseerror-error"></a>mailmailgun.ResponseError.Error

Error formats the provider status and safe correlation identifier without including response content.

### Message Model

#### <a id="attachmentfrombytes"></a>AttachmentFromBytes

AttachmentFromBytes creates one attachment from in-memory content.

```go
attachment := mail.AttachmentFromBytes("report.txt", "text/plain", []byte("hello world"))
fmt.Println(attachment.Filename)
// report.txt
```

#### <a id="attachmentfrompath"></a>AttachmentFromPath

AttachmentFromPath loads one attachment from a local file path.

```go
_ = os.WriteFile("report.txt", []byte("hello world"), 0o644)
defer os.Remove("report.txt")
attachment, _ := mail.AttachmentFromPath("report.txt")
fmt.Println(attachment.Filename)
// report.txt
```

#### <a id="message-clone"></a>Message.Clone

Clone returns a copy of the message safe for reuse in tests and builders.

```go
original := mail.Message{
	To:      []mail.Recipient{{Email: "alice@example.com", Name: "Alice"}},
	Subject: "Welcome",
	Text:    "hello world",
}
cloned := original.Clone()
cloned.Subject = "Changed"
fmt.Println(original.Subject)
// Welcome
```

#### <a id="message-validate"></a>Message.Validate

Validate checks that the message has valid recipients, subject, body, and headers.

```go
err := (mail.Message{
	From:    &mail.Recipient{Email: "no-reply@example.com", Name: "Example"},
	To:      []mail.Recipient{{Email: "alice@example.com", Name: "Alice"}},
	Subject: "Welcome",
	Text:    "hello world",
}).Validate()
fmt.Println(err == nil)
// true
```

### Postmark

#### <a id="mailpostmark-driver-send"></a>mailpostmark.Driver.Send

Send validates and transmits one message through Postmark.

```go
driver, _ := mailpostmark.New(mailpostmark.Config{
	ServerToken: "pm_test_token",
	Endpoint:    "http://127.0.0.1:1",
})
err := driver.Send(context.Background(), mail.Message{
	From:    &mail.Recipient{Email: "no-reply@example.com"},
	To:      []mail.Recipient{{Email: "alice@example.com"}},
	Subject: "Welcome",
	Text:    "hello world",
})
fmt.Println(err == nil)
// false
```

#### <a id="mailpostmark-new"></a>mailpostmark.New

New creates a Postmark mail driver from the given config.

```go
driver, _ := mailpostmark.New(mailpostmark.Config{
	ServerToken: "pm_test_token",
})
fmt.Println(driver != nil)
// true
```

#### <a id="mailpostmark-responseerror-error"></a>mailpostmark.ResponseError.Error

Error formats provider codes and a safe correlation identifier without including response content.

### Resend

#### <a id="mailresend-driver-send"></a>mailresend.Driver.Send

Send validates and transmits one message through Resend.

```go
driver, _ := mailresend.New(mailresend.Config{
	APIKey:   "re_test_key",
	Endpoint: "http://127.0.0.1:1",
})
err := driver.Send(context.Background(), mail.Message{
	From:    &mail.Recipient{Email: "no-reply@example.com"},
	To:      []mail.Recipient{{Email: "alice@example.com"}},
	Subject: "Welcome",
	Text:    "hello world",
})
fmt.Println(err == nil)
// false
```

#### <a id="mailresend-new"></a>mailresend.New

New creates a Resend mail driver from the given config.

```go
driver, _ := mailresend.New(mailresend.Config{
	APIKey: "re_test_key",
})
fmt.Println(driver != nil)
// true
```

#### <a id="mailresend-responseerror-error"></a>mailresend.ResponseError.Error

Error formats the provider status and safe correlation identifier without including response content.

### SES

#### <a id="mailses-driver-send"></a>mailses.Driver.Send

Send validates and transmits one message through Amazon SES.

```go
driver, _ := mailses.New(mailses.Config{
	Region:          "us-east-1",
	AccessKeyID:     "test",
	SecretAccessKey: "test",
	Endpoint:        "http://127.0.0.1:1",
})
err := driver.Send(context.Background(), mail.Message{
	From:    &mail.Recipient{Email: "no-reply@example.com"},
	To:      []mail.Recipient{{Email: "alice@example.com"}},
	Subject: "Welcome",
	Text:    "hello world",
})
fmt.Println(err == nil)
// false
```

#### <a id="mailses-new"></a>mailses.New

New creates an Amazon SES mail driver from the given config.

```go
driver, _ := mailses.New(mailses.Config{
	Region:          "us-east-1",
	AccessKeyID:     "test",
	SecretAccessKey: "test",
})
fmt.Println(driver != nil)
// true
```

### SMTP

#### <a id="mailsmtp-driver-send"></a>mailsmtp.Driver.Send

Send validates and transmits one message over SMTP.

```go
driver, _ := mailsmtp.New(mailsmtp.Config{
	Host: "smtp.example.com",
	Port: 587,
})
err := driver.Send(context.Background(), mail.Message{
	From:    &mail.Recipient{Email: "no-reply@example.com"},
	To:      []mail.Recipient{{Email: "alice@example.com"}},
	Subject: "Welcome",
	Text:    "hello world",
})
fmt.Println(err == nil)
// false
```

#### <a id="mailsmtp-new"></a>mailsmtp.New

New creates an SMTP mail driver from the given config.
TLS defaults to the configured host for ServerName, a minimum of TLS 1.2, and normal certificate verification.

```go
driver, _ := mailsmtp.New(mailsmtp.Config{
	Host: "smtp.example.com",
	Port: 587,
})
fmt.Println(driver != nil)
// true
```

gmail:

```go
driver, _ := mailsmtp.New(mailsmtp.Config{
	Host:     "smtp.gmail.com",
	Port:     587,
	Username: "you@gmail.com",
	Password: "gmail-app-password",
})
fmt.Println(driver != nil)
// true
```

#### <a id="mailsmtp-render"></a>mailsmtp.Render

Render turns one message into an RFC 822 style SMTP payload.

```go
raw, _ := mailsmtp.Render(mail.Message{
	From:    &mail.Recipient{Email: "no-reply@example.com", Name: "Example"},
	To:      []mail.Recipient{{Email: "alice@example.com", Name: "Alice"}},
	Subject: "Welcome",
	Text:    "hello world",
})
fmt.Println(strings.Contains(string(raw), "Subject: Welcome"))
// true
```

### SendGrid

#### <a id="mailsendgrid-driver-send"></a>mailsendgrid.Driver.Send

Send validates and transmits one message through SendGrid.

```go
driver, _ := mailsendgrid.New(mailsendgrid.Config{
	APIKey:   "SG.test_key",
	Endpoint: "http://127.0.0.1:1",
})
err := driver.Send(context.Background(), mail.Message{
	From:    &mail.Recipient{Email: "no-reply@example.com"},
	To:      []mail.Recipient{{Email: "alice@example.com"}},
	Subject: "Welcome",
	Text:    "hello world",
})
fmt.Println(err == nil)
// false
```

#### <a id="mailsendgrid-new"></a>mailsendgrid.New

New creates a SendGrid mail driver from the given config.

```go
driver, _ := mailsendgrid.New(mailsendgrid.Config{
	APIKey: "SG.test_key",
})
fmt.Println(driver != nil)
// true
```

#### <a id="mailsendgrid-responseerror-error"></a>mailsendgrid.ResponseError.Error

Error formats the provider status and safe correlation identifier without including response content.

### Testing

#### <a id="mailfake-driver-last"></a>mailfake.Driver.Last

Last returns the last recorded message when one exists.

```go
fake := mailfake.New()
_ = mail.New(fake).Send(context.Background(), mail.Message{
	From:    &mail.Recipient{Email: "no-reply@example.com"},
	To:      []mail.Recipient{{Email: "alice@example.com"}},
	Subject: "Welcome",
	Text:    "hello world",
})
last, _ := fake.Last()
fmt.Println(last.Subject)
// Welcome
```

#### <a id="mailfake-driver-messages"></a>mailfake.Driver.Messages

Messages returns a copy of every recorded message.

```go
fake := mailfake.New()
_ = mail.New(fake).Send(context.Background(), mail.Message{
	From:    &mail.Recipient{Email: "no-reply@example.com"},
	To:      []mail.Recipient{{Email: "alice@example.com"}},
	Subject: "Welcome",
	Text:    "hello world",
})
fmt.Println(len(fake.Messages()))
// 1
```

#### <a id="mailfake-driver-reset"></a>mailfake.Driver.Reset

Reset clears recorded messages and any configured send error.

```go
fake := mailfake.New()
_ = fake.Send(context.Background(), mail.Message{
	From:    &mail.Recipient{Email: "no-reply@example.com"},
	To:      []mail.Recipient{{Email: "alice@example.com"}},
	Subject: "Welcome",
	Text:    "hello world",
})
fake.Reset()
fmt.Println(fake.SentCount())
// 0
```

#### <a id="mailfake-driver-send"></a>mailfake.Driver.Send

Send validates and records the message, returning the configured delivery error when set.

```go
fake := mailfake.New()
_ = fake.Send(context.Background(), mail.Message{
	From:    &mail.Recipient{Email: "no-reply@example.com"},
	To:      []mail.Recipient{{Email: "alice@example.com"}},
	Subject: "Welcome",
	Text:    "hello world",
})
fmt.Println(fake.SentCount())
// 1
```

#### <a id="mailfake-driver-sentcount"></a>mailfake.Driver.SentCount

SentCount reports the number of recorded messages.

```go
fake := mailfake.New()
_ = fake.Send(context.Background(), mail.Message{
	From:    &mail.Recipient{Email: "no-reply@example.com"},
	To:      []mail.Recipient{{Email: "alice@example.com"}},
	Subject: "Welcome",
	Text:    "hello world",
})
fmt.Println(fake.SentCount())
// 1
```

#### <a id="mailfake-driver-seterror"></a>mailfake.Driver.SetError

SetError configures the error returned by future sends.

```go
fake := mailfake.New()
fake.SetError(errors.New("boom"))
err := fake.Send(context.Background(), mail.Message{
	From:    &mail.Recipient{Email: "no-reply@example.com"},
	To:      []mail.Recipient{{Email: "alice@example.com"}},
	Subject: "Welcome",
	Text:    "hello world",
})
fmt.Println(err != nil)
// true
```

#### <a id="mailfake-new"></a>mailfake.New

New creates an in-memory fake mail driver for tests.

```go
fake := mailfake.New()
_ = mail.New(fake).Send(context.Background(), mail.Message{
	From:    &mail.Recipient{Email: "no-reply@example.com"},
	To:      []mail.Recipient{{Email: "alice@example.com"}},
	Subject: "Welcome",
	Text:    "hello world",
})
fmt.Println(fake.SentCount())
// 1
```
<!-- api:embed:end -->

## Docs Tooling

- `go run ./docs/examplegen/main.go`
- `go run ./docs/readme/main.go`
- `go run ./docs/readme/testcounts/main.go`
- `./docs/watcher.sh`
