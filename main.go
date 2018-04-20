package main // import "github.com/blueimp/aws-smtp-relay"

import (
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/blueimp/aws-smtp-relay/internal/relay"
	"github.com/blueimp/aws-smtp-relay/internal/smtpd"
)

var (
	client   = ses.New(session.Must(session.NewSession()))
	addr     = flag.String("a", ":1025", "TCP listen address")
	name     = flag.String("n", "AWS SMTP Relay", "SMTP service name")
	host     = flag.String("h", "", "Server hostname")
	certFile = flag.String("c", "", "TLS cert file")
	keyFile  = flag.String("k", "", "TLS key file")
	startTLS = flag.Bool("s", false, "Require TLS via STARTTLS extension")
	onlyTLS  = flag.Bool("t", false, "Listen for incoming TLS connections only")
)

func handler(origin net.Addr, from string, to []string, data []byte) {
	relay.Send(client, origin, &from, &to, &data)
}

func server() (srv *smtpd.Server, err error) {
	flag.Parse()
	srv = &smtpd.Server{
		Addr:        *addr,
		Handler:     handler,
		Appname:     *name,
		Hostname:    *host,
		TLSRequired: *startTLS,
		TLSListener: *onlyTLS,
	}
	if *certFile != "" && *keyFile != "" {
		keyPass := os.Getenv("TLS_KEY_PASS")
		if keyPass != "" {
			err = srv.ConfigureTLSWithPassphrase(*certFile, *keyFile, keyPass)
		} else {
			err = srv.ConfigureTLS(*certFile, *keyFile)
		}
	}
	return
}

func main() {
	srv, err := server()
	if err == nil {
		err = srv.ListenAndServe()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
