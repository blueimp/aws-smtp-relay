/*
Package relay provides an interface to relay emails via Amazon SES/Pinpoint API.
*/
package factory

import (
	"errors"

	"github.com/blueimp/aws-smtp-relay/internal/relay"
	"github.com/blueimp/aws-smtp-relay/internal/relay/config"
	pinpointrelay "github.com/blueimp/aws-smtp-relay/internal/relay/pinpoint"
	sesrelay "github.com/blueimp/aws-smtp-relay/internal/relay/ses"
)

// Client provides an interface to send emails.

func NewClient(cfg *config.Config) (relay.Client, error) {

	var client relay.Client
	var err error
	switch cfg.RelayAPI {
	case "pinpoint":
		client = pinpointrelay.New(&cfg.SetName, cfg.AllowFromRegExp, cfg.DenyToRegExp, uint(cfg.MaxMessageBytes))
	case "ses":
		client, err = sesrelay.New(&cfg.SetName, cfg.AllowFromRegExp, cfg.DenyToRegExp, uint(cfg.MaxMessageBytes))
	default:
		return nil, errors.New("Invalid relay API: " + cfg.RelayAPI)
	}
	return client, err
}
