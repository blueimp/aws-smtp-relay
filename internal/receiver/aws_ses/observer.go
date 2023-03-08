package receiver

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/smtp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqsTypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type AwsSesObserver struct {
	SQS struct {
		Client         SQSClient
		MsgInputParams sqs.ReceiveMessageInput
		SQSQueueURL    *string
	}
	S3Client S3Client
	Smtp     SMTP
	Config   Config
}

func (aso *AwsSesObserver) getSqsClient(reset ...bool) (SQSClient, error) {
	if (len(reset) == 0 || !reset[0]) && aso.SQS.Client != nil {
		return aso.SQS.Client, nil
	}
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, LogError("AwsSesObserver.getSqsClient", "error loading aws config, %s", err.Error())
	}
	aso.SQS.Client = sqs.NewFromConfig(cfg)
	return aso.SQS.Client, nil
}

func (aso *AwsSesObserver) getS3Client(reset ...bool) (S3Client, error) {
	if (len(reset) == 0 || !reset[0]) && aso.S3Client != nil {
		return aso.S3Client, nil
	}
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, LogError("AwsSesObserver.getSqsClient", "error loading aws config, %s", err.Error())
	}
	aso.S3Client = s3.NewFromConfig(cfg)
	return aso.S3Client, nil
}

func (aso *AwsSesObserver) InitSQS() error {
	// Get URL of queue
	sqsClient, err := aso.getSqsClient()
	if err != nil {
		return err
	}

	urlResult, err := sqsClient.GetQueueUrl(aso.Config.Context, &sqs.GetQueueUrlInput{
		QueueName: &aso.Config.SQS.Name,
	})
	if err != nil {
		return fmt.Errorf("error getting queue url, " + err.Error())
	}
	aso.SQS.SQSQueueURL = urlResult.QueueUrl

	aso.SQS.MsgInputParams = sqs.ReceiveMessageInput{
		MessageAttributeNames: []string{
			string(sqsTypes.QueueAttributeNameAll),
		},
		QueueUrl:            aso.SQS.SQSQueueURL,
		MaxNumberOfMessages: int32(aso.Config.SQS.MaxMessages),
		VisibilityTimeout:   int32(aso.Config.SQS.Timeout),
	}
	return nil
}

func NewAWSSESObserver(cfg *Config) (*AwsSesObserver, error) {
	if cfg.Context == nil {
		cfg.Context = context.TODO()
	}
	return &AwsSesObserver{
		Config: *cfg,
		Smtp:   &awsSesSmtp{},
	}, nil
}

func (aso *AwsSesObserver) getS3Key(asn *AwsSesNotification) *string {
	my := strings.Join([]string{aso.Config.Bucket.KeyPrefix, asn.Mail.MessageId}, "")
	return &my
}

func (aso *AwsSesObserver) fetchMessage(asn *AwsSesNotification) (*s3.GetObjectOutput, error) {
	var err error
	var out *s3.GetObjectOutput
	for i := 0; i < 2; i++ {
		var s3Client S3Client
		s3Client, err = aso.getS3Client(i > 0)
		if err != nil {
			return nil, err
		}
		out, err = s3Client.GetObject(aso.Config.Context, &s3.GetObjectInput{
			Bucket: &aso.Config.Bucket.Name,
			Key:    aso.getS3Key(asn),
		})
		if err != nil {
			// retry in case of error
			continue
		}
		break
	}
	return out, err
}

func (aso *AwsSesObserver) sendMail(asn *AwsSesNotification, out *s3.GetObjectOutput) (error, error) {
	var err error
	var c SMTPClient
	if !aso.Config.Smtp.ConnectionTLS {
		c, err = aso.Smtp.Dial(fmt.Sprintf("%s:%d", aso.Config.Smtp.Host, aso.Config.Smtp.Port))
	} else {
		c, err = aso.Smtp.DialTLS(fmt.Sprintf("%s:%d", aso.Config.Smtp.Host, aso.Config.Smtp.Port), &tls.Config{InsecureSkipVerify: aso.Config.Smtp.InsecureTLS})
	}
	if err != nil {
		return nil, err
	}
	defer c.Close()
	myName := aso.Config.Smtp.MyName
	err = c.Hello(myName)
	if err != nil {
		return nil, err
	}
	if aso.Config.Smtp.ForceSTARTTLS {
		err = c.StartTLS(&tls.Config{InsecureSkipVerify: aso.Config.Smtp.InsecureTLS})
		if err != nil {
			return nil, err
		}
	}
	if aso.Config.Smtp.User != "" && aso.Config.Smtp.Pass != "" {
		// auth := sasl.NewLoginClient(*aso.Config.Smtp.User, *aso.Config.Smtp.Pass)
		// auth := smtp.CRAMMD5Auth(*aso.Config.Smtp.User, *aso.Config.Smtp.Pass)
		auth := smtp.CRAMMD5Auth(aso.Config.Smtp.User, aso.Config.Smtp.Pass)
		err = c.Auth(auth)
		if err != nil {
			return nil, err
		}
	}

	if err = c.Mail(asn.Mail.CommonHeaders.From[0]); err != nil {
		return err, nil
	}
	rcptCnt := 0
	for _, addr := range asn.Receipt.Recipients {
		if err = c.Rcpt(addr); err == nil {
			rcptCnt++
		}
	}
	if rcptCnt == 0 {
		return errors.New("no valid recipients"), nil
	}
	w, err := c.Data()
	if err != nil {
		return nil, err
	}

	_, err = io.Copy(w, out.Body)
	if err != nil {
		return nil, err
	}

	err = w.Close()
	if err != nil {
		return nil, err
	}
	return c.Quit(), nil
}

