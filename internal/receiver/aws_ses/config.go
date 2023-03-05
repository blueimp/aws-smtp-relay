package receiver

import (
	"context"
	"errors"
	"os"
)

type AwsSesConfig struct {
	QueueName   string
	Context     context.Context
	Timeout     int32 // seconds
	MaxMessages int32
	Bucket      string
	KeyPrefix   string
	SMTP        struct {
		Host          string
		Port          int
		ConnectionTLS bool // 465
		ForceSTARTTLS bool // force STARTTLS
		InsecureTLS   bool // skip TLS verification
		User          *string
		Pass          *string
		Identity      *string // AUTH IDENTITY
		MyName        *string // EHLO name
	}
}

func ConfigureObserver(clis ...CliArgs) (*AwsSesConfig, error) {
	cli := FlagCliArgs
	if len(clis) != 0 {
		cli = clis[0]
	}
	if !*cli.enableObserver {
		return nil, nil
	}
	if *cli.queueName == "" {
		return nil, errors.New("QueueName is required")
	}
	timeout := *cli.queueTimeout
	maxMessage := *cli.queueMaxMessage
	if *cli.queueS3Bucket == "" {
		return nil, errors.New("QueueS3Bucket is required")
	}
	if *cli.queueSmtpHost == "" {
		return nil, errors.New("SMTP Host is required")
	}
	pass, is := os.LookupEnv("QUEUE_SMTP_PASS")
	if is {
		cli.queueSmtpPass = &pass
	}
	if *cli.queueSmtpMyName == "" {
		defMyName := "AWS-SMTP-Relay-Observer"
		cli.queueSmtpMyName = &defMyName
	}

	observeCfg := &AwsSesConfig{
		QueueName:   *cli.queueName,
		Context:     context.Background(),
		Timeout:     int32(timeout),
		MaxMessages: int32(maxMessage),
		Bucket:      *cli.queueS3Bucket,
		KeyPrefix:   *cli.queueS3Prefix,
		SMTP: struct {
			Host          string
			Port          int
			ConnectionTLS bool
			ForceSTARTTLS bool
			InsecureTLS   bool
			User          *string
			Pass          *string
			Identity      *string
			MyName        *string
		}{
			Host:          *cli.queueSmtpHost,
			Port:          *cli.queueSmtpPort,
			ConnectionTLS: *cli.queueSmtpConnectionTLS,
			ForceSTARTTLS: *cli.queueSmtpForceSTARTTLS,
			InsecureTLS:   *cli.queueSmtpInsecureTLS,
			User:          cli.queueSmtpUser,
			Pass:          cli.queueSmtpPass,
			Identity:      cli.queueSmtpIdentity,
			MyName:        cli.queueSmtpMyName,
		},
	}
	return observeCfg, nil
}
