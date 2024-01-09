package pinpoint

import (
	"context"
	"io"
	"net"
	"regexp"

	"github.com/aws/aws-sdk-go-v2/service/pinpointemail"
	pinpointemailtypes "github.com/aws/aws-sdk-go-v2/service/pinpointemail/types"
	"github.com/blueimp/aws-smtp-relay/internal"
	"github.com/blueimp/aws-smtp-relay/internal/relay"
	"github.com/blueimp/aws-smtp-relay/internal/relay/filter"
)

type PinpointEmailClient interface {
	SendEmail(context.Context, *pinpointemail.SendEmailInput, ...func(*pinpointemail.Options)) (*pinpointemail.SendEmailOutput, error)
}

// Client implements the Relay interface.
type Client struct {
	PinpointClient  PinpointEmailClient
	setName         *string
	AllowFromRegExp *regexp.Regexp
	DenyToRegExp    *regexp.Regexp
	maxMessageSize  uint
}

func (c Client) FilterFrom(from string) error {
	_, _, err := filter.FilterAddresses(from, nil, c.AllowFromRegExp, nil)
	return err
}

func (c Client) FilterTo(from string, to []string) ([]string, []string, error) {
	return filter.FilterAddresses(from, to, c.AllowFromRegExp, c.DenyToRegExp)
}

func (c Client) Annotate(rclt relay.Client) relay.Client {
	clt := rclt.(*Client)
	pclt := c.PinpointClient
	if clt.PinpointClient != nil {
		pclt = clt.PinpointClient
	}
	return &Client{
		PinpointClient:  pclt,
		setName:         c.setName,
		AllowFromRegExp: c.AllowFromRegExp,
		DenyToRegExp:    c.DenyToRegExp,
		maxMessageSize:  c.maxMessageSize,
	}
}

// Send uses the given Pinpoint API to send email data
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
		_, sendErr = c.PinpointClient.SendEmail(context.Background(), &pinpointemail.SendEmailInput{
			Content:                        &pinpointemailtypes.EmailContent{Raw: &pinpointemailtypes.RawMessage{Data: data}},
			Destination:                    &pinpointemailtypes.Destination{ToAddresses: allowedRecipients},
			ConfigurationSetName:           c.setName,
			EmailTags:                      []pinpointemailtypes.MessageTag{},
			FeedbackForwardingEmailAddress: new(string),
			FromEmailAddress:               &from,
			ReplyToAddresses:               to,
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
) Client {
	return Client{
		PinpointClient:  pinpointemail.New(pinpointemail.Options{}),
		setName:         configurationSetName,
		AllowFromRegExp: allowFromRegExp,
		DenyToRegExp:    denyToRegExp,
		maxMessageSize:  maxMessageSize,
	}
}
