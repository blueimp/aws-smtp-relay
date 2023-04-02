package receiver

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	awsSes "github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqsTypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"

	"github.com/blueimp/aws-smtp-relay/internal/relay/config"
	"github.com/blueimp/aws-smtp-relay/internal/relay/server"
	"github.com/blueimp/aws-smtp-relay/internal/relay/ses"
	"github.com/blueimp/aws-smtp-relay/internal/test_utils"
	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
	"github.com/google/uuid"
)

func boolPtr(v bool) *bool {
	return &v
}

func strPtr(v string) *string {
	return &v
}

func intPtr(v int) *int {
	return &v
}

type mockSQSClient struct {
	nextMessages     []sqsTypes.Message
	receivedMessages []sqsTypes.Message
	sendMessages     []sqsTypes.Message
}

func (m *mockSQSClient) GetQueueUrl(ctx context.Context, params *sqs.GetQueueUrlInput, optFns ...func(*sqs.Options)) (*sqs.GetQueueUrlOutput, error) {
	if *params.QueueName == "testQ" {
		return &sqs.GetQueueUrlOutput{
			QueueUrl: strPtr("q://testQ"),
		}, nil
	}
	return nil, fmt.Errorf("queue not found")
}
func (m *mockSQSClient) DeleteMessage(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error) {
	found := false
	for i, msg := range m.nextMessages {
		if *msg.ReceiptHandle == *params.ReceiptHandle {
			m.nextMessages = append(m.nextMessages[:i], m.nextMessages[i+1:]...)
			found = true
		}
	}
	if found && *params.QueueUrl == "q://testQ" {
		return nil, nil
	}
	return nil, fmt.Errorf("queue not found")
}

func (m *mockSQSClient) ReceiveMessage(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
	if *params.QueueUrl == "q://testQ" &&
		params.MaxNumberOfMessages == 10 &&
		params.VisibilityTimeout == 10 {
		if len(m.nextMessages) == 0 {
			m.nextMessages = []sqsTypes.Message{sqsMessage()}
		}
		m.receivedMessages = append(m.receivedMessages, m.nextMessages...)
		return &sqs.ReceiveMessageOutput{
			Messages: m.nextMessages,
		}, nil
	}
	return nil, fmt.Errorf("queue not found")
}

func (m *mockSQSClient) SendMessage(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
	my := sqsTypes.Message{
		Body:          params.MessageBody,
		MessageId:     stringPtr(uuid.New().String()),
		ReceiptHandle: stringPtr(uuid.New().String()),
	}
	m.nextMessages = append(m.nextMessages, my)
	m.sendMessages = append(m.sendMessages, my)
	return &sqs.SendMessageOutput{
		MessageId: my.MessageId,
	}, nil
}

type mockS3Client struct {
}

func s3GetObjectOutput(body string) *s3.GetObjectOutput {
	return &s3.GetObjectOutput{
		Body: io.NopCloser(strings.NewReader(body)),
	}
}

func (m *mockS3Client) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	if *params.Bucket == "bucket" && *params.Key == "prefix/nrk5vlqu9usuh476ffj0j3is23okmot9h029da01" {
		return s3GetObjectOutput("testBody"), nil
	}
	return nil, fmt.Errorf("object not found")
}

func (m *mockS3Client) DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	if *params.Bucket == "bucket" && *params.Key == "prefix/nrk5vlqu9usuh476ffj0j3is23okmot9h029da01" {
		return nil, nil
	}
	return nil, fmt.Errorf("object not found")
}

type mockSMTPClient struct {
	buf       bytes.Buffer
	mailErrFn func(string) error
}

func (sc *mockSMTPClient) Close() error {
	return nil
}
func (sc *mockSMTPClient) Hello(host string) error {
	if host == "AWS-SMTP-Relay-Observer" {
		return nil
	}
	return fmt.Errorf("hello error")
}

func (sc *mockSMTPClient) StartTLS(config *tls.Config) error {
	if config.InsecureSkipVerify == true {
		return nil
	}
	return fmt.Errorf("starttls error")
}

func (sc *mockSMTPClient) Auth(a sasl.Client) error {
	mech, _, err := a.Start() // &smtp.ServerInfo{})
	if err != nil {
		return err
	}
	user, err := a.Next([]byte{} /* , true */)
	if err != nil {
		return err
	}
	if err == nil && mech == "CRAM-MD5" && string(user) == "user 6f6f78664432e7632bb899845c4782ba" {
		return nil
	}
	return fmt.Errorf("auth error")
}

func (sc *mockSMTPClient) Mail(from string, _ *smtp.MailOptions) error {
	if sc.mailErrFn != nil {
		return sc.mailErrFn(from)
	}
	if from == "Meno Abels <from@smtp.world>" {
		return nil
	}
	return fmt.Errorf("mail error")
}

func (sc *mockSMTPClient) Rcpt(to string) error {
	if to == "to@smtp.world" {
		return nil
	}
	return fmt.Errorf("mail error")
}

type bufWriteCloser struct {
	buf *bytes.Buffer
}

func (b *bufWriteCloser) Write(p []byte) (n int, err error) {
	return b.buf.Write(p)
}

