package receiver

import (
	"context"
	"crypto/tls"
	"io"

	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

type SQSClient interface {
	GetQueueUrl(ctx context.Context, params *sqs.GetQueueUrlInput, optFns ...func(*sqs.Options)) (*sqs.GetQueueUrlOutput, error)
	DeleteMessage(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error)
	ReceiveMessage(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error)
	SendMessage(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error)
}

type S3Client interface {
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
}

type SMTP interface {
	Dial(addr string) (SMTPClient, error)
	DialTLS(addr string, tls *tls.Config) (SMTPClient, error)
}

type awsSesSmtp struct{}

func (s *awsSesSmtp) Dial(addr string) (SMTPClient, error) {
	clt, err := smtp.Dial(addr)
	// clt.DebugWriter = os.Stdout
	return clt, err
}

func (s *awsSesSmtp) DialTLS(addr string, ctls *tls.Config) (SMTPClient, error) {
	conn, err := tls.Dial("tcp", addr, ctls)
	if err != nil {
		return nil, err
	}
	clt, err := smtp.NewClient(conn, addr)
	if err != nil {
		return nil, err
	}
	// clt.DebugWriter = os.Stdout
	return clt, err
}

type SMTPClient interface {
	// smtp.Client
	Close() error
	Hello(host string) error
	StartTLS(config *tls.Config) error
	Auth(a sasl.Client) error
	Mail(from string, x *smtp.MailOptions) error
	Rcpt(to string) error
	Data() (io.WriteCloser, error)
	Quit() error
}
