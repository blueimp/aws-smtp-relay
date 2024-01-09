package receiver

import (
	"fmt"
	"strings"
	"time"
)

type AwsSesMessage struct {
	Type             string
	MessageId        string
	TopicArn         string
	Subject          string
	Message          string
	Timestamp        string
	SignatureVersion string
	Signature        string
	SigningCertURL   string
	UnsubscribeURL   string
}

type AwsSESStatus struct {
	Status string `json:"status"`
}
type AwsSESHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type AWSJsonTime struct {
	time.Time
}

func (ct *AWSJsonTime) UnmarshalJSON(b []byte) (err error) {
	s := strings.Trim(string(b), "\"")
	if s == "null" {
		ct.Time = time.Time{}
		return
	}
	ct.Time, err = time.Parse(time.RFC3339, s)
	return
}

func (ct *AWSJsonTime) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%s\"", ct.Time.Format(time.RFC3339))), nil
}

type AwsSesNotification struct {
	NotificationType string `json:"notificationType"`
	Mail             struct {
		Timestamp        AWSJsonTime    `json:"timestamp"`
		Source           string         `json:"source"`
		MessageId        string         `json:"messageId"`
		Destination      []string       `json:"destination"`
		HeadersTruncated bool           `json:"headersTruncated"`
		Headers          []AwsSESHeader `json:"headers"`
		CommonHeaders    struct {
			ReturnPath string   `json:"returnPath"`
			From       []string `json:"from"`
			Date       string   `json:"date"`
			To         []string `json:"to"`
			MessageId  string   `json:"messageId"`
			Subject    string   `json:"subject"`
		} `json:"commonHeaders"`
	} `json:"mail"`
	Receipt struct {
		Timestamp            AWSJsonTime  `json:"timestamp"`
		ProcessingTimeMillis uint64       `json:"processingTimeMillis"`
		Recipients           []string     `json:"recipients"`
		SpamVerdict          AwsSESStatus `json:"spamVerdict"`
		VirusVerdict         AwsSESStatus `json:"virusVerdict"`
		SpfVerdict           AwsSESStatus `json:"spfVerdict"`
		DkimVerdict          AwsSESStatus `json:"dkimVerdict"`
		DmarcVerdict         AwsSESStatus `json:"dmarcVerdict"`
		Action               AwsSESStatus `json:"action"`
	} `json:"receipt"`
}
