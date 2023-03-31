package factory

import (
	"reflect"
	"testing"

	"github.com/blueimp/aws-smtp-relay/internal/relay/config"
)

func TestConfigureWithPinpointRelay(t *testing.T) {
	cfg, err := config.Configure(config.Config{
		RelayAPI: "pinpoint",
	})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	client, err := NewClient(cfg)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	typ := reflect.TypeOf(client).String()
	if typ != "pinpoint.Client" {
		t.Errorf("Unexpected type: %s", typ)
	}

	// _, ok := interface{}(client).(pinpointrelay.Client)
	// if !ok {
	// 	t.Error("Unexpected: relayClient function is not an sesrelay.Client")
	// }
}

func TestConfigureWithSesRelay(t *testing.T) {
	cfg, err := config.Configure(config.Config{
		RelayAPI: "ses",
	})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	client, err := NewClient(cfg)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	typ := reflect.TypeOf(client).String()

	if typ != "ses.Client" {
		t.Errorf("Unexpected type: %s", typ)
	}

	// _, ok := interface{}(client).(pinpointrelay.Client)
	// if !ok {
	// 	t.Error("Unexpected: relayClient function is not an sesrelay.Client")
	// }
}

func TestConfigureWithInvalidRelay(t *testing.T) {
	cfg, err := config.Configure(config.Config{
		RelayAPI: "invalid",
	})
	if err != nil {
		t.Error("Unexpected nil error")
	}
	_, err = NewClient(cfg)
	if err == nil {
		t.Error("Unexpected nil error")
	}
}
