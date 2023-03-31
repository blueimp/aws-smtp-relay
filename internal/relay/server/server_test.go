package server

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/pinpointemail"
	awsSes "github.com/aws/aws-sdk-go-v2/service/ses"

	"github.com/blueimp/aws-smtp-relay/internal/relay/config"
	"github.com/blueimp/aws-smtp-relay/internal/relay/pinpoint"
	"github.com/blueimp/aws-smtp-relay/internal/relay/ses"
	"github.com/blueimp/aws-smtp-relay/internal/test_utils"
	"github.com/emersion/go-smtp"
)

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
MIIEpQIBAAKCAQEA8h7vl0gUquis5jRtcnETyD+8WITZO0s53aIzp0Y+9HXiHW6F
GJjbOZjMIvozNVni+83QWKumRTgeSzIIW2j4V8iFMSNrvWmhmCKloesXS1aY6H97
9e01Ve8JWAJFRe6vZJd6gC6Z/P+ELU3ie4Vtr1GYfkV7nZ6VFp5/V/5nxGFag5TU
lpP5hcoS9r2kvXofosVwe3x3udT8SEbv5eBD4bKeVyJs/RLbxSuiU1358Y1cDdVu
Hjcvfm3cajhheQ4vX9WXsk7LGGhnf1SrrPN/y+IDTXfvoHn+nJh4vMAB4yzQdE1V
1N1AB8RA0yBVJ6dwxRrSg4BFrNWhj3gfsvrA7wIDAQABAoIBAQC4F7mIRzAfuwYr
itVk3IF0ErH8hBY/tTTsRbYMi6a2bSNdyXi9eQvGwV0Fg0OIXy3s01KE+Q5VNxQh
jIs6JZgd9sL+17XFsRlC+aUCdiOiDtf4f2YbWR7ilft+xUsynwcQ7vQfnk9LRGSV
pdB8prj9Qoc2C1KwR7GIHz8oArGXaAu3URdaG63+fuDjngk61pKAU/LCziWImoB/
PPSXpb1WwHSF8ifkelsdeeqIGf6gavtIVGPxZdlscE6yTN/Fh90pY8CwcSPWuOlf
t0gZksz8y+emvOEcP+xIxpqGrbDU0KZnCdpFfW7kAElLzjXNhaoCH4WV0KaWXZHQ
QromsehRAoGBAPmF/1vJKqLKBhei/p4Gjb95hwwb9cDvlG3K2WXfN7idCHra+6no
GAUQ93VxSR/5xBR5IIOd+rW1QKoB0pWlX3HiQ/AhYHUG55OQhO4R8l1HC+7J+JlP
ikCQKfDVjjZqBt1fQ3IX+CxlDTDYKyE4Wtr2UbOBbS4giGvheUn9CdXLAoGBAPhn
wCkqXQ9Ketuyxjl/whuiCJZBC82zBTaUNdGuKJ2y7Q1+mofSuczpq1VtjPtzqbIt
NX7CUL5zpj115A3h4mjHClKvFBr8+vcsM2syNKvIqBEReUaW9xNwoNxwTVgs87WJ
c8wBkvJ7NlIzqE6pRzldgcNhMD1qjtMSbh7ZxPztAoGBAKCvpg6ZsZc7ukimcomZ
dtcDj/BAYTZqEo/RvcZYxS1iEv/q3X5BNJauom1DEvBAjAETL9kSd01k98uDePVd
leVk7JNLKy6xz5e7zZ7yd72R7yFLd4hjLIj/TcMGA5sPFHSi0HA891i/iosV6lBu
VjQDxAFxK7o0wSWYAd+f0CGZAoGBAJ2tPczjly6dmF7cm/bjodLoh4rYvyVS/Xwn
mAIBCscPTGnEc1LD8CyiJp+TamoygQUYrVxI+/focR2SN7CYMZ9QuLzDZX+8FZHP
/NOOiuB//i7XaKPmL++nDnTe1DmkTw5ssZRNa3l/vHtxTuSfjxZaxIPArV5OxVo1
2LC8is4BAoGAVkHp95y9j6C2NWnduPwSyvBK7DMrJA7U6gjK6a2yf8v/wOsMFIWm
Qn5Tu7e19ROZr8DYaSjmPctCYVktBdsKtg33f8tHncf6bAFSIjYXy/1yfJlAR3BK
2U1MOtwM3dSTbQ0s62k0XtruV6DsUj3UsTKET00GYLNI8yCyIZVfq3w=
-----END RSA PRIVATE KEY-----`

const keyEncryptedPEM = `-----BEGIN RSA PRIVATE KEY-----
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

// bcrypt hash for the string "password"
var sampleHash = "$2y$10$85/eICRuwBwutrou64G5HeoF3Ek/qf1YKPLba7ckiMxUTAeLIeyaC"

