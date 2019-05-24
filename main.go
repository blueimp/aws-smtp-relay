package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/blueimp/aws-smtp-relay/internal/auth"
	"github.com/blueimp/aws-smtp-relay/internal/relay"
	pinpointrelay "github.com/blueimp/aws-smtp-relay/internal/relay/pinpoint"
	sesrelay "github.com/blueimp/aws-smtp-relay/internal/relay/ses"
	"github.com/mhale/smtpd"
)

var (
	addr     = flag.String("a", ":1025", "TCP listen address")
	name     = flag.String("n", "AWS SMTP Relay", "SMTP service name")
	host     = flag.String("h", "", "Server hostname")
	certFile = flag.String("c", "", "TLS cert file")
	keyFile  = flag.String("k", "", "TLS key file")
	startTLS = flag.Bool("s", false, "Require TLS via STARTTLS extension")
	onlyTLS  = flag.Bool("t", false, "Listen for incoming TLS connections only")
	relayAPI = flag.String("r", "ses", "Relay API to use (ses|pinpoint)")
	setName  = flag.String("e", "", "Amazon SES Configuration Set Name")
	ips      = flag.String("i", "", "Allowed client IPs (comma-separated)")
	user     = flag.String("u", "", "Authentication username")
)

var ipMap map[string]bool
var bcryptHash []byte
var relayClient relay.Client

func server() (srv *smtpd.Server, err error) {
	srv = &smtpd.Server{
		Addr:         *addr,
		Handler:      relayClient.Send,
		Appname:      *name,
		Hostname:     *host,
		TLSRequired:  *startTLS,
		TLSListener:  *onlyTLS,
		AuthRequired: ipMap != nil || *user != "",
		AuthHandler:  auth.New(ipMap, *user, bcryptHash).Handler,
		AuthMechs:    map[string]bool{"CRAM-MD5": false},
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

func configure() error {
	switch *relayAPI {
	case "pinpoint":
		relayClient = pinpointrelay.New(setName)
	case "ses":
		relayClient = sesrelay.New(setName)
	default:
		return errors.New("Invalid relay API: " + *relayAPI)
	}
	if *ips != "" {
		ipMap = make(map[string]bool)
		for _, ip := range strings.Split(*ips, ",") {
			ipMap[ip] = true
		}
	}
	bcryptHash = []byte(os.Getenv("BCRYPT_HASH"))
	return nil
}

func main() {
	flag.Parse()
	var srv *smtpd.Server
	err := configure()
	if err == nil {
		srv, err = server()
		if err == nil {
			err = srv.ListenAndServe()
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
