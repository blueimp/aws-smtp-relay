package relay

import (
	"net"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/pinpointemail"
	"github.com/aws/aws-sdk-go/service/pinpointemail/pinpointemailiface"
	"github.com/blueimp/aws-smtp-relay/internal/relay"
)

// Client implements the Relay interface.
type Client struct {
	pinpointAPI pinpointemailiface.PinpointEmailAPI
	setName     *string
}

// Send uses the given Pinpoint API to send email data
func (c Client) Send(
	origin net.Addr,
	from string,
	to []string,
	data []byte,
) {
	destinations := []*string{}
	for k := range to {
		destinations = append(destinations, &(to)[k])
	}
	_, err := c.pinpointAPI.SendEmail(&pinpointemail.SendEmailInput{
		ConfigurationSetName: c.setName,
		FromEmailAddress:     &from,
		Destination: &pinpointemail.Destination{
			ToAddresses: destinations,
		},
		Content: &pinpointemail.EmailContent{
			Raw: &pinpointemail.RawMessage{
				Data: data,
			},
		},
	})
	relay.Log(origin, from, to, err)
}

// New creates a new client with a session.
func New(configurationSetName *string) Client {
	return Client{
		pinpointAPI: pinpointemail.New(session.Must(session.NewSession())),
		setName:     configurationSetName,
	}
}