func createTmpFile(content string) (fileName *string, err error) {
	file, err := ioutil.TempFile("", "")
	if err != nil {
		return
	}
	_, err = file.Write([]byte(content))
	if err != nil {
		return
	}
	err = file.Close()
	name := file.Name()
	fileName = &name
	return
}

func createTLSFiles(passphrase string) (
	certFile *string,
	keyFile *string,
	err error,
) {
	certFile, err = createTmpFile(certPEM)
	if err != nil {
		return
	}
	if passphrase != "" {
		keyFile, err = createTmpFile(keyEncryptedPEM)
	} else {
		keyFile, err = createTmpFile(keyPEM)
	}
	return
}

func TestServer(t *testing.T) {
	cfg, err := config.Configure()
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	srv, err := Server(cfg)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if srv.Addr != ":1025" {
		t.Errorf("Unexpected addr: %s. Expected: %s", srv.Addr, ":1025")
	}
	// t.Error("TestServer")
	// if srv.Appname != "AWS SMTP Relay" {
	// 	t.Errorf("Unexpected addr: %s. Expected: %s", srv.Appname, "AWS SMTP Relay")
	// }
	if srv.Domain != "" {
		t.Errorf("Unexpected host: %s. Expected host to be empty.", srv.Domain)
	}
	if srv.TLSConfig != nil {
		t.Errorf("Unexpected TLS config defined.")
	}
	if srv.EnableREQUIRETLS != false {
		t.Errorf("Unexpected TLS required: %t", srv.EnableREQUIRETLS)
	}
	// if srv.TLSListener != false {
	// 	t.Errorf("Unexpected TLS listener: %t", srv.TLSListener)
	// }
}

func TestServerWithCustomAddress(t *testing.T) {
	cfg, err := config.Configure(config.Config{
		Addr: "127.0.0.1:25",
	})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	srv, err := Server(cfg)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if srv.Addr != cfg.Addr {
		t.Errorf("Unexpected addr: %s. Expected: %s", srv.Addr, cfg.Addr)
	}
}

func TestServerWithCustomAppname(t *testing.T) {
	cfg, err := config.Configure(config.Config{
		Name: "Custom Appname",
	})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	srv, err := Server(cfg)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if srv.Name != cfg.Name {
		t.Error("TestServerWithCustomAppname")
	}
}

func TestServerWithCustomHostname(t *testing.T) {
	cfg, err := config.Configure(config.Config{
		Host: "test",
	})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	srv, err := Server(cfg)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if srv.Domain != cfg.Host {
		t.Error("TestServerWithCustomAppname")
	}
	// if srv.Hostname != "test" {
	// 	t.Errorf("Unexpected host: %s. Expected: %s", srv.Hostname, "test")
	// }
}

func TestServerWithIPs(t *testing.T) {
	cfg, err := config.Configure(config.Config{
		Ips: "127.0.0.1,2001:4860:0:2001::68",
	})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	srv, err := Server(cfg)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if srv.AuthDisabled != false {
		t.Errorf(
			"Unexpected AuthRequired: %t. Expected: %t",
			srv.AuthDisabled,
			true,
		)
	}
	// authMechs := map[string]bool{}
	// if !reflect.DeepEqual(srv., authMechs) {
	// 	t.Errorf(
	// 		"Unexpected AuthMechs: %v. Expected: %v",
	// 		srv.AuthMechs,
	// 		authMechs,
	// 	)
	// }
}

func TestServerWithBcryptHash(t *testing.T) {
	os.Setenv("BCRYPT_HASH", sampleHash)
	cfg, err := config.Configure(config.Config{
		User: "username",
	})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	srv, err := Server(cfg)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if srv.AuthDisabled != false {
		t.Errorf(
			"Unexpected AuthRequired: %t. Expected: %t",
			srv.AuthDisabled,
			true,
		)
	}
	// authMechs := map[string]bool{"CRAM-MD5": false}
	// if !reflect.DeepEqual(srv.AuthMechs, authMechs) {
	// 	t.Errorf(
	// 		"Unexpected AuthMechs: %v. Expected: %v",
	// 		srv.AuthMechs,
	// 		authMechs,
	// 	)
	// }
}

func TestServerWithTLS(t *testing.T) {
	certFile, keyFile, err := createTLSFiles("")
	if err != nil {
		t.Errorf("Unexpected TLS files creation error: %s", err)
		return
	}
	defer func() {
		os.Remove(*certFile)
		os.Remove(*keyFile)
	}()
	cfg, err := config.Configure(config.Config{
		CertFile: *certFile,
		KeyFile:  *keyFile,
	})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	srv, err := Server(cfg)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if srv.TLSConfig == nil {
		t.Errorf("Unexpected empty TLS config.")
	}
}

