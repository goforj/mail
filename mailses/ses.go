package mailses

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	sestypes "github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"github.com/goforj/mail"
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
func New(config Config) (*Driver, error) {
	region := strings.TrimSpace(config.Region)
	if region == "" {
		return nil, fmt.Errorf("mailses: region is required")
	}

	loadOptions := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(region),
	}
	if config.HTTPClient != nil {
		loadOptions = append(loadOptions, awsconfig.WithHTTPClient(config.HTTPClient))
	}
	if strings.TrimSpace(config.AccessKeyID) != "" || strings.TrimSpace(config.SecretAccessKey) != "" || strings.TrimSpace(config.SessionToken) != "" {
		loadOptions = append(loadOptions, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(config.AccessKeyID, config.SecretAccessKey, config.SessionToken),
		))
	}

	cfg, err := awsconfig.LoadDefaultConfig(context.Background(), loadOptions...)
	if err != nil {
		return nil, err
	}

	client := sesv2.NewFromConfig(cfg, func(o *sesv2.Options) {
		if endpoint := strings.TrimSpace(config.Endpoint); endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
		}
	})

	return newWithClient(client, config), nil
}

func newWithClient(client sendAPI, config Config) *Driver {
	return &Driver{
		client:               client,
		configurationSetName: strings.TrimSpace(config.ConfigurationSetName),
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
		Content: &sestypes.EmailContent{
			Raw: &sestypes.RawMessage{
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

func buildTags(tags []string, metadata map[string]string) []sestypes.MessageTag {
	out := make([]sestypes.MessageTag, 0, len(tags)+len(metadata))
	for key, value := range metadata {
		name := sanitizeTagToken(key, 256)
		tagValue := sanitizeTagToken(value, 256)
		if name == "" || tagValue == "" {
			continue
		}
		out = append(out, sestypes.MessageTag{Name: aws.String(name), Value: aws.String(tagValue)})
	}
	for i, value := range tags {
		tagValue := sanitizeTagToken(value, 256)
		if tagValue == "" {
			continue
		}
		out = append(out, sestypes.MessageTag{
			Name:  aws.String("tag_" + strconv.Itoa(i+1)),
			Value: aws.String(tagValue),
		})
	}
	return out
}

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
