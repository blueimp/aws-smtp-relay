package relay_test

import (
	"encoding/json"
	"io/ioutil"
	"net"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/pinpointemail"
	"github.com/aws/aws-sdk-go/service/pinpointemail/pinpointemailiface"
	"github.com/blueimp/aws-smtp-relay/internal/relay"
	internal "github.com/blueimp/aws-smtp-relay/internal/relay/pinpoint"
)

var testData = struct{ input *pinpointemail.SendEmailInput }{}

type mockPinpointEmailClient struct {
	pinpointemailiface.PinpointEmailAPI
}
func (m *mockPinpointEmailClient) CreateConfigurationSet(input *pinpointemail.CreateConfigurationSetInput) (*pinpointemail.CreateConfigurationSetOutput, error) {
   	return &pinpointemail.CreateConfigurationSetOutput{}, nil
}

func (m *mockPinpointEmailClient) SendEmail(input *pinpointemail.SendEmailInput) (
	*pinpointemail.SendEmailOutput,
	error,
) {
	testData.input = input
	return nil, nil
}

func sendHelper(
	origin net.Addr,
	from *string,
	to *[]string,
	data *[]byte,
	setName *string,
) (email *pinpointemail.SendEmailInput, out []byte, err []byte) {
	outReader, outWriter, _ := os.Pipe()
	errReader, errWriter, _ := os.Pipe()
	originalOut := os.Stdout
	originalErr := os.Stderr
	defer func() {
		testData.input = nil
		os.Stdout = originalOut
		os.Stderr = originalErr
	}()
	os.Stdout = outWriter
	os.Stderr = errWriter
	func() {
		internal.Send(&mockPinpointEmailClient{}, origin, from, to, data, setName)
		outWriter.Close()
		errWriter.Close()
	}()
	stdout, _ := ioutil.ReadAll(outReader)
	stderr, _ := ioutil.ReadAll(errReader)
	return testData.input, stdout, stderr
}

func TestSend(t *testing.T) {
	origin := net.TCPAddr{IP: []byte{127, 0, 0, 1}}
	from := "alice@example.org"
	to := []string{"bob@example.org"}
	data := []byte{'T', 'E', 'S', 'T'}
	setName := ""
	timeBefore := time.Now()
	input, out, err := sendHelper(&origin, &from, &to, &data, &setName)
	timeAfter := time.Now()
	if *input.FromEmailAddress != from {
		t.Errorf(
			"Unexpected source: %s. Expected: %s",
			*input.FromEmailAddress,
			from,
		)
	}
	if *input.Destination.ToAddresses[0] != to[0] {
		t.Errorf(
			"Unexpected destination: %s. Expected: %s",
			*input.Destination.ToAddresses[0],
			to[0],
		)
	}
	inputData := string(input.Content.Raw.Data)
	if inputData != "TEST" {
		t.Errorf("Unexpected data: %s. Expected: %s", inputData, "TEST")
	}
	var req relay.Request
	json.Unmarshal(out, &req)
	if req.Time.Before(timeBefore) {
		t.Errorf("Unexpected 'Time' log: %s", req.Time)
	}
	if req.Time.After(timeAfter) {
		t.Errorf("Unexpected 'Time' log: %s", req.Time)
	}
	if req.IP != "127.0.0.1" {
		t.Errorf("Unexpected 'IP' log: %s. Expected: %s", req.IP, "127.0.0.1")
	}
	if req.From != from {
		t.Errorf("Unexpected 'From' log: %s. Expected: %s", req.From, from)
	}
	if req.To[0] != to[0] {
		t.Errorf("Unexpected 'To' log: %s. Expected: %s", req.To, to)
	}
	if req.Error != "" {
		t.Errorf("Unexpected 'Error' log: %s. Expected: %s", req.Error, "")
	}
	if len(err) != 0 {
		t.Errorf("Unexpected stderr: %s", err)
	}
}

func TestSendWithOriginIPv6(t *testing.T) {
	origin := net.TCPAddr{IP: []byte{
		0x20, 0x01, 0x48, 0x60, 0, 0, 0x20, 0x01, 0, 0, 0, 0, 0, 0, 0x00, 0x68,
	}}
	from := "alice@example.org"
	to := []string{"bob@example.org"}
	data := []byte{'T', 'E', 'S', 'T'}
	setName := ""
	_, out, err := sendHelper(&origin, &from, &to, &data, &setName)
	var req relay.Request
	json.Unmarshal(out, &req)
	if req.IP != "2001:4860:0:2001::68" {
		t.Errorf(
			"Unexpected 'IP' log: %s. Expected: %s",
			req.IP,
			"2001:4860:0:2001::68",
		)
	}
	if req.Error != "" {
		t.Errorf("Unexpected 'Error' log: %s. Expected: %s", req.Error, "")
	}
	if len(err) != 0 {
		t.Errorf("Unexpected stderr: %s", err)
	}
}

func TestSendWithMultipleRecipients(t *testing.T) {
	origin := net.TCPAddr{IP: []byte{127, 0, 0, 1}}
	from := "alice@example.org"
	to := []string{"bob@example.org", "charlie@example.org"}
	data := []byte{'T', 'E', 'S', 'T'}
	setName := ""
	input, out, err := sendHelper(&origin, &from, &to, &data, &setName);
	if len(input.Destination.ToAddresses) != 2 {
		t.Errorf(
			"Unexpected number of destinations: %d. Expected: %d",
			len(input.Destination.ToAddresses),
			2,
		)
	}
	if *input.Destination.ToAddresses[0] != to[0] {
		t.Errorf(
			"Unexpected destination: %s. Expected: %s",
			*input.Destination.ToAddresses[0],
			to[0],
		)
	}
	var req relay.Request
	json.Unmarshal(out, &req)
	if req.To[0] != to[0] {
		t.Errorf("Unexpected 'To' log: %s. Expected: %s", req.To, to)
	}
	if req.Error != "" {
		t.Errorf("Unexpected 'Error' log: %s. Expected: %s", req.Error, "")
	}
	if len(err) != 0 {
		t.Errorf("Unexpected stderr: %s", err)
	}
}
