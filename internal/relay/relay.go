package relay

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/aws/aws-sdk-go/service/ses/sesiface"
)

// Request object
type Request struct {
	Time  time.Time
	addr  net.Addr
	IP    string
	From  string
	To    []string
	Error string
	err   *error
}

// Log prints the Request data to Stdout/Stderr
func (req *Request) Log() {
	req.IP = req.addr.(*net.TCPAddr).IP.String()
	var out *os.File
	err := *req.err
	if err != nil {
		req.Error = err.Error()
		out = os.Stderr
	} else {
		out = os.Stdout
	}
	b, _ := json.Marshal(req)
	fmt.Fprintln(out, string(b))
}

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
	req := &Request{
		Time: time.Now().UTC(),
		addr: origin,
		From: *from,
		To:   *to,
		err:  &err,
	}
	defer req.Log()
	destinations := []*string{}
	for _, v := range *to {
		destinations = append(destinations, &v)
	}
	_, err = sesAPI.SendRawEmail(&ses.SendRawEmailInput{
		ConfigurationSetName: setName,
		Source:               from,
		Destinations:         destinations,
		RawMessage:           &ses.RawMessage{Data: *data},
	})
}
