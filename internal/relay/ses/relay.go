package ses

import (
	"context"
	"io"
	"net"
	"regexp"

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
	allowFromRegExp *regexp.Regexp
	denyToRegExp    *regexp.Regexp
	maxMessageSize  uint
}

func (c Client) Annotate(_clt relay.Client) relay.Client {
	clt := _clt.(*Client)
	pclt := c.SesClient
	if clt.SesClient != nil {
		pclt = clt.SesClient
	}
	return &Client{
		SesClient:       pclt,
		setName:         c.setName,
		allowFromRegExp: c.allowFromRegExp,
		denyToRegExp:    c.denyToRegExp,
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
		c.allowFromRegExp,
		c.denyToRegExp,
	)
	if err != nil {
		internal.Log(origin, from, deniedRecipients, err)
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
		internal.Log(origin, from, allowedRecipients, err)
	}
	return err
}

// New creates a new client with a session.
func New(
	configurationSetName *string,
	allowFromRegExp *regexp.Regexp,
	denyToRegExp *regexp.Regexp,
	maxMessageSize uint,
) Client {
	return Client{
		maxMessageSize:  maxMessageSize,
		SesClient:       ses.New(ses.Options{}),
		setName:         configurationSetName,
		allowFromRegExp: allowFromRegExp,
		denyToRegExp:    denyToRegExp,
	}
}
