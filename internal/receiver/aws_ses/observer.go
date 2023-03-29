package receiver

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqsTypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/google/uuid"

	"github.com/emersion/go-smtp"
)

type RetryAwsSesNotification struct {
	AwsSesNotification
	RetryCount int
}

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
		WaitTimeSeconds:     int32(aso.Config.SQS.WaitTime),
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

func (aso *AwsSesObserver) getS3Key(asn *RetryAwsSesNotification) *string {
	my := strings.Join([]string{aso.Config.Bucket.KeyPrefix, asn.Mail.MessageId}, "")
	return &my
}

func (aso *AwsSesObserver) fetchMessage(asn *RetryAwsSesNotification) (*s3.GetObjectOutput, error) {
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

func stringPtr(s string) *string {
	return &s
}

var re500er = regexp.MustCompile(`^5\d\d\s`)

func retry(err error) bool {
	retry := true
	if re500er.MatchString(err.Error()) {
		retry = false
	}
	return retry
}

func (aso *AwsSesObserver) sendMail(asn *RetryAwsSesNotification, out *s3.GetObjectOutput) ([]string, bool, error, *string) {
	var err error
	var c SMTPClient
	if !aso.Config.Smtp.ConnectionTLS() {
		c, err = aso.Smtp.Dial(fmt.Sprintf("%s:%d", aso.Config.Smtp.Host, aso.Config.Smtp.Port))
	} else {
		c, err = aso.Smtp.DialTLS(fmt.Sprintf("%s:%d", aso.Config.Smtp.Host, aso.Config.Smtp.Port), &tls.Config{InsecureSkipVerify: aso.Config.Smtp.InsecureTLS()})
	}
	if err != nil {
		return nil, true, err, stringPtr("Dial")
	}
	defer func() {
		c.Close()
		c.Quit()
	}()
	myName := aso.Config.Smtp.MyName
	err = c.Hello(myName)
	if err != nil {
		return nil, true, err, stringPtr("Hello")
	}
	if aso.Config.Smtp.ForceSTARTTLS() {
		err = c.StartTLS(&tls.Config{InsecureSkipVerify: aso.Config.Smtp.InsecureTLS()})
		if err != nil {
			return nil, true, err, stringPtr("StartTLS")
		}
	}
	if aso.Config.Smtp.User != "" && aso.Config.Smtp.Pass != "" {
		// auth := sasl.NewPlainClient(aso.Config.Smtp.Identity, aso.Config.Smtp.User, aso.Config.Smtp.Pass)
		// auth := smtp.CRAMMD5Auth(*aso.Config.Smtp.User, *aso.Config.Smtp.Pass)
		// auth/ := .CRAMMD5Auth(aso.Config.Smtp.User, aso.Config.Smtp.Pass)
		auth := NewCramMD5Client(aso.Config.Smtp.User, aso.Config.Smtp.Pass)
		err = c.Auth(auth)
		if err != nil {
			return nil, true, err, stringPtr("Auth")
		}
	}

	if err = c.Mail(asn.Mail.CommonHeaders.From[0], &smtp.MailOptions{}); err != nil {
		return nil, retry(err), err, stringPtr("Mail")
	}
	rcpt := make([]string, 0)
	for _, addr := range asn.Receipt.Recipients {
		if err = c.Rcpt(addr); err == nil {
			rcpt = append(rcpt, addr)
		}
	}
	if len(rcpt) == 0 {
		return rcpt, false, fmt.Errorf("no valid recipients"), stringPtr("Rcpt")
	}
	w, err := c.Data()
	if err != nil {
		return rcpt, retry(err), err, stringPtr("Data")
	}

	_, err = io.Copy(w, out.Body)
	if err != nil {
		return rcpt, retry(err), err, stringPtr("Copy")
	}

	err = w.Close()
	if err != nil {
		return rcpt, retry(err), err, stringPtr("Close")
	}
	return rcpt, false, c.Quit(), stringPtr("Quit")
}

func (aso *AwsSesObserver) deleteMessage(asn *RetryAwsSesNotification, msg *sqsTypes.Message) error {
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

func (aso *AwsSesObserver) sendRetryMessage(asm *AwsSesMessage, asn *RetryAwsSesNotification) error {
	var err error
	if aso.Config.RetryCount > 0 && asn.RetryCount+1 < aso.Config.RetryCount {
		asn.RetryCount++
		var jsonByte []byte
		jsonByte, err = json.Marshal(asn)
		if err != nil {
			return err
		}
		next := *asm
		next.Message = string(jsonByte)
		next.MessageId = uuid.New().String()
		next.Timestamp = time.Now().Format(time.RFC3339)
		jsonByte, err = json.Marshal(next)
		if err != nil {
			return err
		}
		_, err = aso.SQS.Client.SendMessage(aso.Config.Context, &sqs.SendMessageInput{
			MessageBody:  stringPtr(string(jsonByte)),
			QueueUrl:     aso.SQS.SQSQueueURL,
			DelaySeconds: int32(aso.Config.RetryDelaySeconds),
		})
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
		var sqsClient SQSClient
		sqsClient, err = aso.getSqsClient(false)
		if err != nil {
			err = LogError("sqs/getSqsClient", err.Error())
			time.Sleep(1 * time.Second)
			aso.getSqsClient(true)
			continue
		}
		var msgResult *sqs.ReceiveMessageOutput
		msgResult, err = sqsClient.ReceiveMessage(aso.Config.Context, &aso.SQS.MsgInputParams)
		if err != nil {
			err = LogError("sqs/receive", "error receiving messages, %v", err.Error())
			time.Sleep(1 * time.Second)
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
					asn := RetryAwsSesNotification{}
					err = json.Unmarshal([]byte(asm.Message), &asn)
					if err != nil {
						err = LogError("json/AwsSesNotification", err.Error())
						errx := aso.deleteMessage(&asn, &msg)
						if errx != nil {
							LogError("json/AwsSesNotification/delete", errx.Error())
						}
						continue
					}
					out, err := aso.fetchMessage(&asn)
					if err != nil {
						err = LogError("aso/fetchMessage: msg=%v err=%v", asn.Mail.MessageId, err.Error())
						errx := aso.deleteMessage(&asn, &msg)
						if errx != nil {
							LogError("json/fetchMessage/delete", errx.Error())
						}
						continue
					}
					var retry bool
					var rcpt []string
					var component *string
					rcpt, retry, err, component = aso.sendMail(&asn, out)
					mailComponent := "aso/Sendmail"
					if err != nil {
						mailComponent = fmt.Sprintf("%s/%s", mailComponent, *component)
					}
					if !retry && err != nil {
						// abort send if error is not retryable
						LogError(mailComponent, "msg=%s abort=%v from=%v to=%v", asn.Mail.MessageId, err.Error(), asn.Mail.CommonHeaders.From, asn.Mail.CommonHeaders.To)
					} else {
						if err != nil {
							// retryable error
							err = LogError(mailComponent, "msg=%s err=%v from=%v to=%v", asn.Mail.MessageId, err.Error(), asn.Mail.CommonHeaders.From, rcpt)
						} else {
							// all good
							Log(mailComponent, "sent msg=%s from=%v to=%v", asn.Mail.MessageId, asn.Mail.CommonHeaders.From, rcpt)
						}
					}
					if retry {
						err = aso.sendRetryMessage(&asm, &asn)
						if err != nil {
							LogError("sqs/sendRetryMessage", "err=%v msg=%v", err.Error(), asn.Mail.MessageId)
						}
						// delete message from queue if retry is queued
						retry = false
					}
					if !retry {
						// delete message from queue if not retryable
						err = aso.deleteMessage(&asn, &msg)
						if err != nil {
							err = LogError("sqs/deleteMessage: err=%v msg=%v", err.Error(), asn.Mail.MessageId)
							continue
						}
					}
				} else {
					err = LogError("AwsSesMessage", "unknown message type, %s", asm.Type)
				}
			}
		}
	}
	return err
}
