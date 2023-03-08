package receiver

import (
	"context"
	"errors"
	"os"
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
	Host          string
	Port          int
	ConnectionTLS bool
	ForceSTARTTLS bool
	InsecureTLS   bool
	Identity      string
	User          string
	Pass          string
	MyName        string
}

type Config struct {
	Enable  bool
	Context context.Context
	SQS     ConfigSQS
	Bucket  ConfigBucket
	Smtp    ConfigSmtp
}

func ConfigureObserver(clis ...Config) (*Config, error) {
	incli := FlagCliArgs
	if len(clis) != 0 {
		incli = clis[0]
	}
	// own copy
	cli := incli
	if !cli.Enable {
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
