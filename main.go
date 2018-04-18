package main // import "github.com/blueimp/aws-smtp-relay"

import (
	"flag"
	"net"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/blueimp/aws-smtp-relay/internal/relay"
	"github.com/blueimp/aws-smtp-relay/internal/smtpd"
)

var client = ses.New(session.Must(session.NewSession()))

var addr = flag.String("a", ":1025", "TCP listen address")
var name = flag.String("n", "AWS SMTP Relay", "SMTP service name")
var host = flag.String("h", "", "Server hostname")

func handler(origin net.Addr, from string, to []string, data []byte) {
	relay.Send(client, origin, &from, &to, &data)
}

func options() (string, smtpd.Handler, string, string) {
	flag.Parse()
	return *addr, handler, *name, *host
}

func main() {
	smtpd.ListenAndServe(options())
}
