package receiver

import (
	"context"
	"errors"
	"os"

	"github.com/blueimp/aws-smtp-relay/internal"
)

type ConfigSQS struct {
	Name        string
	Timeout     int
	MaxMessages int
}
type ConfigBucket struct {
	Name      string
	KeyPrefix string
}
type ConfigSmtp struct {
	Host             string
	Port             int
	ConnectionTLSStr string
	ForceSTARTTLSStr string
	InsecureTLSStr   string
	Identity         string
	User             string
	Pass             string
	MyName           string
}

func (c ConfigSmtp) ConnectionTLS() bool {
	return internal.String2bool(c.ConnectionTLSStr)
}
func (c ConfigSmtp) ForceSTARTTLS() bool {
	return internal.String2bool(c.ForceSTARTTLSStr)
}
func (c ConfigSmtp) InsecureTLS() bool {
	return internal.String2bool(c.InsecureTLSStr)
}

type Config struct {
	EnableStr string
	Context   context.Context
	SQS       ConfigSQS
	Bucket    ConfigBucket
	Smtp      ConfigSmtp
}

func (c Config) Enable() bool {
	return internal.String2bool(c.EnableStr)
}

func ConfigureObserver(clis ...Config) (*Config, error) {
	incli := FlagCliArgs
	if len(clis) != 0 {
		incli = &clis[0]
	}
	// own copy
	cli := *incli
	if !cli.Enable() {
		return nil, nil
	}
	if cli.Context == nil {
		cli.Context = context.Background()
	}
	if cli.SQS.Name == "" {
		return nil, errors.New("QueueName is required")
	}
	if cli.Bucket.Name == "" {
		return nil, errors.New("QueueS3Bucket is required")
	}
	if cli.Smtp.Host == "" {
		return nil, errors.New("SMTP Host is required")
	}
	pass, is := os.LookupEnv("QUEUE_SMTP_PASS")
	if is {
		cli.Smtp.Pass = pass
	}

	return &cli, nil
}
