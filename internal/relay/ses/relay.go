package relay

import (
	"context"
	"net"
	"regexp"

	"github.com/aws/aws-sdk-go-v2/service/ses"
	sestypes "github.com/aws/aws-sdk-go-v2/service/ses/types"
	"github.com/blueimp/aws-smtp-relay/internal/relay"
)

type SESEmailClient interface {
	SendRawEmail(context.Context, *ses.SendRawEmailInput, ...func(*ses.Options)) (*ses.SendRawEmailOutput, error)
}

// Client implements the Relay interface.
type Client struct {
	sesClient       SESEmailClient
	setName         *string
	allowFromRegExp *regexp.Regexp
	denyToRegExp    *regexp.Regexp
}

// Send uses the client SESAPI to send email data
func (c Client) Send(
	origin net.Addr,
	from string,
	to []string,
	data []byte,
) error {
	allowedRecipients, deniedRecipients, err := relay.FilterAddresses(
		from,
		to,
		c.allowFromRegExp,
		c.denyToRegExp,
	)
	if err != nil {
		relay.Log(origin, from, deniedRecipients, err)
	}
	if len(allowedRecipients) > 0 {
		_, err := c.sesClient.SendRawEmail(context.Background(), &ses.SendRawEmailInput{
			RawMessage:           &sestypes.RawMessage{Data: data},
			ConfigurationSetName: c.setName,
			Destinations:         allowedRecipients,
			FromArn:              new(string),
			ReturnPathArn:        new(string),
			Source:               &from,
			SourceArn:            new(string),
			Tags:                 []sestypes.MessageTag{},
		})
		relay.Log(origin, from, allowedRecipients, err)
		if err != nil {
			return err
		}
	}
	return err
}

// New creates a new client with a session.
func New(
	configurationSetName *string,
	allowFromRegExp *regexp.Regexp,
	denyToRegExp *regexp.Regexp,
) Client {
	return Client{
		sesClient:       ses.New(ses.Options{}),
		setName:         configurationSetName,
		allowFromRegExp: allowFromRegExp,
		denyToRegExp:    denyToRegExp,
	}
}
