package server

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
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/blueimp/aws-smtp-relay/internal/relay"
	"github.com/blueimp/aws-smtp-relay/internal/relay/config"
	"github.com/blueimp/aws-smtp-relay/internal/relay/factory"

	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
)

type Backend struct {
	Client relay.Client
}

func (bkd *Backend) NewSession(conn *smtp.Conn) (smtp.Session, error) {
	return &Session{
		conn:   conn,
		client: bkd.Client,
	}, nil
}

// A Session is returned after EHLO.
type Session struct {
	conn   *smtp.Conn
	client relay.Client
	from   string
	to     []string
}

func (s *Session) AuthPlain(username, password string) error {
	if username != "username" || password != "password" {
		return errors.New("Invalid username or password")
	}
	return nil
}

func (s *Session) Mail(from string, opts *smtp.MailOptions) error {
	err := s.client.FilterFrom(from)
	if err != nil {
		return err
	}
	s.from = from
	return nil
}

func (s *Session) Rcpt(to string) error {
	_, _, err := s.client.FilterTo(s.from, []string{to})
	if err != nil {
		return err
	}
	s.to = append(s.to, to)
	return nil
}

func (s *Session) Data(r io.Reader) error {
	return s.client.Send(s.conn.Conn().RemoteAddr(), s.from, s.to, r)
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

type ServerOpts func(opt interface{})

func applyServerOpts(val interface{}, opts ...ServerOpts) {
	for _, opt := range opts {
		opt(val)
	}
}

func Server(cfg *config.Config, opts ...ServerOpts) (srv *smtp.Server, err error) {
	client, err := factory.NewClient(cfg)
	if err != nil {
		return
	}
	be := &Backend{Client: client}
	applyServerOpts(be, opts...)
	srv = smtp.NewServer(be)

	srv.Addr = cfg.Addr
	srv.Domain = cfg.Host
	srv.Name = cfg.Name
	srv.ReadTimeout = time.Duration(cfg.ReadTimeout)
	srv.WriteTimeout = time.Duration(cfg.WriteTimeout)
	srv.MaxMessageBytes = int(cfg.MaxMessageBytes)
	srv.MaxRecipients = 50
	srv.AllowInsecureAuth = true

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
		applyServerOpts(srv.TLSConfig, opts...)
	}

	applyServerOpts(be, opts...)
	return
}

func SplitAddr(addr string) (host string, port int) {
	port = 25
	host, portStr, _ := net.SplitHostPort(addr)
	if len(portStr) > 0 {
		tmp, err := strconv.ParseUint(portStr, 10, 16)
		if err != nil {
			return
		}
		port = int(tmp)
	}
	return
}

func StartSMTPServer(inCfg config.Config, myLog *log.Logger, clientFn func(srv *smtp.Server, lsr net.Listener), sopts ...ServerOpts) error {
	if myLog == nil {
		myLog = log.Default()
	}
	scfg, err := config.Configure(inCfg)
	if err != nil {
		return fmt.Errorf("Unexpected error: %s", err)
	}
	srv, err := Server(scfg, sopts...)
	if err != nil {
		return fmt.Errorf("unexpected error: %s", err)
	}
	needClose := false
	if scfg.Debug != "" {
		if strings.Contains(scfg.Debug, "stderr") {
			srv.Debug = os.Stderr
		} else if strings.Contains(scfg.Debug, "stdout") {
			srv.Debug = os.Stdout
		} else if scfg.Debug == "-" {
			srv.Debug = os.Stdout
		} else {
			srv.Debug, err = os.Open(scfg.Debug)
			if err != nil {
				return err
			}
			needClose = true
		}
	}
	defer func() {
		if srv.Debug != nil && needClose {
			srv.Debug.(io.WriteCloser).Close()
		}
	}()

	var lsr net.Listener
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		err = srv.ListenAndServe(func(my net.Listener) {
			lsr = my
			wg.Done()
		})
		if err != nil {
			myLog.Printf("SMTP server error: %s", err)
		}
	}()
	wg.Wait()
	defer srv.Close()
	clientFn(srv, lsr)
	return nil
}
