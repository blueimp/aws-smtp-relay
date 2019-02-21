package main

import (
	"io/ioutil"
	"net"
	"os"
	"testing"
)

// bcrypt hash for the string "password"
var sampleHash = "$2y$10$85/eICRuwBwutrou64G5HeoF3Ek/qf1YKPLba7ckiMxUTAeLIeyaC"

func createTmpFile(content string) (file *os.File, err error) {
	file, err = ioutil.TempFile("", "")
	if err != nil {
		return
	}
	_, err = file.Write([]byte(content))
	if err != nil {
		return
	}
	err = file.Close()
	return
}

func createTLSFiles() (
	certFile *os.File,
	keyFile *os.File,
	passphrase string,
	err error,
) {
	const certPEM = `-----BEGIN CERTIFICATE-----
MIIDRzCCAi+gAwIBAgIJAKtg4oViVwv4MA0GCSqGSIb3DQEBCwUAMBQxEjAQBgNV
BAMMCWxvY2FsaG9zdDAgFw0xODA0MjAxMzMxNTBaGA8yMDg2MDUwODEzMzE1MFow
FDESMBAGA1UEAwwJbG9jYWxob3N0MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIB
CgKCAQEA8h7vl0gUquis5jRtcnETyD+8WITZO0s53aIzp0Y+9HXiHW6FGJjbOZjM
IvozNVni+83QWKumRTgeSzIIW2j4V8iFMSNrvWmhmCKloesXS1aY6H979e01Ve8J
WAJFRe6vZJd6gC6Z/P+ELU3ie4Vtr1GYfkV7nZ6VFp5/V/5nxGFag5TUlpP5hcoS
9r2kvXofosVwe3x3udT8SEbv5eBD4bKeVyJs/RLbxSuiU1358Y1cDdVuHjcvfm3c
ajhheQ4vX9WXsk7LGGhnf1SrrPN/y+IDTXfvoHn+nJh4vMAB4yzQdE1V1N1AB8RA
0yBVJ6dwxRrSg4BFrNWhj3gfsvrA7wIDAQABo4GZMIGWMB0GA1UdDgQWBBQ4/ncp
befFuKH1hoYkPqLwuRrPRjAfBgNVHSMEGDAWgBQ4/ncpbefFuKH1hoYkPqLwuRrP
RjAJBgNVHRMEAjAAMBEGCWCGSAGG+EIBAQQEAwIGQDALBgNVHQ8EBAMCBaAwEwYD
VR0lBAwwCgYIKwYBBQUHAwEwFAYDVR0RBA0wC4IJbG9jYWxob3N0MA0GCSqGSIb3
DQEBCwUAA4IBAQBJBetEXiEIzKAEpXGX87j6aUON51Fdf6BiLMCghuGKyhnaOG32
4KJhtvVoS3ZUKPylh9c2VdItYlhWp76zd7YKk+3xUOixWeTMQHIvCvRGTyFibOPT
mApwp2pEnJCe4vjUrBaRhiyI+xnB70cWVF2qeernlLUeJA1mfYyQLz+v06ebDWOL
c/hPVQFB94lEdiyjGO7RZfIe8KwcK48g7iv0LQU4+c9MoWM2ZsVM1AL2tHzokSeA
u64gDTW4K0Tzx1ab7KmOFXYUjbz/xWuReMt33EwDXAErKCjbVt2T55Qx8UoKzSh1
tY0KDHdnYOzgsm2HIj2xcJqbeylYQvckNnoC
-----END CERTIFICATE-----`

	const keyPEM = `-----BEGIN RSA PRIVATE KEY-----
Proc-Type: 4,ENCRYPTED
DEK-Info: AES-256-CBC,C16BF8745B2CDB53AC2B1D7609893AA0

O13z7Yq7butaJmMfg9wRis9YnIDPsp4coYI6Ud+JGcP7iXoy95QMhovKWx25o1ol
tvUTsrsG27fHGf9qG02KizApIVtO9c1e0swCWzFrKRQX0JDiZDmilb9xosBNNst1
BOzOTRZEwFGSOCKZRBfSXyqC93TvLJ3DO9IUnKIeGt7upipvg29b/Dur/fyCy2WV
bLHXwUTDBm7j49yfoEyGkDjoB2QO9wgcgbacbnQJQ25fTFUwZpZJEJv6o1tRhoYM
ZMOhC9x1URmdHKN1+z2y5BrB6oNpParfeAMEvs/9FE6jJwYUR28Ql6Mhphfvr9W2
5Gxd3J65Ao9Vi2I5j5X6aBuNjyhXN3ScLjPG4lVZm9RU/uTPEt81pig/d5nSAjvF
Nfc08NuG3cnMyJSE/xScJ4D+GtX8U969wO4oKPCR4E/NFyXPR730ppupDFG6hzPD
PDmiszDtU438JAZ8AuFa1LkbyFnEW6KVD4h7VRr8YDjirCqnkgjNSI6dFY0NQ8H7
SyexB0lrceX6HZc+oNdAtkX3tYdzY3ExzUM5lSF1dkldnRbApLbqc4uuNIVXhXFM
dJnoPdKAzM6i+2EeVUxWNdafKDxnjVSHIHzHfIFJLQ4GS5rnz9keRFdyDjQL07tT
Lu9pPOmsadDXp7oSa81RgoCUfNZeR4jKpCk2BOft0L6ZSqwYFLcQHLIfJaGfn902
TUOTxHt0KzEUYeYSrXC2a6cyvXAd1YI7lOgy60qG89VHyCc2v5Bs4c4FNUDC/+Dj
4ZwogaAbSNkLaE0q3sYQRPdxSqLftyX0KitAgE7oGtdzBfe1cdBoozw3U67NEMMT
6qvk5j7RepPRSrapHtK5pMMdg5XpKFWcOXZ26VHVrDCj4JKdjVb4iyiQi94VveV0
w9+KcOtyrM7/jbQlCWnXpsIkP8VA/RIgh7CBn/h4oF1sO8ywP25OGQ7VWAVq1R9D
8bl8GzIdR9PZpFyOxuIac4rPa8tkDeoXKs4cxoao7H/OZO9o9aTB7CJMTL9yv0Kb
ntWuYxQchE6syoGsOgdGyZhaw4JeFkasDUP5beyNY+278NkzgGTOIMMTXIX46woP
ehzHKGHXVGf7ZiSFF+zAHMXZRSwNVMkOYwlIoRg1IbvIRbAXqAR6xXQTCVzNG0SU
cskojycBca1Cz3hDVIKYZd9beDhprVdr2a4K2nft2g2xRNjKPopsaqXx+VPibFUx
X7542eQ3eAlhkWUuXvt0q5a9WJdjJp9ODA0/d0akF6JQlEHIAyLfoUKB1HYwgUGG
6uRm651FDAab9U4cVC5PY1hfv/QwzpkNDkzgJAZ5SMOfZhq7IdBcqGd3lzPmq2FP
Vy1LVZIl3eM+9uJx5TLsBHH6NhMwtNhFCNa/5ksodQYlTvR8IrrgWlYg4EL69vjS
yt6HhhEN3lFCWvrQXQMp93UklbTlpVt6qcDXiC7HYbs3+EINargRd5Z+xL5i5vkN
f9k7s0xqhloWNPZcyOXMrox8L81WOY+sP4mVlGcfDRLdEJ8X2ofJpOAcwYCnjsKd
uEGsi+l2fTj/F+eZLE6sYoMprgJrbfeqtRWFguUgTn7s5hfU0tZ46al5d0vz8fWK
-----END RSA PRIVATE KEY-----`

	passphrase = "test"

	certFile, err = createTmpFile(certPEM)
	if err != nil {
		return
	}
	keyFile, err = createTmpFile(keyPEM)
	return
}