func TestServerWithTLSWithPassphrase(t *testing.T) {
	passphrase := "test"
	certFile, keyFile, err := createTLSFiles(passphrase)
	if err != nil {
		t.Errorf("Unexpected TLS files creation error: %s", err)
		return
	}
	defer func() {
		os.Remove(*certFile)
		os.Remove(*keyFile)
	}()
	cfg, err := config.Configure(config.Config{
		CertFile: *certFile,
		KeyFile:  *keyFile,
	})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	os.Setenv("TLS_KEY_PASS", passphrase)
	srv, err := Server(cfg)
	os.Unsetenv("TLS_KEY_PASS")
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if srv.TLSConfig == nil {
		t.Errorf("Unexpected empty TLS config.")
	}
}

type mockSesClient struct {
	mailInput awsSes.SendRawEmailInput
}

func (m *mockSesClient) SendRawEmail(_ context.Context, mi *awsSes.SendRawEmailInput, opts ...func(*awsSes.Options)) (*awsSes.SendRawEmailOutput, error) {
	m.mailInput = *mi
	return nil, nil
}

func startSMTPServerTest(fn func(srv *smtp.Server), opts ...ServerOpts) error {
	certFile, keyFile, deferFn, err := test_utils.GenerateX509()
	if err != nil {
		return fmt.Errorf("Unexpected error: %s", err)
	}
	defer deferFn()
	cfg, err := config.Configure(config.Config{
		Addr:       "127.0.0.1:52526",
		User:       "user",
		BcryptHash: []byte("pass"),

		CertFile: certFile,
		KeyFile:  keyFile,
	})
	if err != nil {
		return err
	}
	applyServerOpts(cfg, opts...)
	return StartSMTPServer(*cfg, nil, fn, opts...)
}

func TestFullStackAwsSes(t *testing.T) {
	mockSesClient := &mockSesClient{}
	err := startSMTPServerTest(func(srv *smtp.Server) {
		from := "Test Name <test.name@dest.test>"
		tos := []string{"bla@blub.test", "bla2@blub.test"}
		// cli.Smtp.Host, cli.Smtp.Port = SplitAddr(srv.Addr)
		err := smtp.SendMail(srv.Addr,
			nil, from, tos,
			bytes.NewBufferString("Test message"),
			func(srv interface{}) {
				tlsCfg, ok := srv.(*tls.Config)
				if !ok || tlsCfg == nil {
					return
				}
				tlsCfg.InsecureSkipVerify = true
			},
		)
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		mi := mockSesClient.mailInput
		if strings.TrimSpace(string(mi.RawMessage.Data)) != "Test message" {
			t.Errorf("Unexpected message: %s", string(mi.RawMessage.Data))
		}
		if !reflect.DeepEqual(mi.Destinations, tos) {
			t.Errorf("Unexpected destinations: %v", mi.Destinations)
		}
		if *mi.Source != from {
			t.Errorf("Unexpected source: %s", *mi.Source)
		}

	}, func(be interface{}) {
		backend, ok := be.(*Backend)
		if !ok || backend == nil {
			return
		}
		clt := backend.Client.Annotate(&ses.Client{
			SesClient: mockSesClient,
		})
		backend.Client = clt
	})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
}

type mockPinPointClient struct {
	pinPointInput pinpointemail.SendEmailInput
}

func (m *mockPinPointClient) SendEmail(_ context.Context, pi *pinpointemail.SendEmailInput, fns ...func(*pinpointemail.Options)) (*pinpointemail.SendEmailOutput, error) {
	m.pinPointInput = *pi
	return nil, nil
}

func TestFullStackPinPoint(t *testing.T) {
	mockPPClient := &mockPinPointClient{}
	err := startSMTPServerTest(func(srv *smtp.Server) {
		from := "Test Name <test.name@dest.test>"
		tos := []string{"bla@blub.test", "bla2@blub.test"}
		// cli.Smtp.Host, cli.Smtp.Port = SplitAddr(srv.Addr)
		err := smtp.SendMail(srv.Addr,
			nil, from, tos,
			bytes.NewBufferString("Test message"),
			func(srv interface{}) {
				tlsCfg, ok := srv.(*tls.Config)
				if !ok || tlsCfg == nil {
					return
				}
				tlsCfg.InsecureSkipVerify = true
			},
		)
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		pi := mockPPClient.pinPointInput
		if strings.TrimSpace(string(pi.Content.Raw.Data)) != "Test message" {
			t.Errorf("Unexpected message: %s", string(pi.Content.Raw.Data))
		}
		if !reflect.DeepEqual(pi.ReplyToAddresses, tos) {
			t.Errorf("Unexpected destinations: %v", pi.ReplyToAddresses)
		}
		if *pi.FromEmailAddress != from {
			t.Errorf("Unexpected source: %s", *pi.FromEmailAddress)
		}

	}, func(be interface{}) {
		backend, ok := be.(*Backend)
		if !ok || backend == nil {
			return
		}
		clt := backend.Client.Annotate(&pinpoint.Client{
			PinpointClient: mockPPClient,
		})
		backend.Client = clt
	}, func(cfg interface{}) {
		config, ok := cfg.(*config.Config)
		if !ok || config == nil {
			return
		}
		config.RelayAPI = "pinpoint"
	})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
}