func (aso *AwsSesObserver) deleteMessage(asn *AwsSesNotification, msg *sqsTypes.Message) error {
	var err error
	for i := 0; i < 2; i++ {
		client, err := aso.getSqsClient(i > 0)
		if err != nil {
			return err
		}
		_, err = client.DeleteMessage(aso.Config.Context, &sqs.DeleteMessageInput{
			QueueUrl:      aso.SQS.SQSQueueURL,
			ReceiptHandle: msg.ReceiptHandle,
		})
		if err != nil {
			// retry in case of error
			continue
		}
		break
	}

	for i := 0; i < 2; i++ {
		client, err := aso.getS3Client(i > 0)
		if err != nil {
			return err
		}
		_, err = client.DeleteObject(aso.Config.Context, &s3.DeleteObjectInput{
			Bucket: &aso.Config.Bucket.Name,
			Key:    aso.getS3Key(asn),
		})
		if err != nil {
			// retry in case of error
			continue
		}
		break
	}
	return err
}

func (aso *AwsSesObserver) Observe(cnts ...int) error {
	cnt := -1
	if len(cnts) > 0 {
		cnt = cnts[0]
	}
	var err error
	Log("sqs/observe", "start observing %d messages", cnt)
	for i := 0; cnt < 0 || i < cnt; i++ {
		var sqs SQSClient
		sqs, err = aso.getSqsClient(false)
		if err != nil {
			err = LogError("sqs/getSqsClient", err.Error())
			time.Sleep(1000 * time.Millisecond)
			aso.getSqsClient(true)
			continue
		}
		msgResult, err := sqs.ReceiveMessage(aso.Config.Context, &aso.SQS.MsgInputParams)
		if err != nil {
			err = LogError("sqs/receive", "error receiving messages, %v", err.Error())
			time.Sleep(1000 * time.Millisecond)
			aso.getSqsClient(true)
			continue
		}

		if msgResult.Messages != nil {
			for _, msg := range msgResult.Messages {
				asm := AwsSesMessage{}
				err = json.Unmarshal([]byte(*msg.Body), &asm)
				if err != nil {
					err = LogError("json/AwsSesMessage", err.Error())
					continue
				}
				if asm.Type == "Notification" {
					asn := AwsSesNotification{}
					err = json.Unmarshal([]byte(asm.Message), &asn)
					if err != nil {
						err = LogError("json/AwsSesNotification", err.Error())
						continue
					}
					out, err := aso.fetchMessage(&asn)
					if err != nil {
						err = LogError("aso/fetchMessage", err.Error())
						continue
					}
					warn, err := aso.sendMail(&asn, out)
					if warn != nil {
						LogError("aso/sendMail", "warn=%v msg=%s to=%v", warn.Error(), asn.Mail.MessageId, asn.Mail.CommonHeaders.To)
					}
					if err != nil {
						err = LogError("aso/sendMail", "err=%v msg=%s to=%v", err.Error(), asn.Mail.MessageId, asn.Mail.CommonHeaders.To)
					} else {
						Log("smtp/sendMail", "sent msg=%s to=%v", asn.Mail.MessageId, asn.Mail.CommonHeaders.To)
					}
					err = aso.deleteMessage(&asn, &msg)
					if err != nil {
						err = LogError("sqs/deleteMessage", err.Error())
						continue
					}
				} else {
					err = LogError("AwsSesMessage", "unknown message type, %s", asm.Type)
				}
			}
		}
	}
	return err
}
