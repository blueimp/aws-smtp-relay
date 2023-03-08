package relay

import (
	"os"

	"github.com/blueimp/aws-smtp-relay/internal/auth"
	"github.com/blueimp/aws-smtp-relay/internal/relay/client"
	"github.com/blueimp/aws-smtp-relay/internal/relay/config"
	"github.com/mhale/smtpd"
)

func Server(cfg *config.Config) (srv *smtpd.Server, err error) {
	authMechs := make(map[string]bool)
	if cfg.User != "" && len(cfg.BcryptHash) > 0 && len(cfg.Password) == 0 {
		authMechs["CRAM-MD5"] = false
	}
	client, err := client.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	srv = &smtpd.Server{
		Addr:         cfg.Addr,
		Handler:      client.Send,
		Appname:      cfg.Name,
		Hostname:     cfg.Host,
		TLSRequired:  cfg.StartTLS,
		TLSListener:  cfg.OnlyTLS,
		AuthRequired: cfg.IpMap != nil || cfg.User != "",
		AuthHandler:  auth.New(cfg.IpMap, cfg.User, cfg.BcryptHash, cfg.Password).Handler,
		AuthMechs:    authMechs,
	}
	if cfg.CertFile != "" && cfg.KeyFile != "" {
		keyPass := os.Getenv("TLS_KEY_PASS")
		if keyPass != "" {
			err = srv.ConfigureTLSWithPassphrase(cfg.CertFile, cfg.KeyFile, keyPass)
		} else {
			err = srv.ConfigureTLS(cfg.CertFile, cfg.KeyFile)
		}
	}
	return
}