func (b *bufWriteCloser) Close() error {
	return nil
}

func (sc *mockSMTPClient) Data() (io.WriteCloser, error) {
	return &bufWriteCloser{buf: &sc.buf}, nil
}

func (sc *mockSMTPClient) Quit() error {
	if sc.buf.String() == "testBody" {
		return nil
	}
	return fmt.Errorf("quit error")
}

func mockNewAWSSESObserver(cfg *Config) (*AwsSesObserver, error) {
	obs, err := NewAWSSESObserver(cfg)
	if err != nil {
		return nil, err
	}
	obs.SQS.Client = &mockSQSClient{}
	err = obs.InitSQS()
	if err != nil {
		return nil, err
	}
	if *obs.SQS.SQSQueueURL != "q://testQ" {
		return nil, err
	}
	obs.S3Client = &mockS3Client{}
	obs.Smtp = &mockSMTP{}
	return obs, nil
}

func (sc *mockSMTPClient) SendMail(from string, to []string, msg io.Reader) error {
	var err error
	if from == "Meno Abels <from@smtp.world>" && to[0] == "to@smtp.world" {
		buf := make([]byte, 1024)
		len, err := msg.Read(buf)
		if string(buf[:len]) == "testBody" && err == nil {
			return nil
		}
	}
	return fmt.Errorf("sendmail error: %v", err)
}

type mockSMTP struct {
	smtpClient SMTPClient
}

func (s *mockSMTP) Dial(addr string) (SMTPClient, error) {
	if addr == "host:25" {
		if s.smtpClient == nil {
			my := mockSMTPClient{}
			return &my, nil
		}
		return s.smtpClient, nil
	}
	return nil, fmt.Errorf("dial error")
}

func (s *mockSMTP) DialTLS(addr string, tls *tls.Config) (SMTPClient, error) {
	if addr == "host:25" {
		if s.smtpClient == nil {
			my := mockSMTPClient{}
			return &my, nil
		}
		return s.smtpClient, nil
	}
	return nil, fmt.Errorf("dial error")
}

