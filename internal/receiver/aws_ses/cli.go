package receiver

import "flag"

func initCliArgs() Config {
	ret := Config{}
	flag.BoolVar(&ret.Enable, "SES.ObserverEnable", false, "Enable AWS SES Observer")
	flag.StringVar(&ret.SQS.Name, "SES.SQS.Name", "", "Observer AWS SQS Queue Name")
	flag.IntVar(&ret.SQS.Timeout, "SES.SQS.Timeout", 10, "Observer AWS SQS Queue Timeout")
	flag.IntVar(&ret.SQS.MaxMessages, "SES.SQS.MaxMessages", 10, "Observer AWS SQS Queue MaxMessages")
	flag.StringVar(&ret.Bucket.Name, "SES.Bucket.Name", "", "Observer AWS SQS Queue S3 Bucket")
	flag.StringVar(&ret.Bucket.KeyPrefix, "SES.Bucket.KeyPrefix", "", "Observer AWS S3 Key Prefix")
	flag.StringVar(&ret.Smtp.Host, "SES.Smtp.Host", "", "Observer Relay SMTP Host")
	flag.IntVar(&ret.Smtp.Port, "SES.Smtp.Port", 25, "Observer Relay SMTP Port")
	flag.BoolVar(&ret.Smtp.ConnectionTLS, "SES.Smtp.ConnectionTLS", false, "Observer Relay SMTP Connection TLS")
	flag.BoolVar(&ret.Smtp.ForceSTARTTLS, "SES.Smtp.ForceSTARTTLS", true, "Observer Relay SMTP Force STARTTLS")
	flag.BoolVar(&ret.Smtp.InsecureTLS, "SES.Smtp.InsecureTLS", true, "Observer Relay SMTP Insecure TLS(Certs)")
	flag.StringVar(&ret.Smtp.Identity, "SES.Smtp.Identity", "", "Observer Relay SMTP Identity")
	flag.StringVar(&ret.Smtp.User, "SES.Smtp.User", "", "Observer Relay SMTP User")
	flag.StringVar(&ret.Smtp.Pass, "SES.Smtp.Pass", "", "Observer Relay SMTP User[QUEUE_SMTP_PASS]")
	flag.StringVar(&ret.Smtp.MyName, "SES.Smtp.MyName", "AWS-SMTP-Relay-Observer", "Observer Relay SMTP Hello Name")
	return ret
}

var FlagCliArgs = initCliArgs()
