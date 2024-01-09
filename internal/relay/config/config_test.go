package config

import (
	"os"
	"testing"
)

// func resetHelper() {
// 	os.Args = []string{"noop"}
// 	flag.Parse()
// 	*addr = ":1025"
// 	*name = "AWS SMTP Relay"
// 	*host = ""
// 	*certFile = ""
// 	*keyFile = ""
// 	*startTLS = false
// 	*onlyTLS = false
// 	*relayAPI = "ses"
// 	*setName = ""
// 	*ips = ""
// 	*user = ""
// 	*allowFrom = ""
// 	*denyTo = ""
// 	ipMap = nil
// 	bcryptHash = nil
// 	password = nil
// 	relayClient = nil
// 	os.Unsetenv("BCRYPT_HASH")
// 	os.Unsetenv("PASSWORD")
// 	os.Unsetenv("TLS_KEY_PASS")
// }

func TestConfigure(t *testing.T) {
	cfg, err := Configure()
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if cfg.User != "" {
		t.Errorf("Unexpected username: %s", cfg.User)
	}
	if string(cfg.BcryptHash) != "" {
		t.Errorf("Unexpected bhash: %s", string(cfg.BcryptHash))
	}
	if cfg.Ips != "" {
		t.Errorf("Unexpected IPs string: %s", cfg.Ips)
	}
	if len(cfg.IpMap) != 0 {
		t.Errorf("Unexpected IP map size: %d", len(cfg.IpMap))
	}
	if cfg.RelayAPI != "ses" {
		t.Errorf("Unexpected relay API: %s", cfg.RelayAPI)
	}
	if string(cfg.BcryptHash) != "" {
		t.Errorf("Unexpected bhash: %s", string(cfg.BcryptHash))
	}
}
func TestConfigureWithBcryptHash(t *testing.T) {
	sampleHash := "$2y$10$85/eICRuwBwutrou64G5HeoF3Ek/qf1YKPLba7ckiMxUTAeLIeyaC"
	os.Setenv("BCRYPT_HASH", sampleHash)
	cfg, err := Configure()
	os.Unsetenv("BCRYPT_HASH")
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if string(cfg.BcryptHash) != sampleHash {
		t.Errorf("Unexpected bhash: %s", string(cfg.BcryptHash))
	}
}

func TestConfigureWithPassword(t *testing.T) {
	os.Setenv("PASSWORD", "password")
	cfg, err := Configure()
	os.Unsetenv("PASSWORD")
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if string(cfg.Password) != "password" {
		t.Errorf("Unexpected password: %s", string(cfg.Password))
	}
}

func TestConfigureWithIPs(t *testing.T) {
	cfg, err := Configure(Config{
		Ips: "127.0.0.1,2001:4860:0:2001::68",
	})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if len(cfg.IpMap) != 2 {
		t.Errorf("Unexpected IP map size: %d", len(cfg.IpMap))
	}
}

func TestConfigureWithAllowFrom(t *testing.T) {
	_, err := Configure(Config{
		AllowFrom: "^^admin@example\\.org$",
	})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
}

func TestConfigureWithInvalidAllowFrom(t *testing.T) {
	_, err := Configure(Config{
		AllowFrom: "(",
	})
	if err == nil {
		t.Error("Unexpected nil error")
	}
}

func TestConfigureWithDenyTo(t *testing.T) {
	_, err := Configure(Config{
		DenyTo: "^^bob@example\\.org$",
	})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
}

func TestConfigureWithInvalidDenyTo(t *testing.T) {
	_, err := Configure(Config{
		DenyTo: "(",
	})
	if err == nil {
		t.Error("Unexpected nil error")
	}
}