func sqsMessage() sqsTypes.Message {
	ses := AwsSesMessage{
		Type:             "Notification",
		MessageId:        uuid.NewString(),
		TopicArn:         "arn:aws:sns:us-east-1:XXXXXXX:smtp-q",
		Subject:          "Amazon SES Email Receipt Notification",
		Message:          "{\"notificationType\":\"Received\",\"mail\":{\"timestamp\":\"2023-03-03T10:25:18.793Z\",\"source\":\"test@test.ipv6\",\"messageId\":\"nrk5vlqu9usuh476ffj0j3is23okmot9h029da01\",\"destination\":[\"dest@lurks.com\"],\"headersTruncated\":false,\"headers\":[{\"name\":\"Return-Path\",\"value\":\"<lurks@kddkker.mdmd>\"},{\"name\":\"Received\",\"value\":\"from mail-ua1-f52.google.com (mail-ua1-f52.google.com [209.85.222.52]) by inbound-smtp.us-east-1.amazonaws.com with SMTP id nrk5vlqu9usuh476ffj0j3is23okmot9h029da01 for jhuhdvh@sdkfkjfkdd.dodo; Fri, 03 Mar 2023 10:25:18 +0000 (UTC)\"},{\"name\":\"X-SES-Spam-Verdict\",\"value\":\"PASS\"},{\"name\":\"X-SES-Virus-Verdict\",\"value\":\"PASS\"},{\"name\":\"Received-SPF\",\"value\":\"pass (spfCheck: domain of _spf.google.com designates 209.85.222.52 as permitted sender) client-ip=209.85.222.52; envelope-from=from@smtp.world; helo=mail-ua1-f52.google.com;\"},{\"name\":\"Authentication-Results\",\"value\":\"amazonses.com; spf=pass (spfCheck: domain of _spf.google.com designates 209.85.222.52 as permitted sender) client-ip=209.85.222.52; envelope-from=from@smtp.world; helo=mail-ua1-f52.google.com; dkim=pass header.i=@gmail.com; dmarc=pass header.from=gmail.com;\"},{\"name\":\"X-SES-RECEIPT\",\"value\":\"AEFBQUFBQUFBQUFGcnE2YU0rN2FyVGhUUHh4Q2pyS3pNRmtXV1hVN1RvSFltcE9ZakdoM3ozRExaWFdhM05MNzBZWG9xSVRUbmg4RmpYOFZvWnNnTys3NFpFSlZJL1ZOaW9KOWFwd3dyZDhwdFM4WTJWOEpsc2VsbUZ2NFlsTHZnYWRRRERheVZGZnd0aEkxTW4zTUI4Q29jVDFoYjRnK2hmYlkySC9xWm8wUVo4MjBEdWdXN0dEazdndnBKa0xUb2VvaHNOa3ZoblM5MU1HRnhyZitpT0oxYlZITWlLcFlVNUMzNkxkc0RUbDg1bXQ3My93cWZzNitWaTRBM2ZJMGcrTDVwSkRKei90eUdybW9hN1VOQjg5R1JzQUFSUXFhMnJkSGFCWXJJYWI4VWRpRXBLV0lCY0E9PQ==\"},{\"name\":\"X-SES-DKIM-SIGNATURE\",\"value\":\"a=rsa-sha256; q=dns/txt; b=CXFYhALt4tfdyrB7s8fEBNtD1htsVN9R25Szm4LCI9is4apzx5Gdu9iiExL1MYBcamzMMk0mamLCxNosLC7HCgOmp5IPjTG2hFNf9UAkbg+3jS3mlAY6fSWw96s/dujH8gZoXvinkfUDlf0HYvYuETYOSVYRzNmXtLiLHqbdoqQ=; c=relaxed/simple; s=6gbrjpgwjskckoa6a5zn6fwqkn67xbtw; d=amazonses.com; t=1677839119; v=1; bh=BCp5hxcYf0BCkCwMBUE/WxEPF1FnOMQIUcxnNYyPm2I=; h=From:To:Cc:Bcc:Subject:Date:Message-ID:MIME-Version:Content-Type:X-SES-RECEIPT;\"},{\"name\":\"Received\",\"value\":\"by mail-ua1-f52.google.com with SMTP id n4so1266357ual.13 for <to@smtp.world>; Fri, 03 Mar 2023 02:25:18 -0800 (PST)\"},{\"name\":\"DKIM-Signature\",\"value\":\"v=1; a=rsa-sha256; c=relaxed/relaxed; d=gmail.com; s=20210112; t=1677839118; h=to:subject:message-id:date:from:mime-version:from:to:cc:subject:date:message-id:reply-to; bh=BCp5hxcYf0BCkCwMBUE/WxEPF1FnOMQIUcxnNYyPm2I=; b=omWvzm0ZX8KPQd1JJKSvZoHm1MES89nEFjzIUJly22fqcfusPuJvOl7t5lNUlfxuiRewN7ZLjfvhKNmx6twlqp2OxI8GZaPFDoshptLEVYRcmzRv8S01bUrRdhGTlvQ/ayaghAADZq/VDJVeWw8cj0woJ1GwTEIPyRP3L2wmqm1G2NReXts0Yq6BrikRBNT3MVRFUlpdsHs0GWgRCLPPZAlyui29ig3BcWowYCVATFkO8i0vlmX8FkdFMSapo8RtrMD43x0zFZ8FgmMBascqx0BruBqcOyqFU0zj+56sKQPkcGrdgRyvh2Sxy//QSJsWry7XJeQUXyoh0ZQmI0X7Zw==\"},{\"name\":\"X-Google-DKIM-Signature\",\"value\":\"v=1; a=rsa-sha256; c=relaxed/relaxed; d=1e100.net; s=20210112; t=1677839118; h=to:subject:message-id:date:from:mime-version:x-gm-message-state :from:to:cc:subject:date:message-id:reply-to; bh=BCp5hxcYf0BCkCwMBUE/WxEPF1FnOMQIUcxnNYyPm2I=; b=kJFq/D3qY/eCttS85FW+ktATEk+C4Fen3Jrn9sloZ0peqxVWA3S3X7t2rFduxsab1h syK1i9gzLluwCx3ExDr3o2OaQrzHOZwzf4PS6x7ON/NJ4GgQv4HK6rNY80xEoGErxJDQ PMw4A8k/UUCbTEej+3yL4Dticl/hIY08W6Y0yp3gXe1o426BsA/WRR5UXR02MagidOSL siz892AcvurH02GPRJezj/LNx6Mqeqtzv0fpiBpy4r0TRO7JeLBPCSbInndV4he3uZC3 qwuLZURFkQ7e7lugfNGvIuYi+473JvOWWVlqkpfV4vJxaRarioslX2O9jrM4pt83Uesm nHWQ==\"},{\"name\":\"X-Gm-Message-State\",\"value\":\"AO0yUKUSN5Ddt3VxfzBRaVlzT9BMAJqc2+iYXZYmbrEr6FqLd+vNWdQh LPuCi9VffmKPiYdU+aa1ziGOPrByk+VqZ8XPwhRQZfqA\"},{\"name\":\"X-Google-Smtp-Source\",\"value\":\"AK7set8pXO/RqbyoqOESJPr8IWGbCTnfvowIa5MsDmDCCZwED6lgsH8iru4WddjFLwV6XbGC0vn6RWJAEHPcDinJsko=\"},{\"name\":\"X-Received\",\"value\":\"by 2002:a1f:cec4:0:b0:40e:fee9:667a with SMTP id e187-20020a1fcec4000000b0040efee9667amr966128vkg.3.1677839118135; Fri, 03 Mar 2023 02:25:18 -0800 (PST)\"},{\"name\":\"MIME-Version\",\"value\":\"1.0\"},{\"name\":\"From\",\"value\":\"Meno Abels <from@smtp.world>\"},{\"name\":\"Date\",\"value\":\"Fri, 3 Mar 2023 11:25:07 +0100\"},{\"name\":\"Message-ID\",\"value\":\"<CAPpNkKDm59_UihXCS21B1joBPHUGqpsVDwwCtXKaJ31t12bu=A@mail.gmail.com>\"},{\"name\":\"Subject\",\"value\":\"hallo\"},{\"name\":\"To\",\"value\":\"to@smtp.world\"},{\"name\":\"Content-Type\",\"value\":\"multipart/alternative; boundary=\\\"000000000000a0746805f5fc5c6c\\\"\"}],\"commonHeaders\":{\"returnPath\":\"from@smtp.world\",\"from\":[\"Meno Abels <from@smtp.world>\"],\"date\":\"Fri, 3 Mar 2023 11:25:07 +0100\",\"to\":[\"to@smtp.world\"],\"messageId\":\"<CAPpNkKDm59_UihXCS21B1joBPHUGqpsVDwwCtXKaJ31t12bu=A@mail.gmail.com>\",\"subject\":\"hallo\"}},\"receipt\":{\"timestamp\":\"2023-03-03T10:25:18.793Z\",\"processingTimeMillis\":751,\"recipients\":[\"to@smtp.world\"],\"spamVerdict\":{\"status\":\"PASS\"},\"virusVerdict\":{\"status\":\"PASS\"},\"spfVerdict\":{\"status\":\"PASS\"},\"dkimVerdict\":{\"status\":\"PASS\"},\"dmarcVerdict\":{\"status\":\"PASS\"},\"action\":{\"type\":\"S3\",\"topicArn\":\"arn:aws:sns:us-east-1:XXXXXXX:smtp-q\",\"bucketName\":\"adviser-smtp-q\",\"objectKey\":\"nrk5vlqu9usuh476ffj0j3is23okmot9h029da01\"}}}",
		Timestamp:        time.Now().Format(time.RFC3339),
		SignatureVersion: "1",
		Signature:        "iWyTC5N4JaIIwBZgKQJ+Whzk4aOd+Iu0O+ubwVRWJsstlkrWE/v2n+vjcapLMGa4n98JmnCyMGMwoa3LiR17MOD5r+ScW4zaskShQzSpV3454xggPNy24DcwQz2UlUjSoawUxkkfgBrvjcScEx229W5k2Cm36S9WWRnxQ8ZXkVO0MJNwHK02/mnsXokGVMkIml8b4uKvO+9KaPmtYgLBx3SnUzS2SSyOdS+cYjPBwWu4eEeol29hGEkGJ5IjX44ANlG4mTcz5ZPaolD82qjXXCo7YHvFiOiKrvyDU8BS2tsy8pvxxyCWfYKNkDvmd7auQfVBzk7FwXMyIchmXRw3ew==",
		SigningCertURL:   "https://sns.us-east-1.amazonaws.com/SimpleNotificationService-56e67fcb41f6fec09b0196692625d385.pem",
		UnsubscribeURL:   "https://sns.us-east-1.amazonaws.com/?Action=Unsubscribe&SubscriptionArn=arn:aws:sns:us-east-1:XXXXXXX:smtp-q:f88c920b-6a88-450d-9319-471bcffb5c2d",
	}
	jsonByte, _ := json.Marshal(ses)
	return sqsTypes.Message{
		MessageId:     strPtr(uuid.NewString()),
		Body:          strPtr(string(jsonByte)),
		ReceiptHandle: strPtr(uuid.NewString()),
	}
}

