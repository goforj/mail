package mailses

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	sestypes "github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"github.com/goforj/mail"
)

type stubClient struct {
	input *sesv2.SendEmailInput
	err   error
}

func (s *stubClient) SendEmail(_ context.Context, params *sesv2.SendEmailInput, _ ...func(*sesv2.Options)) (*sesv2.SendEmailOutput, error) {
	s.input = params
	if s.err != nil {
		return nil, s.err
	}
	return &sesv2.SendEmailOutput{MessageId: aws.String("ses_123")}, nil
}

func TestNewRequiresRegion(t *testing.T) {
	_, err := New(Config{})
	if err == nil || err.Error() != "mailses: region is required" {
		t.Fatalf("new error = %v, want region error", err)
	}
}

func TestDriverSendBuildsRawEmailAndTags(t *testing.T) {
	client := &stubClient{}
	driver := newWithClient(client, Config{ConfigurationSetName: "transactional"})

	err := driver.Send(context.Background(), mail.Message{
		From:    &mail.Recipient{Email: "no-reply@example.com", Name: "Example"},
		To:      []mail.Recipient{{Email: "alice@example.com", Name: "Alice"}},
		Subject: "Welcome",
		Text:    "hello world",
		Tags:    []string{"welcome"},
		Metadata: map[string]string{
			"tenant_id": "tenant_123",
		},
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}

	if client.input == nil {
		t.Fatal("expected send input")
	}
	if client.input.ConfigurationSetName == nil || *client.input.ConfigurationSetName != "transactional" {
		t.Fatalf("configuration set = %#v", client.input.ConfigurationSetName)
	}
	if client.input.Content == nil || client.input.Content.Raw == nil || len(client.input.Content.Raw.Data) == 0 {
		t.Fatalf("raw content = %#v", client.input.Content)
	}
	if len(client.input.EmailTags) != 2 {
		t.Fatalf("email tags = %#v", client.input.EmailTags)
	}
}

func TestBuildTags(t *testing.T) {
	got := buildTags([]string{"welcome"}, map[string]string{"tenant_id": "tenant_123"})
	if len(got) != 2 {
		t.Fatalf("tag count = %d", len(got))
	}
	assertTag := func(tag sestypes.MessageTag, name, value string) {
		if tag.Name == nil || *tag.Name != name || tag.Value == nil || *tag.Value != value {
			t.Fatalf("tag = %#v, want %s=%s", tag, name, value)
		}
	}
	assertTag(got[0], "tenant_id", "tenant_123")
	assertTag(got[1], "tag_1", "welcome")
}
