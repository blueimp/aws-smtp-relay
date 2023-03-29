package config

import (
	"errors"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/blueimp/aws-smtp-relay/internal"
	"github.com/spf13/pflag"
)

type Config struct {
	Addr         string
	Name         string
	Host         string
	ReadTimeout  int64
	WriteTimeout int64
	CertFile     string
	KeyFile      string
	StartTLSStr  string
	OnlyTLSStr   string
	RelayAPI     string
	SetName      string
	Ips          string
	User         string
	IpMap        map[string]bool
	BcryptHash   []byte
	Password     []byte

	AllowFrom       string
	AllowFromRegExp *regexp.Regexp
	DenyTo          string
	DenyToRegExp    *regexp.Regexp
	// RelayClient Client
}

func (c Config) StartTLS() bool {
	return internal.String2bool(c.StartTLSStr)
}

func (c Config) OnlyTLS() bool {
	return internal.String2bool(c.OnlyTLSStr)
}

func initCliArgs() *Config {
	cfg := Config{}
	pflag.StringVarP(&cfg.Addr, "addr", "a", ":1025", "TCP listen address")
	pflag.StringVarP(&cfg.Name, "name", "n", "AWS SMTP Relay", "SMTP service name")
	pflag.StringVarP(&cfg.Host, "host", "h", "", "Server hostname")
	pflag.StringVarP(&cfg.CertFile, "certFile", "c", "", "TLS cert file")
	pflag.StringVarP(&cfg.KeyFile, "keyFile", "k", "", "TLS key file")
	pflag.StringVarP(&cfg.StartTLSStr, "startTLS", "s", "false", "Require TLS via STARTTLS extension")
	pflag.StringVarP(&cfg.OnlyTLSStr, "onlyTLS", "t", "false", "Listen for incoming TLS connections only")
	pflag.StringVarP(&cfg.RelayAPI, "relayAPI", "r", "ses", "Relay API to use (ses|pinpoint)")
	pflag.StringVarP(&cfg.SetName, "setName", "e", "", "Amazon SES Configuration Set Name")
	pflag.StringVarP(&cfg.Ips, "ips", "i", "", "Allowed client IPs (comma-separated)")
	pflag.StringVarP(&cfg.User, "user", "u", "", "Authentication username")
	pflag.StringVarP(&cfg.AllowFrom, "allowFrom", "l", "", "Allowed sender emails regular expression")
	pflag.StringVarP(&cfg.DenyTo, "denyTo", "d", "", "Denied recipient emails regular expression")
	pflag.Int64VarP(&cfg.ReadTimeout, "readTimeout", "", int64(1*time.Minute), "Read timeout in seconds")
	pflag.Int64VarP(&cfg.WriteTimeout, "writeTimeout", "", int64(1*time.Minute), "Read timeout in seconds")
	return &cfg
}

var FlagCliArgs = initCliArgs()

func merge(dominator, defaults Config) Config {
	if dominator.Addr == "" {
		dominator.Addr = defaults.Addr
	}
	if dominator.Name == "" {
		dominator.Name = defaults.Name
	}
	if dominator.Host == "" {
		dominator.Host = defaults.Host
	}
	if dominator.CertFile == "" {
		dominator.CertFile = defaults.CertFile
	}
	if dominator.KeyFile == "" {
		dominator.KeyFile = defaults.KeyFile
	}
	if dominator.RelayAPI == "" {
		dominator.RelayAPI = defaults.RelayAPI
	}
	if dominator.SetName == "" {
		dominator.SetName = defaults.SetName
	}
	if dominator.Ips == "" {
		dominator.Ips = defaults.Ips
	}
	if dominator.User == "" {
		dominator.User = defaults.User
	}
	if dominator.AllowFrom == "" {
		dominator.AllowFrom = defaults.AllowFrom
	}
	if dominator.DenyTo == "" {
		dominator.DenyTo = defaults.DenyTo
	}
	return dominator
}

func Configure(clis ...Config) (*Config, error) {
	incli := *FlagCliArgs
	if len(clis) != 0 {
		incli = clis[0]
	}
	// own copy
	cli := merge(incli, *FlagCliArgs)

	var err error
	if cli.AllowFrom != "" {
		cli.AllowFromRegExp, err = regexp.Compile(cli.AllowFrom)
		if err != nil {
			return nil, errors.New("Allowed sender emails: " + err.Error())
		}
	}
	if cli.DenyTo != "" {
		cli.DenyToRegExp, err = regexp.Compile(cli.DenyTo)
		if err != nil {
			return nil, errors.New("Denied recipient emails: " + err.Error())
		}
	}

	cli.IpMap = make(map[string]bool)
	if cli.Ips != "" {
		for _, ip := range strings.Split(cli.Ips, ",") {
			cli.IpMap[ip] = true
		}
	}
	cli.BcryptHash = []byte(os.Getenv("BCRYPT_HASH"))
	cli.Password = []byte(os.Getenv("PASSWORD"))

	return &cli, nil
}
