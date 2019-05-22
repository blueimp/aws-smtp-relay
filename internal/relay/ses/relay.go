package relay

import (
	"net"
	"time"

	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/aws/aws-sdk-go/service/ses/sesiface"
	"github.com/blueimp/aws-smtp-relay/internal/relay"
)

// Send uses the given SESAPI to send email data
func Send(
	sesAPI sesiface.SESAPI,
	origin net.Addr,
	from *string,
	to *[]string,
	data *[]byte,
	setName *string,
) {
	var err error
	req := &relay.Request{
		Time: time.Now().UTC(),
		Addr: origin,
		From: *from,
		To:   *to,
		Err:  &err,
	}
	defer req.Log()
	destinations := []*string{}
	for k := range *to {
		destinations = append(destinations, &(*to)[k])
	}
	_, err = sesAPI.SendRawEmail(&ses.SendRawEmailInput{
		ConfigurationSetName: setName,
		Source:               from,
		Destinations:         destinations,
		RawMessage:           &ses.RawMessage{Data: *data},
	})
}
