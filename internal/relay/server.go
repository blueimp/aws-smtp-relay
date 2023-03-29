package relay

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/blueimp/aws-smtp-relay/internal/relay/config"

	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
)

type Backend struct{}

func (bkd *Backend) NewSession(_ *smtp.Conn) (smtp.Session, error) {
	return &Session{}, nil
}

// A Session is returned after EHLO.
type Session struct{}

func (s *Session) AuthPlain(username, password string) error {
	if username != "username" || password != "password" {
		return errors.New("Invalid username or password")
	}
	return nil
}

func (s *Session) Mail(from string, opts *smtp.MailOptions) error {
	log.Println("Mail from:", from)
	return nil
}

func (s *Session) Rcpt(to string) error {
	log.Println("Rcpt to:", to)
	return nil
}

func (s *Session) Data(r io.Reader) error {
	if b, err := ioutil.ReadAll(r); err != nil {
		return err
	} else {
		log.Println("Data:", string(b))
	}
	return nil
}

func (s *Session) Reset() {}

func (s *Session) Logout() error {
	return nil
}

// The CRAM-MD5 mechanism name.
const CramMD5 = "CRAM-MD5"

const (
	SendChallenge    = 1
	ValidateResponse = 2
	Done             = 3
)

type cramMD5Server struct {
	Username     string
	Secret       string
	Challange334 string
	State        int
}

// var _ sasl.Server = &cramMD5Server{}

func (a *cramMD5Server) Next(challenge []byte) (response []byte, done bool, err error) {
	switch a.State {
	case SendChallenge:
		a.State = ValidateResponse
		return []byte(a.Challange334), false, nil

	case ValidateResponse:
		a.State = Done
		sp := strings.SplitN(string(challenge), " ", 2)
		if len(sp) != 2 {
			return []byte{}, false, fmt.Errorf("Invalid challenge format: %s", string(challenge))
		}
		if a.Username != sp[0] {
			return []byte{}, false, fmt.Errorf("Invalid User: %s", string(challenge))
		}
		d := hmac.New(md5.New, []byte(a.Secret))
		d.Write([]byte(a.Challange334))
		s := make([]byte, 0, d.Size())
		sum := fmt.Sprintf("%x", d.Sum(s))
		return []byte{}, sum == sp[1], nil
	default:
		return []byte{}, false, fmt.Errorf("Invalid state: %d", a.State)
	}
}

// NewCramMD5Client implements the CRAM-MD5 authentication mechanism, as
// described in RFC 2195.
// The returned Client uses the given username and secret to authenticate to the
// server using the challenge-response mechanism.
func NewCramMD5Server(username, secret string) smtp.SaslServerFactory {
	return func(conn *smtp.Conn) sasl.Server {
		ir := make([]byte, 16)
		rand.Read(ir)
		strIr := base64.StdEncoding.EncodeToString(ir)
		return &cramMD5Server{username, secret, strIr, SendChallenge}
	}
}

func Server(cfg *config.Config) (srv *smtp.Server, err error) {
	be := &Backend{}
	srv = smtp.NewServer(be)

	srv.Addr = cfg.Addr
	srv.Domain = cfg.Host
	srv.Name = cfg.Name
	srv.ReadTimeout = time.Duration(cfg.ReadTimeout)
	srv.WriteTimeout = time.Duration(cfg.WriteTimeout)
	srv.MaxMessageBytes = 1024 * 1024
	srv.MaxRecipients = 50
	srv.AllowInsecureAuth = true
	srv.Domain = cfg.Host

	if cfg.User != "" && len(cfg.BcryptHash) > 0 && len(cfg.Password) == 0 {
		srv.EnableAuth("CRAM-MD5", NewCramMD5Server(cfg.User, string(cfg.BcryptHash)))
	}
	if cfg.CertFile != "" && cfg.KeyFile != "" {
		keyPass := os.Getenv("TLS_KEY_PASS")
		var cer tls.Certificate
		if keyPass != "" {
			var keyPem []byte
			keyPem, err = ioutil.ReadFile(cfg.KeyFile)
			if err != nil {
				return
			}
			keyPemBlock, _ := pem.Decode(keyPem)
			var keyPEMDecrypted []byte
			keyPEMDecrypted, err = x509.DecryptPEMBlock(keyPemBlock, []byte(keyPass))
			if err != nil {
				return
			}
			var pemBlock pem.Block
			pemBlock.Type = keyPemBlock.Type
			pemBlock.Bytes = keyPEMDecrypted
			keyBlock := pem.EncodeToMemory(&pemBlock)

			var certPem []byte
			certPem, err = ioutil.ReadFile(cfg.CertFile)
			if err != nil {
				return
			}
			cer, err = tls.X509KeyPair(certPem, keyBlock)
			if err != nil {
				return
			}
		} else {
			cer, err = tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
			if err != nil {
				return
			}
		}
		srv.TLSConfig = &tls.Config{
			Certificates:       []tls.Certificate{cer},
			Time:               time.Now,
			Rand:               rand.Reader,
			InsecureSkipVerify: true, // test server certificate is not trusted.
		}
	}

	return
}
