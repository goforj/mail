<p align="center">
  <strong>mail</strong>
</p>

<p align="center">
  Fluent email composition and pluggable delivery for GoForj packages and apps.
</p>

<p align="center">
<!-- test-count:embed:start -->
<img src="https://img.shields.io/badge/unit_tests-23-brightgreen" alt="Unit tests (executed count)">
<img src="https://img.shields.io/badge/integration_tests-0-blue" alt="Integration tests (executed count)">
<!-- test-count:embed:end -->
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

## API

<!-- api:embed:start -->
## API Index

| Group | Functions |
|------:|:-----------|
| **Composition** | [Mailer.Message](#mailer-message) [MessageBuilder.Bcc](#messagebuilder-bcc) [MessageBuilder.Cc](#messagebuilder-cc) [MessageBuilder.From](#messagebuilder-from) [MessageBuilder.Message](#messagebuilder-message) [MessageBuilder.ReplyTo](#messagebuilder-replyto) [MessageBuilder.To](#messagebuilder-to) |
| **Construction** | [New](#new) |
| **Content** | [MessageBuilder.HTML](#messagebuilder-html) [MessageBuilder.Header](#messagebuilder-header) [MessageBuilder.Metadata](#messagebuilder-metadata) [MessageBuilder.Subject](#messagebuilder-subject) [MessageBuilder.Tag](#messagebuilder-tag) [MessageBuilder.Text](#messagebuilder-text) |
| **Defaults** | [WithDefaultFrom](#withdefaultfrom) [WithDefaultHeader](#withdefaultheader) [WithDefaultMetadata](#withdefaultmetadata) [WithDefaultReplyTo](#withdefaultreplyto) [WithDefaultTag](#withdefaulttag) |
| **Delivery** | [Mailer.Send](#mailer-send) [MessageBuilder.Build](#messagebuilder-build) [MessageBuilder.Send](#messagebuilder-send) |
| **Logging** | [maillog.Driver.Send](#maillog-driver-send) [maillog.New](#maillog-new) [maillog.WithBodies](#maillog-withbodies) [maillog.WithNow](#maillog-withnow) |
| **Mailgun** | [mailmailgun.Driver.Send](#mailmailgun-driver-send) [mailmailgun.New](#mailmailgun-new) |
| **Message Model** | [Message.Clone](#message-clone) [Message.Validate](#message-validate) |
| **Postmark** | [mailpostmark.Driver.Send](#mailpostmark-driver-send) [mailpostmark.New](#mailpostmark-new) |
| **Resend** | [mailresend.Driver.Send](#mailresend-driver-send) [mailresend.New](#mailresend-new) |
| **SES** | [mailses.Driver.Send](#mailses-driver-send) [mailses.New](#mailses-new) |
| **SMTP** | [mailsmtp.Driver.Send](#mailsmtp-driver-send) [mailsmtp.New](#mailsmtp-new) [mailsmtp.Render](#mailsmtp-render) |
| **Testing** | [mailfake.Driver.Last](#mailfake-driver-last) [mailfake.Driver.Messages](#mailfake-driver-messages) [mailfake.Driver.Reset](#mailfake-driver-reset) [mailfake.Driver.Send](#mailfake-driver-send) [mailfake.Driver.SentCount](#mailfake-driver-sentcount) [mailfake.Driver.SetError](#mailfake-driver-seterror) [mailfake.New](#mailfake-new) |


## API Reference

_Generated from public API comments and examples._

### Composition

#### <a id="mailer-message"></a>`Mailer.Message`

Message starts a new fluent message builder bound to this mailer.

compose and send one message:

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

#### <a id="messagebuilder-bcc"></a>`MessageBuilder.Bcc`

Bcc appends one blind-carbon-copy recipient.

add a bcc recipient:

```go
msg, _ := mail.New(mailfake.New()).Message().
	To("alice@example.com", "Alice").
	Bcc("audit@example.com", "Audit").
	Subject("Welcome").
	Text("hello world").
	Build()
fmt.Println(msg.Bcc[0].Email)
// audit@example.com
```

#### <a id="messagebuilder-cc"></a>`MessageBuilder.Cc`

Cc appends one carbon-copy recipient.

add a cc recipient:

```go
msg, _ := mail.New(mailfake.New()).Message().
	To("alice@example.com", "Alice").
	Cc("manager@example.com", "Manager").
	Subject("Welcome").
	Text("hello world").
	Build()
fmt.Println(msg.Cc[0].Email)
// manager@example.com
```

#### <a id="messagebuilder-from"></a>`MessageBuilder.From`

From sets the from recipient.

override the from recipient for one message:

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

#### <a id="messagebuilder-message"></a>`MessageBuilder.Message`

Message returns the currently composed message without applying mailer defaults.

inspect the in-progress message:

```go
msg := mail.New(mailfake.New()).Message().
	To("alice@example.com", "Alice").
	Subject("Welcome").
	Text("hello world").
	Message()
fmt.Println(msg.Subject)
// Welcome
```

#### <a id="messagebuilder-replyto"></a>`MessageBuilder.ReplyTo`

ReplyTo appends one reply-to recipient.

add a reply-to recipient:

```go
msg, _ := mail.New(mailfake.New()).Message().
	To("alice@example.com", "Alice").
	ReplyTo("support@example.com", "Support").
	Subject("Welcome").
	Text("hello world").
	Build()
fmt.Println(msg.ReplyTo[0].Email)
// support@example.com
```

#### <a id="messagebuilder-to"></a>`MessageBuilder.To`

To appends one primary recipient.

add one primary recipient:

```go
msg, _ := mail.New(mailfake.New()).Message().
	To("alice@example.com", "Alice").
	Subject("Welcome").
	Text("hello world").
	Build()
fmt.Println(len(msg.To))
// 1
```

### Construction

#### <a id="new"></a>`New`

New creates a Mailer backed by the provided driver.

create a mailer with a default from address:

```go
fake := mailfake.New()
mailer := mail.New(fake, mail.WithDefaultFrom("no-reply@example.com", "Example"))
fmt.Println(mailer != nil)
// true
```

### Content

#### <a id="messagebuilder-html"></a>`MessageBuilder.HTML`

HTML sets the HTML body.

set an html body:

```go
msg := mail.New(mailfake.New()).Message().
	To("alice@example.com", "Alice").
	Subject("Welcome").
	HTML("<p>hello world</p>").
	Message()
fmt.Println(msg.HTML)
// <p>hello world</p>
```

#### <a id="messagebuilder-header"></a>`MessageBuilder.Header`

Header sets or replaces one message header.

attach headers, tags, and metadata:

```go
message, _ := mail.New(mailfake.New()).Message().
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

#### <a id="messagebuilder-metadata"></a>`MessageBuilder.Metadata`

Metadata sets one provider-facing metadata key/value pair.

add provider-facing metadata:

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

#### <a id="messagebuilder-subject"></a>`MessageBuilder.Subject`

Subject sets the message subject.

set the subject line:

```go
msg := mail.New(mailfake.New()).Message().
	To("alice@example.com", "Alice").
	Subject("Welcome").
	Text("hello world").
	Message()
fmt.Println(msg.Subject)
// Welcome
```

#### <a id="messagebuilder-tag"></a>`MessageBuilder.Tag`

Tag appends one provider-facing message tag.

add a provider-facing tag:

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

#### <a id="messagebuilder-text"></a>`MessageBuilder.Text`

Text sets the plain text body.

set a text body:

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

#### <a id="withdefaultfrom"></a>`WithDefaultFrom`

WithDefaultFrom configures the default from recipient applied when a message omits one.

apply a default from address:

```go
mailer := mail.New(
	mailfake.New(),
	mail.WithDefaultFrom("no-reply@example.com", "Example"),
)
fmt.Println(mailer != nil)
// true
```

#### <a id="withdefaultheader"></a>`WithDefaultHeader`

WithDefaultHeader configures a header applied when a message omits that header key.

apply a default header:

```go
msg, _ := mail.New(
	mailfake.New(),
	mail.WithDefaultHeader("X-App", "goforj"),
).Message().
	To("alice@example.com", "Alice").
	Subject("Welcome").
	Text("hello world").
	Build()
fmt.Println(msg.Headers["X-App"])
// goforj
```

#### <a id="withdefaultmetadata"></a>`WithDefaultMetadata`

WithDefaultMetadata configures metadata applied when a message omits that metadata key.

apply default metadata:

```go
msg, _ := mail.New(
	mailfake.New(),
	mail.WithDefaultMetadata("tenant_id", "tenant_123"),
).Message().
	To("alice@example.com", "Alice").
	Subject("Welcome").
	Text("hello world").
	Build()
fmt.Println(msg.Metadata["tenant_id"])
// tenant_123
```

#### <a id="withdefaultreplyto"></a>`WithDefaultReplyTo`

WithDefaultReplyTo configures the default reply-to recipients applied when a message omits them.

apply a default reply-to recipient:

```go
mailer := mail.New(
	mailfake.New(),
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

#### <a id="withdefaulttag"></a>`WithDefaultTag`

WithDefaultTag configures a tag prepended to every message sent by the mailer.

apply a default tag:

```go
msg, _ := mail.New(
	mailfake.New(),
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

#### <a id="mailer-send"></a>`Mailer.Send`

Send validates the message, applies defaults, and delegates delivery to the driver.

send a prebuilt message:

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

#### <a id="messagebuilder-build"></a>`MessageBuilder.Build`

Build applies defaults, validates, and returns the composed message without sending it.

build without sending:

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

#### <a id="messagebuilder-send"></a>`MessageBuilder.Send`

Send delegates the composed message to the bound mailer.

send through the bound mailer:

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

#### <a id="maillog-driver-send"></a>`maillog.Driver.Send`

Send writes one JSON log record for the message.

write one log entry directly:

```go
var out bytes.Buffer
_ = maillog.New(&out).Send(context.Background(), mail.Message{
	To:      []mail.Recipient{{Email: "alice@example.com"}},
	Subject: "Welcome",
	Text:    "hello world",
})
fmt.Println(strings.Contains(out.String(), "\"subject\":\"Welcome\""))
// true
```

#### <a id="maillog-new"></a>`maillog.New`

New creates a log mail driver that writes one JSON record per sent message.

log one message to a buffer:

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

#### <a id="maillog-withbodies"></a>`maillog.WithBodies`

WithBodies controls whether HTML and text bodies are included in log output.

include bodies in log output:

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

#### <a id="maillog-withnow"></a>`maillog.WithNow`

WithNow overrides the timestamp source used by log entries.

control the logged timestamp:

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

#### <a id="mailmailgun-driver-send"></a>`mailmailgun.Driver.Send`

Send validates and transmits one message through Mailgun.

send one message through Mailgun:

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

#### <a id="mailmailgun-new"></a>`mailmailgun.New`

New creates a Mailgun mail driver from the given config.

configure a Mailgun mail driver:

```go
driver, _ := mailmailgun.New(mailmailgun.Config{
	Domain: "mg.example.com",
	APIKey: "key-test",
})
fmt.Println(driver != nil)
// true
```

### Message Model

#### <a id="message-clone"></a>`Message.Clone`

Clone returns a copy of the message safe for reuse in tests and builders.

clone before mutating:

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

#### <a id="message-validate"></a>`Message.Validate`

Validate checks that the message has valid recipients, subject, body, and headers.

validate a complete message:

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

#### <a id="mailpostmark-driver-send"></a>`mailpostmark.Driver.Send`

Send validates and transmits one message through Postmark.

send one message through Postmark:

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

#### <a id="mailpostmark-new"></a>`mailpostmark.New`

New creates a Postmark mail driver from the given config.

configure a Postmark mail driver:

```go
driver, _ := mailpostmark.New(mailpostmark.Config{
	ServerToken: "pm_test_token",
})
fmt.Println(driver != nil)
// true
```

### Resend

#### <a id="mailresend-driver-send"></a>`mailresend.Driver.Send`

Send validates and transmits one message through Resend.

send one message through Resend:

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

#### <a id="mailresend-new"></a>`mailresend.New`

New creates a Resend mail driver from the given config.

configure a Resend mail driver:

```go
driver, _ := mailresend.New(mailresend.Config{
	APIKey: "re_test_key",
})
fmt.Println(driver != nil)
// true
```

### SES

#### <a id="mailses-driver-send"></a>`mailses.Driver.Send`

Send validates and transmits one message through Amazon SES.

send one message through Amazon SES:

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

#### <a id="mailses-new"></a>`mailses.New`

New creates an Amazon SES mail driver from the given config.

configure an Amazon SES mail driver:

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

#### <a id="mailsmtp-driver-send"></a>`mailsmtp.Driver.Send`

Send validates and transmits one message over SMTP.

send one message over SMTP:

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

#### <a id="mailsmtp-new"></a>`mailsmtp.New`

New creates an SMTP mail driver from the given config.

configure an SMTP mail driver:

```go
driver, _ := mailsmtp.New(mailsmtp.Config{
	Host: "smtp.example.com",
	Port: 587,
})
fmt.Println(driver != nil)
// true
```

#### <a id="mailsmtp-render"></a>`mailsmtp.Render`

Render turns one message into an RFC 822 style SMTP payload.

render a text message:

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

### Testing

#### <a id="mailfake-driver-last"></a>`mailfake.Driver.Last`

Last returns the last recorded message when one exists.

read the last recorded subject:

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

#### <a id="mailfake-driver-messages"></a>`mailfake.Driver.Messages`

Messages returns a copy of every recorded message.

inspect recorded messages:

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

#### <a id="mailfake-driver-reset"></a>`mailfake.Driver.Reset`

Reset clears recorded messages and any configured send error.

clear recorded state:

```go
fake := mailfake.New()
_ = fake.Send(context.Background(), mail.Message{
	To:      []mail.Recipient{{Email: "alice@example.com"}},
	Subject: "Welcome",
	Text:    "hello world",
})
fake.Reset()
fmt.Println(fake.SentCount())
// 0
```

#### <a id="mailfake-driver-send"></a>`mailfake.Driver.Send`

Send records the message and returns the configured error when set.

record a sent message directly:

```go
fake := mailfake.New()
_ = fake.Send(context.Background(), mail.Message{
	To:      []mail.Recipient{{Email: "alice@example.com"}},
	Subject: "Welcome",
	Text:    "hello world",
})
fmt.Println(fake.SentCount())
// 1
```

#### <a id="mailfake-driver-sentcount"></a>`mailfake.Driver.SentCount`

SentCount reports the number of recorded messages.

count recorded messages:

```go
fake := mailfake.New()
_ = fake.Send(context.Background(), mail.Message{
	To:      []mail.Recipient{{Email: "alice@example.com"}},
	Subject: "Welcome",
	Text:    "hello world",
})
fmt.Println(fake.SentCount())
// 1
```

#### <a id="mailfake-driver-seterror"></a>`mailfake.Driver.SetError`

SetError configures the error returned by future sends.

force sends to fail:

```go
fake := mailfake.New()
fake.SetError(errors.New("boom"))
err := fake.Send(context.Background(), mail.Message{
	To:      []mail.Recipient{{Email: "alice@example.com"}},
	Subject: "Welcome",
	Text:    "hello world",
})
fmt.Println(err != nil)
// true
```

#### <a id="mailfake-new"></a>`mailfake.New`

New creates an in-memory fake mail driver for tests.

record one sent message:

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
