package relay

import (
	"errors"
	"net"
	"os"
	"regexp"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/pinpointemail"
	"github.com/aws/aws-sdk-go/service/pinpointemail/pinpointemailiface"
	"github.com/blueimp/aws-smtp-relay/internal/request"
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
	denyRegex := os.Getenv("EMAIL_DENY_REGEX")
	destinations := []*string{}
	for k := range to {
		if denyRegex != "" {
			res, _ := regexp.MatchString(denyRegex, to[k])
			if !res {
				request.Log(origin, from, to, errors.New("message cannot be sent to destination "+to[k]+", - email matches regex for exclusion"))
			}
		} else {
			destinations = append(destinations, &(to)[k])
		}
	}
	if denyRegex != "" {
		res, err := regexp.MatchString(denyRegex, from)
		if res && err == nil {
			request.Log(origin, from, to, errors.New("message not sent, sender email ("+from+") matches regex for exclusion"))
			return
		}
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
	request.Log(origin, from, to, err)
}

// New creates a new client with a session.
func New(configurationSetName *string) Client {
	return Client{
		pinpointAPI: pinpointemail.New(session.Must(session.NewSession())),
		setName:     configurationSetName,
	}
}
