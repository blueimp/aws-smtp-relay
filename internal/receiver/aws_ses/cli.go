package receiver

import "flag"

type CliArgs struct {
	enableObserver         *bool
	queueName              *string
	queueTimeout           *int
	queueMaxMessage        *int
	queueS3Bucket          *string
	queueSmtpHost          *string
	queueSmtpPort          *int
	queueSmtpConnectionTLS *bool
	queueSmtpForceSTARTTLS *bool
	queueSmtpInsecureTLS   *bool
	queueSmtpIdentity      *string
	queueSmtpUser          *string
	queueSmtpPass          *string
	queueSmtpMyName        *string
	queueS3Prefix          *string
}

var FlagCliArgs = CliArgs{
	enableObserver:         flag.Bool("o", false, "Enable AWS SES Observer"),
	queueName:              flag.String("q", "", "Observer AWS SQS Queue Name"),
	queueTimeout:           flag.Int("T", 10, "Observer AWS SQS Queue Timeout"),
	queueMaxMessage:        flag.Int("m", 10, "Observer AWS SQS Queue MaxMessages"),
	queueS3Bucket:          flag.String("b", "", "Observer AWS SQS Queue S3 Bucket"),
	queueSmtpHost:          flag.String("H", "", "Observer Relay SMTP Host"),
	queueSmtpPort:          flag.Int("P", 25, "Observer Relay SMTP Port"),
	queueSmtpConnectionTLS: flag.Bool("S", false, "Observer Relay SMTP Connection TLS"),
	queueSmtpForceSTARTTLS: flag.Bool("F", true, "Observer Relay SMTP Force STARTTLS"),
	queueSmtpInsecureTLS:   flag.Bool("I", true, "Observer Relay SMTP Insecure TLS(Certs)"),
	queueSmtpIdentity:      flag.String("Y", "", "Observer Relay SMTP Identity"),
	queueSmtpUser:          flag.String("U", "", "Observer Relay SMTP User"),
	queueSmtpPass:          flag.String("A", "", "Observer Relay SMTP User[QUEUE_SMTP_PASS]"),
	queueSmtpMyName:        flag.String("E", "", "Observer Relay SMTP Hello Name"),
	queueS3Prefix:          flag.String("X", "", "Observer AWS S3 Key Prefix"),
}
