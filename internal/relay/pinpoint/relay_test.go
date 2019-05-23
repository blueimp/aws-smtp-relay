package relay

import (
	"io/ioutil"
	"net"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/service/pinpointemail"
	"github.com/aws/aws-sdk-go/service/pinpointemail/pinpointemailiface"
)

var testData = struct{ input *pinpointemail.SendEmailInput }{}

type mockPinpointEmailClient struct {
	pinpointemailiface.PinpointEmailAPI
}

func (m *mockPinpointEmailClient) CreateConfigurationSet(
	input *pinpointemail.CreateConfigurationSetInput,
) (*pinpointemail.CreateConfigurationSetOutput, error) {
	return &pinpointemail.CreateConfigurationSetOutput{}, nil
}

func (m *mockPinpointEmailClient) SendEmail(
	input *pinpointemail.SendEmailInput,
) (*pinpointemail.SendEmailOutput, error) {
	testData.input = input
	return nil, nil
}

func sendHelper(
	origin net.Addr,
	from string,
	to []string,
	data []byte,
	configurationSetName *string,
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
		c := Client{
			pinpointAPI: &mockPinpointEmailClient{},
			setName:     configurationSetName,
		}
		c.Send(origin, from, to, data)
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
	input, out, err := sendHelper(&origin, from, to, data, &setName)
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
	if len(out) == 0 {
		t.Error("Unexpected empty stdout")
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
	input, out, err := sendHelper(&origin, from, to, data, &setName)
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
	if len(out) == 0 {
		t.Error("Unexpected empty stdout")
	}
	if len(err) != 0 {
		t.Errorf("Unexpected stderr: %s", err)
	}
}
