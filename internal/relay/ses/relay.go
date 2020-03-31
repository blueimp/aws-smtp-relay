package relay

import (
	"errors"
	"net"
	"os"
	"regexp"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/aws/aws-sdk-go/service/ses/sesiface"
	"github.com/blueimp/aws-smtp-relay/internal/request"
)

// Client implements the Relay interface.
type Client struct {
	sesAPI  sesiface.SESAPI
	setName *string
}

// Send uses the client SESAPI to send email data
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
	denyRegex := os.Getenv("EMAIL_DENY_REGEX")
	if denyRegex != "" {
		res, err := regexp.MatchString(denyRegex, from)
		if res && err == nil {
			request.Log(origin, from, to, errors.New("requested email matches regex for exclusion"))
			return
		}
	}
	_, err := c.sesAPI.SendRawEmail(&ses.SendRawEmailInput{
		ConfigurationSetName: c.setName,
		Source:               &from,
		Destinations:         destinations,
		RawMessage:           &ses.RawMessage{Data: data},
	})
	request.Log(origin, from, to, err)
}

// New creates a new client with a session.
func New(configurationSetName *string) Client {
	return Client{
		sesAPI:  ses.New(session.Must(session.NewSession())),
		setName: configurationSetName,
	}
}
