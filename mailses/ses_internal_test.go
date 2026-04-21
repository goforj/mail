package mailses

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	sestypes "github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"github.com/goforj/mail"
)

func TestDriverSendEarlyBranchesAndStubError(t *testing.T) {
	client := &stubClient{err: errors.New("boom")}
	driver := newWithClient(client, Config{})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := driver.Send(ctx, mail.Message{}); err == nil || err != context.Canceled {
		t.Fatalf("Send canceled error = %v, want context canceled", err)
	}

	if err := driver.Send(context.Background(), mail.Message{}); err == nil {
		t.Fatal("expected validation error")
	}

	err := driver.Send(context.Background(), mail.Message{
		From:    &mail.Recipient{Email: "no-reply@example.com"},
		To:      []mail.Recipient{{Email: "alice@example.com"}},
		Subject: "Welcome",
		Text:    "hello world",
	})
	if err == nil || err.Error() != "boom" {
		t.Fatalf("Send stub error = %v, want boom", err)
	}

	if client.input == nil || len(client.input.Content.Raw.Data) == 0 {
		t.Fatalf("client input = %#v", client.input)
	}
}

func TestBuildTagsSkipsInvalidValues(t *testing.T) {
	got := buildTags(
		[]string{"hello world", "___"},
		map[string]string{
			"tenant id": "tenant-123",
			"___":       "skip",
		},
	)
	if len(got) != 2 {
		t.Fatalf("tag count = %d, want 2", len(got))
	}
	assert := func(tag sestypes.MessageTag, name, value string) {
		if tag.Name == nil || *tag.Name != name || tag.Value == nil || *tag.Value != value {
			t.Fatalf("tag = %#v, want %s=%s", tag, name, value)
		}
	}
	assert(got[0], "tenant_id", "tenant-123")
	assert(got[1], "tag_1", "hello_world")

	if got := sanitizeTagToken("value", 0); got != "" {
		t.Fatalf("sanitizeTagToken(max<=0) = %q", got)
	}
}

func TestNewWithClientTrimsConfigurationSet(t *testing.T) {
	driver := newWithClient(&stubClient{}, Config{ConfigurationSetName: "  transactional  "})
	if driver.configurationSetName != "transactional" {
		t.Fatalf("configurationSetName = %q", driver.configurationSetName)
	}
}

func TestStubClientReturnsMessageID(t *testing.T) {
	client := &stubClient{}
	out, err := client.SendEmail(context.Background(), &sesv2.SendEmailInput{
		Content: &sestypes.EmailContent{Raw: &sestypes.RawMessage{Data: []byte("raw")}},
	})
	if err != nil {
		t.Fatalf("SendEmail() error = %v", err)
	}
	if out.MessageId == nil || *out.MessageId != "ses_123" {
		t.Fatalf("message id = %#v", out.MessageId)
	}
	if client.input == nil || string(client.input.Content.Raw.Data) != "raw" {
		t.Fatalf("client input = %#v", client.input)
	}
	_ = aws.String("keep import stable")
}