func setupAsn(obs *AwsSesObserver) *RetryAwsSesNotification {
	msg := sqsMessage()
	var asm AwsSesMessage
	json.Unmarshal([]byte(*msg.Body), &asm)
	var asn RetryAwsSesNotification
	json.Unmarshal([]byte(asm.Message), &asn)
	return &asn
}

func TestSendMailOK(t *testing.T) {
	cli := *FlagCliArgs
	cli.EnableStr = "true"
	cli.SQS.Name = "testQ"
	cli.Bucket.Name = "bucket"
	cli.Bucket.KeyPrefix = "prefix/"
	cli.Smtp.Host = "host"
	cli.Smtp.Identity = "identity"
	cli.Smtp.User = "user"
	cli.Smtp.Pass = "pass"
	cfg, err := ConfigureObserver(cli)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	obs, err := mockNewAWSSESObserver(cfg)
	if err != nil {
		t.Error(err)
	}
	asn := setupAsn(obs)
	asn.Receipt.Recipients = []string{
		"to@smtp.world",
		"kaputt@smtp.world",
	}
	rcpt, retry, err, _ := obs.sendMail(asn, s3GetObjectOutput("testBody"))
	if len(rcpt) != 1 {
		t.Errorf("Expected 1 recipient, got %d", len(rcpt))
	}
	if rcpt[0] != "to@smtp.world" {
		t.Errorf("Unexpected recipient: %v", rcpt[0])
	}
	if retry {
		t.Error("Unexpected retry")
	}
	if err != nil {
		t.Error("Expected no error", err)
	}
}

