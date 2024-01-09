package ses

import (
	"context"
	"io"
	"net"
	"regexp"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	sestypes "github.com/aws/aws-sdk-go-v2/service/ses/types"
	"github.com/blueimp/aws-smtp-relay/internal"
	"github.com/blueimp/aws-smtp-relay/internal/relay"
	"github.com/blueimp/aws-smtp-relay/internal/relay/filter"
)

type SESEmailClient interface {
	SendRawEmail(context.Context, *ses.SendRawEmailInput, ...func(*ses.Options)) (*ses.SendRawEmailOutput, error)
}

// Client implements the Relay interface.
type Client struct {
	SesClient       SESEmailClient
	setName         *string
	AllowFromRegExp *regexp.Regexp
	DenyToRegExp    *regexp.Regexp
	maxMessageSize  uint
}

func (c Client) FilterFrom(from string) error {
	_, _, err := filter.FilterAddresses(from, nil, c.AllowFromRegExp, c.DenyToRegExp)
	return err
}

func (c Client) FilterTo(from string, to []string) ([]string, []string, error) {
	return filter.FilterAddresses(from, to, c.AllowFromRegExp, c.DenyToRegExp)
}

func (c Client) Annotate(_clt relay.Client) relay.Client {
	clt := _clt.(*Client)
	pclt := c.SesClient
	if clt.SesClient != nil {
		pclt = clt.SesClient
	}
	allowRe := c.AllowFromRegExp
	if clt.AllowFromRegExp != nil {
		allowRe = clt.AllowFromRegExp
	}
	denyRe := c.DenyToRegExp
	if clt.DenyToRegExp != nil {
		denyRe = clt.DenyToRegExp
	}
	return &Client{
		SesClient:       pclt,
		setName:         c.setName,
		AllowFromRegExp: allowRe,
		DenyToRegExp:    denyRe,
		maxMessageSize:  c.maxMessageSize,
	}
}

// Send uses the client SESAPI to send email data
func (c Client) Send(
	origin net.Addr,
	from string,
	to []string,
	dr io.Reader,
) error {
	allowedRecipients, deniedRecipients, err := filter.FilterAddresses(
		from,
		to,
		c.AllowFromRegExp,
		c.DenyToRegExp,
	)
	if err != nil {
		internal.LogEmail(origin, from, deniedRecipients, err)
	}

	if len(allowedRecipients) > 0 {
		data, sendErr := relay.ConsumeToBytes(dr, c.maxMessageSize)
		if sendErr != nil {
			return sendErr
		}
		_, sendErr = c.SesClient.SendRawEmail(context.Background(), &ses.SendRawEmailInput{
			RawMessage:           &sestypes.RawMessage{Data: data},
			ConfigurationSetName: c.setName,
			Destinations:         allowedRecipients,
			FromArn:              new(string),
			ReturnPathArn:        new(string),
			Source:               &from,
			SourceArn:            new(string),
			Tags:                 []sestypes.MessageTag{},
		})
		if sendErr != nil {
			err = sendErr
		}
		internal.LogEmail(origin, from, allowedRecipients, err)
	}
	return err
}

// New creates a new client with a session.
func New(
	configurationSetName *string,
	allowFromRegExp *regexp.Regexp,
	denyToRegExp *regexp.Regexp,
	maxMessageSize uint,
) (*Client, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, internal.LogError("AwsSesObserver.getSqsClient", "error loading aws config, %s", err.Error())
	}
	return &Client{
		maxMessageSize:  maxMessageSize,
		SesClient:       ses.NewFromConfig(cfg),
		setName:         configurationSetName,
		AllowFromRegExp: allowFromRegExp,
		DenyToRegExp:    denyToRegExp,
	}, nil
}
