package relay

import (
	"net"
	"regexp"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/aws/aws-sdk-go/service/ses/sesiface"
	"github.com/blueimp/aws-smtp-relay/internal/relay"
)

// Client implements the Relay interface.
type Client struct {
	sesAPI          sesiface.SESAPI
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
		relay.Log(origin, &from, deniedRecipients, err)
	}
	if len(allowedRecipients) > 0 {
		_, err := c.sesAPI.SendRawEmail(&ses.SendRawEmailInput{
			ConfigurationSetName: c.setName,
			Source:               &from,
			Destinations:         allowedRecipients,
			RawMessage:           &ses.RawMessage{Data: data},
		})
		relay.Log(origin, &from, allowedRecipients, err)
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
	roleArn *string,
) Client {
	return Client{
		sesAPI:          ses.New(AssumeRoleAWS(*roleArn)),
		setName:         configurationSetName,
		allowFromRegExp: allowFromRegExp,
		denyToRegExp:    denyToRegExp,
	}
}

func AssumeRoleAWS(roleArn string) *session.Session {
	var creds *credentials.Credentials
	sess := session.Must(session.NewSession())
	if regexp.MustCompile(`^arn:aws:iam::(.+):role/([^/]+)(/.+)?$`).MatchString(roleArn) {
		creds = stscreds.NewCredentials(sess, roleArn)
		return session.Must(session.NewSession(&aws.Config{Credentials: creds}))
	} else {
		return sess
	}
}