func TestSendMailFromDefectAsnEmptyFrom(t *testing.T) {
	cli := *FlagCliArgs
	cli.EnableStr = "true"
	cli.SQS.Name = "testQ"
	cli.Bucket.Name = "bucket"
	cli.Bucket.KeyPrefix = "prefix/"
	cli.Smtp.Host = "host"
	cli.Smtp.Identity = "identity"
	cli.Smtp.User = "user"
	cli.Smtp.Pass = "pass"
	cfg, err := ConfigureObserver(cli)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	obs, err := mockNewAWSSESObserver(cfg)
	if err != nil {
		t.Error(err)
	}
	asn := setupAsn(obs)
	asn.Mail.CommonHeaders.From = []string{}
	asn.Mail.CommonHeaders.ReturnPath = ""
	asn.Receipt.Recipients = []string{
		"to@smtp.world",
		"kaputt@smtp.world",
	}
	_, _, err, _ = obs.sendMail(asn, s3GetObjectOutput("testBody"))
	if err.Error() != "no from address found" {
		t.Error("Expected no error", err)
	}
}

func TestSendMailFromDefectAsnFallReturnPath(t *testing.T) {
	cli := *FlagCliArgs
	cli.EnableStr = "true"
	cli.SQS.Name = "testQ"
	cli.Bucket.Name = "bucket"
	cli.Bucket.KeyPrefix = "prefix/"
	cli.Smtp.Host = "host"
	cli.Smtp.Identity = "identity"
	cli.Smtp.User = "user"
	cli.Smtp.Pass = "pass"
	cfg, err := ConfigureObserver(cli)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	obs, err := mockNewAWSSESObserver(cfg)
	if err != nil {
		t.Error(err)
	}
	obs.Smtp.(*mockSMTP).smtpClient = &mockSMTPClient{
		mailErrFn: func(from string) error {
			if from == "return@path" {
				return nil
			}
			return fmt.Errorf("543 5.7.1 not expected from address")
		},
	}
	asn := setupAsn(obs)
	asn.Mail.CommonHeaders.From = []string{}
	asn.Mail.CommonHeaders.ReturnPath = "return@path"
	asn.Receipt.Recipients = []string{
		"to@smtp.world",
		"kaputt@smtp.world",
	}
	_, _, err, _ = obs.sendMail(asn, s3GetObjectOutput("testBody"))
	if err != nil {
		t.Error("Expected no error", err)
	}
}

func TestSendMailFromDefectAsnFallEmptyFrom(t *testing.T) {
	cli := *FlagCliArgs
	cli.EnableStr = "true"
	cli.SQS.Name = "testQ"
	cli.Bucket.Name = "bucket"
	cli.Bucket.KeyPrefix = "prefix/"
	cli.Smtp.Host = "host"
	cli.Smtp.Identity = "identity"
	cli.Smtp.User = "user"
	cli.Smtp.Pass = "pass"
	cfg, err := ConfigureObserver(cli)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	obs, err := mockNewAWSSESObserver(cfg)
	if err != nil {
		t.Error(err)
	}
	obs.Smtp.(*mockSMTP).smtpClient = &mockSMTPClient{
		mailErrFn: func(from string) error {
			if from == "return@path" {
				return nil
			}
			return fmt.Errorf("543 5.7.1 not expected from address")
		},
	}
	asn := setupAsn(obs)
	asn.Mail.CommonHeaders.From = []string{""}
	asn.Mail.CommonHeaders.ReturnPath = "return@path"
	asn.Receipt.Recipients = []string{
		"to@smtp.world",
		"kaputt@smtp.world",
	}
	_, _, err, _ = obs.sendMail(asn, s3GetObjectOutput("testBody"))
	if err != nil {
		t.Error("Expected no error", err)
	}
}

func TestSendMailFailedButOk(t *testing.T) {
	cli := *FlagCliArgs
	cli.EnableStr = "true"
	cli.SQS.Name = "testQ"
	cli.Bucket.Name = "bucket"
	cli.Bucket.KeyPrefix = "prefix/"
	cli.Smtp.Host = "host"
	cli.Smtp.Identity = "identity"
	cli.Smtp.User = "user"
	cli.Smtp.Pass = "pass"
	cfg, err := ConfigureObserver(cli)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	obs, err := mockNewAWSSESObserver(cfg)
	if err != nil {
		t.Error(err)
	}
	asn := setupAsn(obs)
	asn.Receipt.Recipients = []string{
		"kaputt@smtp.world",
	}
	rcpt, retry, err, _ := obs.sendMail(asn, s3GetObjectOutput("testBody"))
	if len(rcpt) != 0 {
		t.Errorf("Expected 0 recipient, got %d", len(rcpt))
	}
	if retry {
		t.Error("Unexpected retry")
	}
	if err == nil {
		t.Error("Expected no error", err)
	}
}

