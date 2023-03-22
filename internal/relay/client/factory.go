/*
Package relay provides an interface to relay emails via Amazon SES/Pinpoint API.
*/
package client

import (
	"errors"
	"net"

	"github.com/blueimp/aws-smtp-relay/internal/relay/config"
	pinpointrelay "github.com/blueimp/aws-smtp-relay/internal/relay/pinpoint"
	sesrelay "github.com/blueimp/aws-smtp-relay/internal/relay/ses"
)

// Client provides an interface to send emails.
type Client interface {
	Send(
		origin net.Addr,
		from string,
		to []string,
		data []byte,
	) error
}

func NewClient(cfg *config.Config) (Client, error) {

	var client Client
	switch cfg.RelayAPI {
	case "pinpoint":
		client = pinpointrelay.New(&cfg.SetName, cfg.AllowFromRegExp, cfg.DenyToRegExp)
	case "ses":
		client = sesrelay.New(&cfg.SetName, cfg.AllowFromRegExp, cfg.DenyToRegExp)
	default:
		return nil, errors.New("Invalid relay API: " + cfg.RelayAPI)
	}
	return client, nil
}
