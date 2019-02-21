package relay

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
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

// NetAddrToString returns a string representation for a given net.Addr
func NetAddrToString(addr net.Addr) string {
	ip := addr.(*net.TCPAddr).IP.String()
	if strings.IndexByte(ip, ':') >= 0 {
		ip = "[" + ip + "]"
	}
	return ip
}

// Log prints the Request data to Stdout/Stderr
func (req *Request) Log() {
	req.IP = NetAddrToString(req.addr)
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
		Source:       from,
		Destinations: destinations,
		RawMessage:   &ses.RawMessage{Data: *data},
	})
}
