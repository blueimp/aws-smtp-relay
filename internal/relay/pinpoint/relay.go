package relay

import (
	"net"
	"time"

	"github.com/aws/aws-sdk-go/service/pinpointemail"
	"github.com/aws/aws-sdk-go/service/pinpointemail/pinpointemailiface"
	"github.com/blueimp/aws-smtp-relay/internal/relay"
)

// Send uses the given Pinpoint API to send email data
func Send(
	pinpointAPI pinpointemailiface.PinpointEmailAPI,
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
	_, err = pinpointAPI.SendEmail(&pinpointemail.SendEmailInput{
		ConfigurationSetName: setName,
		FromEmailAddress: from,
		Destination: &pinpointemail.Destination{
			ToAddresses: destinations,
		},
		Content: &pinpointemail.EmailContent{
			Raw: &pinpointemail.RawMessage{
				Data: *data,
			},
		},
	})
}