func TestOptions(t *testing.T) {
	os.Args = []string{"noop"}
	srv, err := server()
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if srv.Addr != ":1025" {
		t.Errorf("Unexpected addr: %s. Expected: %s", srv.Addr, ":1025")
	}
	if srv.Appname != "AWS SMTP Relay" {
		t.Errorf("Unexpected addr: %s. Expected: %s", srv.Appname, "AWS SMTP Relay")
	}
	if srv.Hostname != "" {
		t.Errorf("Unexpected host: %s. Expected host to be empty.", srv.Hostname)
	}
	if srv.TLSConfig != nil {
		t.Errorf("Unexpected TLS config defined.")
	}
	if srv.TLSRequired != false {
		t.Errorf("Unexpected TLS required: %t", srv.TLSRequired)
	}
	if srv.TLSListener != false {
		t.Errorf("Unexpected TLS listener: %t", srv.TLSListener)
	}
	if *user != "" {
		t.Errorf("Unexpected username: %s", *user)
	}
	if string(bcryptHash) != "" {
		t.Errorf("Unexpected bhash: %s", string(bcryptHash))
	}
	if *ips != "" {
		t.Errorf("Unexpected IPs string: %s", *ips)
	}
	if len(ipMap) != 0 {
		t.Errorf("Unexpected IP map size: %d", len(ipMap))
	}
	os.Args = append(
		[]string{"noop"},
		"-a",
		"127.0.0.1:25",
		"-n",
		"BANANA",
		"-h",
		"localhost",
		"-s",
		"-t",
		"-u",
		"username",
		"-i",
		"127.0.0.1,[2001:4860:0:2001::68]",
	)
	os.Setenv("BCRYPT_HASH", sampleHash)
	srv, err = server()
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if srv.Addr != os.Args[2] {
		t.Errorf("Unexpected addr: %s. Expected: %s", srv.Addr, os.Args[2])
	}
	if srv.Appname != os.Args[4] {
		t.Errorf("Unexpected addr: %s. Expected: %s", srv.Appname, os.Args[4])
	}
	if srv.Hostname != os.Args[6] {
		t.Errorf("Unexpected host: %s. Expected: %s", srv.Hostname, os.Args[6])
	}
	if srv.TLSRequired != true {
		t.Errorf("Unexpected TLS required: %t", srv.TLSRequired)
	}
	if srv.TLSListener != true {
		t.Errorf("Unexpected TLS listener: %t", srv.TLSListener)
	}
	if *user != "username" {
		t.Errorf("Unexpected username: %s", *user)
	}
	if string(bcryptHash) != sampleHash {
		t.Errorf("Unexpected bhash: %s", string(bcryptHash))
	}
	if *ips != "127.0.0.1,[2001:4860:0:2001::68]" {
		t.Errorf("Unexpected IPs string: %s", *ips)
	}
	if len(ipMap) != 2 {
		t.Errorf("Unexpected IP map size: %d", len(ipMap))
	}
	certFile, keyFile, passphrase, err := createTLSFiles()
	if err != nil {
		t.Errorf("Unexpected TLS files creation error: %s", err)
		return
	}
	defer func() {
		os.Remove(certFile.Name())
		os.Remove(keyFile.Name())
	}()
	os.Args = append(
		[]string{"noop"},
		"-c",
		certFile.Name(),
		"-k",
		keyFile.Name(),
	)
	os.Setenv("TLS_KEY_PASS", passphrase)
	srv, err = server()
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if srv.TLSConfig == nil {
		t.Errorf("Unexpected empty TLS config.")
	}
}

func TestAuthHandler(t *testing.T) {
	*user = "username"
	*ips = "127.0.0.1,[2001:4860:0:2001::68]"
	ipMap = map[string]bool{"127.0.0.1": true, "[2001:4860:0:2001::68]": true}
	bcryptHash = []byte(sampleHash)
	origin := net.TCPAddr{IP: []byte{127, 0, 0, 1}}
	success, err := authHandler(
		&origin,
		"LOGIN",
		[]byte("username"),
		[]byte("password"),
		nil,
	)
	if success != true {
		t.Errorf("Unexpected authentication failure.")
	}
	if err != nil {
		t.Errorf("Unexpected authentication error.")
	}
	success, err = authHandler(
		&origin,
		"LOGIN",
		[]byte("username"),
		[]byte("invalid"),
		nil,
	)
	if success != false {
		t.Errorf("Unexpected password authentication success.")
	}
	if err == nil {
		t.Errorf("Unexpected missing password authentication error.")
	}
	success, err = authHandler(
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
	*user = ""
	origin = net.TCPAddr{IP: []byte{192, 168, 0, 1}}
	success, err = authHandler(
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
	*ips = ""
	success, err = authHandler(
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
