package mailses

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"github.com/goforj/mail"
	"github.com/goforj/mail/internal/mailhttp"
	"github.com/goforj/mail/mailsmtp"
)

// Config configures Amazon SES delivery.
// @group SES
type Config struct {
	Region               string
	AccessKeyID          string
	SecretAccessKey      string
	SessionToken         string
	Endpoint             string
	ConfigurationSetName string
	HTTPClient           *http.Client
}

type sendAPI interface {
	SendEmail(ctx context.Context, params *sesv2.SendEmailInput, optFns ...func(*sesv2.Options)) (*sesv2.SendEmailOutput, error)
}

// Driver sends messages through Amazon SES.
// @group SES
type Driver struct {
	client               sendAPI
	configurationSetName string
}

// New creates an Amazon SES mail driver from the given config.
// @group SES
//
// Example: configure an Amazon SES mail driver
//
//	driver, _ := mailses.New(mailses.Config{
//		Region:          "us-east-1",
//		AccessKeyID:     "test",
//		SecretAccessKey: "test",
//	})
//	fmt.Println(driver != nil)
//	// true
func New(settings Config) (*Driver, error) {
	region := strings.TrimSpace(settings.Region)
	if region == "" {
		return nil, fmt.Errorf("mailses: region is required")
	}

	accessKeyID := strings.TrimSpace(settings.AccessKeyID)
	secretAccessKey := strings.TrimSpace(settings.SecretAccessKey)
	if (accessKeyID == "") != (secretAccessKey == "") {
		return nil, fmt.Errorf("mailses: access key id and secret access key must be provided together")
	}
	if accessKeyID == "" && strings.TrimSpace(settings.SessionToken) != "" {
		return nil, fmt.Errorf("mailses: session token requires static credentials")
	}
	endpoint := strings.TrimSpace(settings.Endpoint)
	if endpoint != "" {
		var err error
		endpoint, err = mailhttp.Endpoint("mailses", endpoint, "")
		if err != nil {
			return nil, err
		}
	}

	loadOptions := []func(*config.LoadOptions) error{
		config.WithRegion(region),
	}
	if settings.HTTPClient != nil {
		loadOptions = append(loadOptions, config.WithHTTPClient(settings.HTTPClient))
	}
	if accessKeyID != "" {
		loadOptions = append(loadOptions, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, settings.SessionToken),
		))
	}

	cfg, err := config.LoadDefaultConfig(context.Background(), loadOptions...)
	if err != nil {
		return nil, err
	}

	client := sesv2.NewFromConfig(cfg, func(o *sesv2.Options) {
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
		}
	})

	return newWithClient(client, settings), nil
}

// newWithClient isolates SDK construction from the provider contract exercised by tests.
func newWithClient(client sendAPI, settings Config) *Driver {
	return &Driver{
		client:               client,
		configurationSetName: strings.TrimSpace(settings.ConfigurationSetName),
	}
}

// Send validates and transmits one message through Amazon SES.
// @group SES
//
// Example: send one message through Amazon SES
//
//	driver, _ := mailses.New(mailses.Config{
//		Region:          "us-east-1",
//		AccessKeyID:     "test",
//		SecretAccessKey: "test",
//		Endpoint:        "http://127.0.0.1:1",
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

	raw, err := mailsmtp.Render(message)
	if err != nil {
		return err
	}

	input := &sesv2.SendEmailInput{
		Content: &types.EmailContent{
			Raw: &types.RawMessage{
				Data: raw,
			},
		},
		EmailTags: buildTags(message.Tags, message.Metadata),
	}
	if d.configurationSetName != "" {
		input.ConfigurationSetName = aws.String(d.configurationSetName)
	}

	_, err = d.client.SendEmail(ctx, input)
	return err
}

// buildTags produces deterministic, uniquely named SES message tags.
func buildTags(tags []string, metadata map[string]string) []types.MessageTag {
	out := make([]types.MessageTag, 0, len(tags)+len(metadata))
	usedNames := make(map[string]struct{}, len(tags)+len(metadata))
	keys := make([]string, 0, len(metadata))
	for key := range metadata {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := metadata[key]
		name := sanitizeTagToken(key, 256)
		tagValue := sanitizeTagToken(value, 256)
		if name == "" || tagValue == "" {
			continue
		}
		if _, exists := usedNames[name]; exists {
			continue
		}
		usedNames[name] = struct{}{}
		out = append(out, types.MessageTag{Name: aws.String(name), Value: aws.String(tagValue)})
	}
	nextTagIndex := 1
	for _, value := range tags {
		tagValue := sanitizeTagToken(value, 256)
		if tagValue == "" {
			continue
		}
		name := ""
		for {
			name = "tag_" + strconv.Itoa(nextTagIndex)
			nextTagIndex++
			if _, exists := usedNames[name]; !exists {
				break
			}
		}
		usedNames[name] = struct{}{}
		out = append(out, types.MessageTag{
			Name:  aws.String(name),
			Value: aws.String(tagValue),
		})
	}
	return out
}

// sanitizeTagToken limits values to SES's portable ASCII tag subset.
func sanitizeTagToken(value string, max int) string {
	if max <= 0 {
		return ""
	}
	var builder strings.Builder
	builder.Grow(len(value))
	for _, r := range strings.TrimSpace(value) {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			builder.WriteRune(r)
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		case r == '_' || r == '-':
			builder.WriteRune(r)
		default:
			builder.WriteRune('_')
		}
		if builder.Len() >= max {
			break
		}
	}
	return strings.Trim(builder.String(), "_-")
}