func TestSendMailFailed500er(t *testing.T) {
	cli := *FlagCliArgs
	cli.EnableStr = "true"
	cli.SQS.Name = "testQ"
	cli.Bucket.Name = "bucket"
	cli.Bucket.KeyPrefix = "prefix/"
	cli.Smtp.Host = "host"
	cli.Smtp.Identity = "identity"
	cli.Smtp.User = "user"
	cli.Smtp.Pass = "pass"
	cfg, err := ConfigureObserver(cli)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	obs, err := mockNewAWSSESObserver(cfg)
	if err != nil {
		t.Error(err)
	}
	asn := setupAsn(obs)
	asn.Mail.CommonHeaders.From = []string{
		"kaputt@smtp.world",
	}
	obs.Smtp.(*mockSMTP).smtpClient = &mockSMTPClient{
		mailErrFn: func(_ string) error { return fmt.Errorf("543 5.7.1 fake error") },
	}
	_, retry, err, _ := obs.sendMail(asn, s3GetObjectOutput("testBody"))
	if retry {
		t.Error("Unexpected retry")
	}
	if err.Error() != "543 5.7.1 fake error" {
		t.Error("Expected no error", err)
	}
}

func TestSendMailFailed400er(t *testing.T) {
	cli := *FlagCliArgs
	cli.EnableStr = "true"
	cli.SQS.Name = "testQ"
	cli.Bucket.Name = "bucket"
	cli.Bucket.KeyPrefix = "prefix/"
	cli.Smtp.Host = "host"
	cli.Smtp.Identity = "identity"
	cli.Smtp.User = "user"
	cli.Smtp.Pass = "pass"
	cfg, err := ConfigureObserver(cli)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	obs, err := mockNewAWSSESObserver(cfg)
	if err != nil {
		t.Error(err)
	}
	asn := setupAsn(obs)
	asn.Mail.CommonHeaders.From = []string{
		"kaputt@smtp.world",
	}
	obs.Smtp.(*mockSMTP).smtpClient = &mockSMTPClient{
		mailErrFn: func(_ string) error { return fmt.Errorf("443 4.7.1 fake error") },
	}
	_, retry, err, _ := obs.sendMail(asn, s3GetObjectOutput("testBody"))
	if !retry {
		t.Error("Unexpected retry")
	}
	if err.Error() != "443 4.7.1 fake error" {
		t.Error("Expected no error", err)
	}
}

func TestObserver(t *testing.T) {
	cli := *FlagCliArgs
	cli.EnableStr = "true"
	cli.SQS.Name = "testQ"
	cli.Bucket.Name = "bucket"
	cli.Bucket.KeyPrefix = "prefix/"
	cli.Smtp.Host = "host"
	cli.Smtp.Identity = "identity"
	cli.Smtp.User = "user"
	cli.Smtp.Pass = "pass"
	cfg, err := ConfigureObserver(cli)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	obs, err := mockNewAWSSESObserver(cfg)
	if err != nil {
		t.Error(err)
	}
	err = obs.Observe(1)
	if err != nil {
		t.Error(err)
	}
}

func getMsgAndNotification(msg sqsTypes.Message) (AwsSesMessage, RetryAwsSesNotification, error) {
	asm := AwsSesMessage{}
	err := json.Unmarshal([]byte(*msg.Body), &asm)
	if err != nil {
		return asm, RetryAwsSesNotification{}, err
	}
	if asm.Type != "Notification" {
		return asm, RetryAwsSesNotification{}, fmt.Errorf("Expected Notification, got %s", asm.Type)
	}
	asn := RetryAwsSesNotification{}
	err = json.Unmarshal([]byte(asm.Message), &asn)
	return asm, asn, err
}

func TestObserverFailedRetry(t *testing.T) {
	cli := *FlagCliArgs
	cli.EnableStr = "true"
	cli.SQS.Name = "testQ"
	cli.Bucket.Name = "bucket"
	cli.Bucket.KeyPrefix = "prefix/"
	cli.Smtp.Host = "host"
	cli.Smtp.Identity = "identity"
	cli.Smtp.User = "user"
	cli.Smtp.Pass = "pass"
	cfg, err := ConfigureObserver(cli)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	obs, err := mockNewAWSSESObserver(cfg)
	if err != nil {
		t.Error(err)
	}
	mailErrCnt := 0
	obs.Smtp.(*mockSMTP).smtpClient = &mockSMTPClient{
		mailErrFn: func(_ string) error {
			mailErrCnt++
			if mailErrCnt <= cli.RetryCount {
				return fmt.Errorf("443 4.7.1 fake error")
			}
			return nil
		},
	}
	err = obs.Observe(cli.RetryCount + 1)
	if err != nil {
		t.Error(err)
	}
	my := obs.SQS.Client.(*mockSQSClient)
	if len(my.nextMessages) != 0 {
		out, _ := json.Marshal(my.nextMessages)
		t.Errorf("Expected no delete message:%v", out)
	}
	if len(my.receivedMessages) != cli.RetryCount+1 {
		t.Errorf("Expected %d messages, got %d", cli.RetryCount, len(my.receivedMessages))
	}
	for i, msg := range my.receivedMessages {
		_, asn, err := getMsgAndNotification(msg)
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if asn.RetryCount != i%cli.RetryCount {
			t.Errorf("Expected %d, got %d", i, asn.RetryCount)
		}
	}
	if len(my.sendMessages) != cli.RetryCount-1 {
		t.Errorf("Expected %d messages, got %d", cli.RetryCount, len(my.receivedMessages))
	}
	for i, msg := range my.sendMessages {
		_, asn, err := getMsgAndNotification(msg)
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if asn.RetryCount != i+1 {
			t.Errorf("Expected %d, got %d", i, asn.RetryCount)
		}
	}
}

