package main

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/pinpointemail"
	"github.com/aws/aws-sdk-go/service/ses"
	pinpointrelay "github.com/blueimp/aws-smtp-relay/internal/relay/pinpoint"
	sesrelay "github.com/blueimp/aws-smtp-relay/internal/relay/ses"
	"github.com/mhale/smtpd"
	"golang.org/x/crypto/bcrypt"
)

var (
	addr     = flag.String("a", ":1025", "TCP listen address")
	name     = flag.String("n", "AWS SMTP Relay", "SMTP service name")
	host     = flag.String("h", "", "Server hostname")
	certFile = flag.String("c", "", "TLS cert file")
	keyFile  = flag.String("k", "", "TLS key file")
	setName  = flag.String("r", "", "Amazon SES Configuration Set Name")
	startTLS = flag.Bool("s", false, "Require TLS via STARTTLS extension")
	onlyTLS  = flag.Bool("t", false, "Listen for incoming TLS connections only")
	pinpoint = flag.Bool("p", false, "Use AWS Pinpoint instead of SES")
	ips      = flag.String("i", "", "Allowed client IPs (comma-separated)")
	user     = flag.String("u", "", "Authentication username")
)

var ipMap map[string]bool
var bcryptHash []byte

func handler(origin net.Addr, from string, to []string, data []byte) {
	if *pinpoint {
		client := pinpointemail.New(session.Must(session.NewSession()))
		pinpointrelay.Send(client, origin, &from, &to, &data, setName)
	} else {
		client := ses.New(session.Must(session.NewSession()))
		sesrelay.Send(client, origin, &from, &to, &data, setName)
	}
}

func authHandler(
	remoteAddr net.Addr,
	mechanism string,
	username []byte,
	password []byte,
	shared []byte,
) (bool, error) {
	if *ips != "" {
		ip := remoteAddr.(*net.TCPAddr).IP.String()
		if ipMap[ip] != true {
			return false, errors.New("Invalid client IP: " + ip)
		}
	}
	if *user != "" {
		if string(username) != *user {
			return false, errors.New("Invalid username: " + string(username))
		}
		err := bcrypt.CompareHashAndPassword(bcryptHash, password)
		return err == nil, err
	}
	return true, nil
}

func server() (srv *smtpd.Server, err error) {
	flag.Parse()
	if *ips != "" {
		ipMap = make(map[string]bool)
		for _, ip := range strings.Split(*ips, ",") {
			ipMap[ip] = true
		}
	}
	bcryptHash = []byte(os.Getenv("BCRYPT_HASH"))
	srv = &smtpd.Server{
		Addr:         *addr,
		Handler:      handler,
		Appname:      *name,
		Hostname:     *host,
		TLSRequired:  *startTLS,
		TLSListener:  *onlyTLS,
		AuthRequired: *ips != "" || *user != "",
		AuthHandler:  authHandler,
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
