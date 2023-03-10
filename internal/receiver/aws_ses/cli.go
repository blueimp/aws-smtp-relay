package receiver

import (
	"github.com/spf13/pflag"
)

func initCliArgs() *Config {
	ret := Config{}
	pflag.StringVar(&ret.EnableStr, "SES.ObserverEnable", "false", "Enable AWS SES Observer")
	pflag.StringVar(&ret.SQS.Name, "SES.SQS.Name", "", "Observer AWS SQS Queue Name")
	pflag.IntVar(&ret.SQS.Timeout, "SES.SQS.Timeout", 10, "Observer AWS SQS Queue Timeout")
	pflag.IntVar(&ret.SQS.MaxMessages, "SES.SQS.MaxMessages", 10, "Observer AWS SQS Queue MaxMessages")
	pflag.IntVar(&ret.SQS.WaitTime, "SES.SQS.WaitTime", 10, "Observer AWS SQS Queue WaitTime")
	pflag.StringVar(&ret.Bucket.Name, "SES.Bucket.Name", "", "Observer AWS SQS Queue S3 Bucket")
	pflag.StringVar(&ret.Bucket.KeyPrefix, "SES.Bucket.KeyPrefix", "", "Observer AWS S3 Key Prefix")
	pflag.StringVar(&ret.Smtp.Host, "SES.Smtp.Host", "", "Observer Relay SMTP Host")
	pflag.IntVar(&ret.Smtp.Port, "SES.Smtp.Port", 25, "Observer Relay SMTP Port")
	pflag.StringVar(&ret.Smtp.ConnectionTLSStr, "SES.Smtp.ConnectionTLS", "false", "Observer Relay SMTP Connection TLS")
	pflag.StringVar(&ret.Smtp.ForceSTARTTLSStr, "SES.Smtp.ForceSTARTTLS", "true", "Observer Relay SMTP Force STARTTLS")
	pflag.StringVar(&ret.Smtp.InsecureTLSStr, "SES.Smtp.InsecureTLS", "true", "Observer Relay SMTP Insecure TLS(Certs)")
	pflag.StringVar(&ret.Smtp.Identity, "SES.Smtp.Identity", "", "Observer Relay SMTP Identity")
	pflag.StringVar(&ret.Smtp.User, "SES.Smtp.User", "", "Observer Relay SMTP User")
	pflag.StringVar(&ret.Smtp.Pass, "SES.Smtp.Pass", "", "Observer Relay SMTP User[QUEUE_SMTP_PASS]")
	pflag.StringVar(&ret.Smtp.MyName, "SES.Smtp.MyName", "AWS-SMTP-Relay-Observer", "Observer Relay SMTP Hello Name")
	return &ret
}

var FlagCliArgs = initCliArgs()
