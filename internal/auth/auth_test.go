package auth

import (
	"crypto/hmac"
	"crypto/md5"
	"encoding/hex"
	"hash"
	"net"
	"testing"
)

// bcrypt hash for the string "password"
var sampleHash = "$2y$10$85/eICRuwBwutrou64G5HeoF3Ek/qf1YKPLba7ckiMxUTAeLIeyaC"

func createMAC(fn func() hash.Hash, message, key []byte) []byte {
	mac := hmac.New(fn, key)
	mac.Write(message)
	src := mac.Sum(nil)
	dst := make([]byte, hex.EncodedLen(len(src)))
	hex.Encode(dst, src)
	return dst
}

func TestHandler(t *testing.T) {
	ipMap := map[string]bool{"127.0.0.1": true, "2001:4860:0:2001::68": true}
	user := "username"
	bcryptHash := []byte(sampleHash)
	origin := net.TCPAddr{IP: []byte{127, 0, 0, 1}}
	auth := New(ipMap, user, bcryptHash, nil)
	success, err := auth.Handler(
		&origin,
		"LOGIN",
		[]byte(user),
		[]byte("password"),
		nil,
	)
	if success != true {
		t.Errorf("Unexpected authentication failure.")
	}
	if err != nil {
		t.Errorf("Unexpected authentication error.")
	}
}

func TestHandlerWithOriginIPv6(t *testing.T) {
	ipMap := map[string]bool{"127.0.0.1": true, "2001:4860:0:2001::68": true}
	user := "username"
	bcryptHash := []byte(sampleHash)
	origin := net.TCPAddr{IP: []byte{
		0x20, 0x01, 0x48, 0x60, 0, 0, 0x20, 0x01, 0, 0, 0, 0, 0, 0, 0x00, 0x68,
	}}
	auth := New(ipMap, user, bcryptHash, nil)
	success, err := auth.Handler(
		&origin,
		"LOGIN",
		[]byte(user),
		[]byte("password"),
		nil,
	)
	if success != true {
		t.Errorf("Unexpected authentication failure.")
	}
	if err != nil {
		t.Errorf("Unexpected authentication error.")
	}
}

func TestHandlerWithInvalidPassword(t *testing.T) {
	user := "username"
	bcryptHash := []byte(sampleHash)
	origin := net.TCPAddr{IP: []byte{127, 0, 0, 1}}
	auth := New(nil, user, bcryptHash, nil)
	success, err := auth.Handler(
		&origin,
		"LOGIN",
		[]byte(user),
		[]byte("invalid"),
		nil,
	)
	if success != false {
		t.Errorf("Unexpected password authentication success.")
	}
	if err == nil {
		t.Errorf("Unexpected missing password authentication error.")
	}
}

func TestHandlerWithInvalidUsername(t *testing.T) {
	user := "username"
	bcryptHash := []byte(sampleHash)
	origin := net.TCPAddr{IP: []byte{127, 0, 0, 1}}
	auth := New(nil, user, bcryptHash, nil)
	success, err := auth.Handler(
		&origin,
		"LOGIN",
		[]byte("invalid"),
		[]byte("password"),
		nil,
	)
	if success != false {
		t.Errorf("Unexpected username authentication success.")
	}
	if err == nil {
		t.Errorf("Unexpected missing username authentication error.")
	}
}

func TestHandlerWithNonAllowedIP(t *testing.T) {
	ipMap := map[string]bool{"127.0.0.1": true, "2001:4860:0:2001::68": true}
	origin := net.TCPAddr{IP: []byte{192, 168, 0, 1}}
	auth := New(ipMap, "", nil, nil)
	success, err := auth.Handler(
		&origin,
		"",
		nil,
		nil,
		nil,
	)
	if success != false {
		t.Errorf("Unexpected IP authentication success.")
	}
	if err == nil {
		t.Errorf("Unexpected missing IP authentication error.")
	}
}

func TestHandlerWithAuthenticationDisabled(t *testing.T) {
	origin := net.TCPAddr{IP: []byte{127, 0, 0, 1}}
	auth := New(nil, "", nil, nil)
	success, err := auth.Handler(
		&origin,
		"",
		nil,
		nil,
		nil,
	)
	if success != true {
		t.Errorf("Unexpected IP authentication failure.")
	}
	if err != nil {
		t.Errorf("Unexpected IP authentication error.")
	}
}

func TestHandlerWithPasswordConfig(t *testing.T) {
	user := "username"
	password := []byte("password")
	origin := net.TCPAddr{IP: []byte{127, 0, 0, 1}}
	auth := New(nil, user, nil, password)
	success, err := auth.Handler(
		&origin,
		"LOGIN",
		[]byte(user),
		[]byte("password"),
		nil,
	)
	if success != true {
		t.Errorf("Unexpected password authentication failure.")
	}
	if err != nil {
		t.Errorf("Unexpected password authentication error.")
	}
}

func TestHandlerWithPLAIN(t *testing.T) {
	ipMap := map[string]bool{"127.0.0.1": true, "2001:4860:0:2001::68": true}
	user := "username"
	bcryptHash := []byte(sampleHash)
	origin := net.TCPAddr{IP: []byte{127, 0, 0, 1}}
	auth := New(ipMap, user, bcryptHash, nil)
	success, err := auth.Handler(
		&origin,
		"PLAIN",
		[]byte(user),
		[]byte("password"),
		nil,
	)
	if success != true {
		t.Errorf("Unexpected authentication failure.")
	}
	if err != nil {
		t.Errorf("Unexpected authentication error.")
	}
}

func TestHandlerWithCRAMMD5(t *testing.T) {
	user := "username"
	password := []byte("password")
	origin := net.TCPAddr{IP: []byte{127, 0, 0, 1}}
	auth := New(nil, user, nil, password)
	shared := []byte("shared")
	success, err := auth.Handler(
		&origin,
		"CRAM-MD5",
		[]byte(user),
		createMAC(md5.New, shared, password),
		shared,
	)
	if success != true {
		t.Errorf("Unexpected password authentication failure.")
	}
	if err != nil {
		t.Errorf("Unexpected password authentication error.")
	}
}

func TestHandlerWithEmptyPasswordAndCRAMMD5(t *testing.T) {
	user := "username"
	origin := net.TCPAddr{IP: []byte{127, 0, 0, 1}}
	auth := New(nil, user, nil, nil)
	shared := []byte("shared")
	success, err := auth.Handler(
		&origin,
		"CRAM-MD5",
		[]byte(user),
		createMAC(md5.New, shared, nil),
		shared,
	)
	if success != true {
		t.Errorf("Unexpected password authentication failure.")
	}
	if err != nil {
		t.Errorf("Unexpected password authentication error.")
	}
}
