package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
	"strconv"
	"log"

	"github.com/blueimp/aws-smtp-relay/internal/auth"
	"github.com/blueimp/aws-smtp-relay/internal/relay"
	pinpointrelay "github.com/blueimp/aws-smtp-relay/internal/relay/pinpoint"
	sesrelay "github.com/blueimp/aws-smtp-relay/internal/relay/ses"
	"github.com/mhale/smtpd"
)

var (
	addr      = flag.String("a", LookupEnvOrString("LISTEN_ADDRESS", ":1025"), "TCP listen address")
	name      = flag.String("n", LookupEnvOrString("SMTP_SERVICE_NAME", "AWS SMTP Relay"), "SMTP service name")
	host      = flag.String("h", LookupEnvOrString("HOSTNAME", ""), "Server hostname")
	certFile  = flag.String("c", LookupEnvOrString("CERT_FILE", ""), "TLS cert file")
	keyFile   = flag.String("k", LookupEnvOrString("KEY_FILE", ""), "TLS key file")
	startTLS  = flag.Bool("s", LookupEnvOrBool("REQUIRE_STARTTLS", false), "Require TLS via STARTTLS extension")
	onlyTLS   = flag.Bool("t", LookupEnvOrBool("REQUIRE_TLS", false), "Listen for incoming TLS connections only")
	relayAPI  = flag.String("r", LookupEnvOrString("RELAY_API", "ses"), "Relay API to use (ses|pinpoint)")
	setName   = flag.String("e", LookupEnvOrString("SES_CONFIGURATION_SET_NAME", ""), "Amazon SES Configuration Set Name")
	ips       = flag.String("i", LookupEnvOrString("ALLOWED_IPS", ""), "Allowed client IPs (comma-separated)")
	user      = flag.String("u", LookupEnvOrString("AUTH_USERNAME", ""), "Authentication username")
	allowFrom = flag.String("l", LookupEnvOrString("ALLOWED_SENDERS_REGEX", ""), "Allowed sender emails regular expression")
	denyTo    = flag.String("d", LookupEnvOrString("DENIED_SENDERS_REGEX", ""), "Denied recipient emails regular expression")
)

var ipMap map[string]bool
var bcryptHash []byte
var password []byte
var relayClient relay.Client

func server() (srv *smtpd.Server, err error) {
	authMechs := make(map[string]bool)
	if *user != "" && len(bcryptHash) > 0 && len(password) == 0 {
		authMechs["CRAM-MD5"] = false
	}

	// force-enable the login mechanism: only needed if there is no external routing and TLS is not configured
	if LookupEnvOrString("ENABLE_LOGIN", "") == "true" {
		authMechs["LOGIN"] = true
	}
	
	version := LookupEnvOrString("GIT_REV", "")
	if version == "" {
		log.Printf("Listening on %v\r\n", *addr)
	}else{
		log.Printf("(Revision: %s) Listening on %v\r\n", version, *addr)
	}
	srv = &smtpd.Server{
		Addr:         *addr,
		Handler:      relayClient.Send,
		Appname:      *name,
		Hostname:     *host,
		TLSRequired:  *startTLS,
		TLSListener:  *onlyTLS,
		AuthRequired: ipMap != nil || *user != "",
		AuthHandler:  auth.New(ipMap, *user, bcryptHash, password).Handler,
		AuthMechs:    authMechs,
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
	var allowFromRegExp *regexp.Regexp
	var denyToRegExp *regexp.Regexp
	var err error
	if *allowFrom != "" {
		allowFromRegExp, err = regexp.Compile(*allowFrom)
		if err != nil {
			return errors.New("Allowed sender emails: " + err.Error())
		}
	}
	if *denyTo != "" {
		denyToRegExp, err = regexp.Compile(*denyTo)
		if err != nil {
			return errors.New("Denied recipient emails: " + err.Error())
		}
	}
	switch *relayAPI {
	case "pinpoint":
		relayClient = pinpointrelay.New(setName, allowFromRegExp, denyToRegExp)
	case "ses":
		relayClient = sesrelay.New(setName, allowFromRegExp, denyToRegExp)
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
	password = []byte(os.Getenv("PASSWORD"))
	return nil
}

// returns the value of an environment variable or the default value
func LookupEnvOrString(key string, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return defaultVal
}

func LookupEnvOrInt(key string, defaultVal int) int {
	if val, ok := os.LookupEnv(key); ok {
		v, err := strconv.Atoi(val)
		if err != nil {
			log.Fatalf("LookupEnvOrInt[%s]: %v", key, err)
		}
		return v
	}
	return defaultVal
}

func LookupEnvOrBool(key string, defaultVal bool) bool {
	if val, ok := os.LookupEnv(key); ok {
		val = strings.ToLower(val)
		if val == "true" {
			return true
		}else if val == "false" {
			return false
		}
		log.Fatalf("LookupEnvOrBool[%s]: invalid value %v", key, val)
	}
	return defaultVal
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