func TestNotEnabledObserver(t *testing.T) {
	cli := *FlagCliArgs
	_, err := ConfigureObserver(cli)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
}

func TestEnabledObserver(t *testing.T) {
	cli := *FlagCliArgs
	cli.EnableStr = "true"
	_, err := ConfigureObserver(cli)
	if err == nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestConfiguredDefault(t *testing.T) {
	cli := *FlagCliArgs
	cli.EnableStr = "true"
	cli.SQS.Name = "queueName"
	cli.Bucket.Name = "bucket"
	cli.Smtp.Host = "host"
	cli.Smtp.Pass = "pass"
	obs, err := ConfigureObserver(cli)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if obs.SQS.Name != "queueName" {
		t.Errorf("Unexpected queue name: %s", obs.SQS.Name)
	}
	if obs.SQS.Timeout != 10 {
		t.Errorf("Unexpected timeout: %d", obs.SQS.Timeout)
	}
	if obs.SQS.MaxMessages != 10 {
		t.Errorf("Unexpected max messages: %d", obs.SQS.MaxMessages)
	}
	if obs.Bucket.Name != "bucket" {
		t.Errorf("Unexpected bucket: %s", obs.Bucket.Name)
	}
	if obs.Bucket.KeyPrefix != "" {
		t.Errorf("Unexpected KeyPrefix: %s", obs.Bucket.KeyPrefix)
	}
	if obs.Smtp.Host != "host" {
		t.Errorf("Unexpected SMTP host: %s", obs.Smtp.Host)
	}
	if obs.Smtp.Port != 25 {
		t.Errorf("Unexpected SMTP port: %d", obs.Smtp.Port)
	}
	if obs.Smtp.Pass != "pass" {
		t.Errorf("Unexpected SMTP pass: %s", obs.Smtp.Pass)
	}
	if obs.Smtp.ConnectionTLS() != false {
		t.Errorf("Unexpected SMTP connection TLS: %t", obs.Smtp.ConnectionTLS())
	}
	if obs.Smtp.ForceSTARTTLS() != true {
		t.Errorf("Unexpected SMTP force STARTTLS: %t", obs.Smtp.ForceSTARTTLS())
	}
	if obs.Smtp.InsecureTLS() != true {
		t.Errorf("Unexpected SMTP insecure TLS: %t", obs.Smtp.InsecureTLS())
	}
	if obs.Smtp.Identity != "" {
		t.Errorf("Unexpected SMTP identity: %s", obs.Smtp.Identity)
	}
	if obs.Smtp.MyName != "AWS-SMTP-Relay-Observer" {
		t.Errorf("Unexpected SMTP my name: %s", obs.Smtp.MyName)
	}
}

func TestConfiguredSet(t *testing.T) {
	cli := *FlagCliArgs
	cli.EnableStr = "true"
	cli.SQS.Name = "queueName"
	cli.Bucket.Name = "bucket"
	cli.Bucket.KeyPrefix = "prefix"
	cli.Smtp.Host = "host"
	cli.Smtp.Port = 27
	cli.Smtp.ConnectionTLSStr = "true"
	cli.Smtp.ForceSTARTTLSStr = "false"
	cli.Smtp.InsecureTLSStr = "false"
	cli.Smtp.MyName = "myName"
	cli.Smtp.Identity = "identity"
	os.Setenv("QUEUE_SMTP_PASS", "pass")
	obs, err := ConfigureObserver(cli)
	os.Unsetenv("QUEUE_SMTP_PASS")
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if obs.SQS.Name != "queueName" {
		t.Errorf("Unexpected queue name: %s", obs.SQS.Name)
	}
	if obs.SQS.Timeout != 10 {
		t.Errorf("Unexpected timeout: %d", obs.SQS.Timeout)
	}
	if obs.SQS.MaxMessages != 10 {
		t.Errorf("Unexpected max messages: %d", obs.SQS.MaxMessages)
	}
	if obs.Bucket.Name != "bucket" {
		t.Errorf("Unexpected bucket: %s", obs.Bucket.Name)
	}
	if obs.Bucket.KeyPrefix != "prefix" {
		t.Errorf("Unexpected KeyPrefix: %s", obs.Bucket.KeyPrefix)
	}
	if obs.Smtp.Host != "host" {
		t.Errorf("Unexpected SMTP host: %s", obs.Smtp.Host)
	}
	if obs.Smtp.Port != 27 {
		t.Errorf("Unexpected SMTP port: %d", obs.Smtp.Port)
	}
	if obs.Smtp.Pass != "pass" {
		t.Errorf("Unexpected SMTP pass: %s", obs.Smtp.Pass)
	}
	if obs.Smtp.ConnectionTLS() != true {
		t.Errorf("Unexpected SMTP connection TLS: %t", obs.Smtp.ConnectionTLS())
	}
	if obs.Smtp.ForceSTARTTLS() != false {
		t.Errorf("Unexpected SMTP force STARTTLS: %t", obs.Smtp.ForceSTARTTLS())
	}
	if obs.Smtp.InsecureTLS() != false {
		t.Errorf("Unexpected SMTP insecure TLS: %t", obs.Smtp.InsecureTLS())
	}
	if obs.Smtp.Identity != "identity" {
		t.Errorf("Unexpected SMTP identity: %s", obs.Smtp.Identity)
	}
	if obs.Smtp.MyName != "myName" {
		t.Errorf("Unexpected SMTP my name: %s", obs.Smtp.MyName)
	}
}

type gosmtpSMTP struct {
	smtpClient SMTPClient
}

func (s *gosmtpSMTP) Dial(addr string) (SMTPClient, error) {
	return smtp.Dial(addr)
}

func (s *gosmtpSMTP) DialTLS(addr string, tls *tls.Config) (SMTPClient, error) {
	return smtp.DialTLS(addr, tls)
}

type mockSesClient struct {
}

func (m *mockSesClient) SendRawEmail(_ context.Context, mi *awsSes.SendRawEmailInput, opts ...func(*awsSes.Options)) (*awsSes.SendRawEmailOutput, error) {
	return nil, nil
}

func startSMTPServerTest(fn func(*smtp.Server, net.Listener)) error {
	certFile, keyFile, deferFn, err := test_utils.GenerateX509()
	if err != nil {
		return fmt.Errorf("Unexpected error: %s", err)
	}
	defer deferFn()
	cfg, err := config.Configure(config.Config{
		Addr:       "127.0.0.1:0",
		User:       "user",
		BcryptHash: []byte("pass"),
		CertFile:   certFile,
		KeyFile:    keyFile,
	})
	if err != nil {
		return err
	}
	return server.StartSMTPServer(*cfg, nil, fn, func(be interface{}) {
		backend, ok := be.(*server.Backend)
		if !ok || backend == nil {
			return
		}
		clt := backend.Client.Annotate(&ses.Client{
			SesClient: &mockSesClient{},
		})
		backend.Client = clt
	})
}

func TestRealSmtp(t *testing.T) {
	err := startSMTPServerTest(func(srv *smtp.Server, lsr net.Listener) {
		cli := *FlagCliArgs
		cli.EnableStr = "true"
		cli.SQS.Name = "testQ"
		cli.Bucket.Name = "bucket"
		cli.Bucket.KeyPrefix = "prefix/"
		cli.Smtp.Host, _ = server.SplitAddr(srv.Addr)
		cli.Smtp.Port = lsr.Addr().(*net.TCPAddr).Port
		cli.Smtp.Identity = "identity"
		cli.Smtp.User = "user"
		cli.Smtp.Pass = "pass"
		cfg, err := ConfigureObserver(cli)
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		obs, err := mockNewAWSSESObserver(cfg)
		if err != nil {
			t.Error(err)
		}

		obs.Smtp = &gosmtpSMTP{}
		asn := setupAsn(obs)
		asn.Receipt.Recipients = []string{
			"to@smtp.world",
			"kaputt@smtp.world",
		}
		preFileId := test_utils.GetNextFileDescriptor()
		rcpt, retry, err, _ := obs.sendMail(asn, s3GetObjectOutput("testBody"))
		if test_utils.GetNextFileDescriptor() != preFileId {
			t.Errorf("File descriptor leak: %d", test_utils.GetNextFileDescriptor())
		}
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if retry != false {
			t.Errorf("Unexpected retry: %t", retry)
		}
		if len(rcpt) != 2 {
			t.Errorf("Unexpected recipient count: %d", len(rcpt))
		}
	})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
}

func TestRealSmtpFromFail(t *testing.T) {
	err := startSMTPServerTest(func(srv *smtp.Server, lsr net.Listener) {
		cli := *FlagCliArgs
		cli.EnableStr = "true"
		cli.SQS.Name = "testQ"
		cli.Bucket.Name = "bucket"
		cli.Bucket.KeyPrefix = "prefix/"
		cli.Smtp.Host = "127.0.0.1"
		cli.Smtp.Port = lsr.Addr().(*net.TCPAddr).Port
		cli.Smtp.Identity = "identity"
		cli.Smtp.User = "user"
		cli.Smtp.Pass = "pass"
		cfg, err := ConfigureObserver(cli)
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		obs, err := mockNewAWSSESObserver(cfg)
		if err != nil {
			t.Error(err)
		}

		obs.Smtp = &gosmtpSMTP{}
		asn := setupAsn(obs)
		asn.Receipt.Recipients = []string{
			"to@smtp.world",
			"kaputt@smtp.world",
		}
		asn.Mail.CommonHeaders.From = []string{"<from@kaputt"}
		preFileId := test_utils.GetNextFileDescriptor()
		_, _, err, _ = obs.sendMail(asn, s3GetObjectOutput("testBody"))
		if test_utils.GetNextFileDescriptor() != preFileId {
			t.Errorf("File descriptor leak: %d", test_utils.GetNextFileDescriptor())
		}
		if err == nil {
			t.Errorf("Unexpected error: %s", err)
		}
	})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
}
