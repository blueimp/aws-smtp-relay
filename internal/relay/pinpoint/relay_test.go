package pinpoint

import (
	"context"
	"errors"
	"io/ioutil"
	"net"
	"os"
	"reflect"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/pinpointemail"
	"github.com/blueimp/aws-smtp-relay/internal/relay/filter"
)

var testData = struct {
	input *pinpointemail.SendEmailInput
	err   error
}{}

type mockPinpointEmailClient struct {
}

func (m *mockPinpointEmailClient) SendEmail(
	context context.Context,
	input *pinpointemail.SendEmailInput,
	fns ...func(*pinpointemail.Options)) (*pinpointemail.SendEmailOutput, error) {
	testData.input = input
	return nil, testData.err
}

func sendHelper(
	origin net.Addr,
	from string,
	to []string,
	data []byte,
	configurationSetName *string,
	allowFromRegExp *regexp.Regexp,
	denyToRegExp *regexp.Regexp,
	apiErr error,
) (email *pinpointemail.SendEmailInput, out []byte, err []byte, sendErr error) {
	outReader, outWriter, _ := os.Pipe()
	errReader, errWriter, _ := os.Pipe()
	originalOut := os.Stdout
	originalErr := os.Stderr
	defer func() {
		testData.input = nil
		testData.err = nil
		os.Stdout = originalOut
		os.Stderr = originalErr
	}()
	os.Stdout = outWriter
	os.Stderr = errWriter
	func() {
		c := Client{
			pinpointClient:  &mockPinpointEmailClient{},
			setName:         configurationSetName,
			allowFromRegExp: allowFromRegExp,
			denyToRegExp:    denyToRegExp,
		}
		testData.err = apiErr
		sendErr = c.Send(origin, from, to, data)
		outWriter.Close()
		errWriter.Close()
	}()
	stdout, _ := ioutil.ReadAll(outReader)
	stderr, _ := ioutil.ReadAll(errReader)
	return testData.input, stdout, stderr, sendErr
}

func TestSend(t *testing.T) {
	origin := net.TCPAddr{IP: []byte{127, 0, 0, 1}}
	from := "alice@example.org"
	to := []string{"bob@example.org"}
	data := []byte{'T', 'E', 'S', 'T'}
	setName := ""
	input, out, err, _ := sendHelper(&origin, from, to, data, &setName, nil, nil, nil)
	if *input.FromEmailAddress != from {
		t.Errorf(
			"Unexpected source: %s. Expected: %s",
			*input.FromEmailAddress,
			from,
		)
	}
	if len(input.Destination.ToAddresses) != 1 {
		t.Errorf(
			"Unexpected number of destinations: %d. Expected: %d",
			len(input.Destination.ToAddresses),
			1,
		)
	}
	if input.Destination.ToAddresses[0] != to[0] {
		t.Errorf(
			"Unexpected destination: %s. Expected: %s",
			input.Destination.ToAddresses[0],
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
	input, out, err, _ := sendHelper(&origin, from, to, data, &setName, nil, nil, nil)
	if len(input.Destination.ToAddresses) != 2 {
		t.Errorf(
			"Unexpected number of destinations: %d. Expected: %d",
			len(input.Destination.ToAddresses),
			2,
		)
	}
	if input.Destination.ToAddresses[0] != to[0] {
		t.Errorf(
			"Unexpected destination: %s. Expected: %s",
			input.Destination.ToAddresses[0],
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

func TestSendWithDeniedSender(t *testing.T) {
	origin := net.TCPAddr{IP: []byte{127, 0, 0, 1}}
	from := "alice@example.org"
	to := []string{"bob@example.org", "charlie@example.org"}
	data := []byte{'T', 'E', 'S', 'T'}
	setName := ""
	regexp, _ := regexp.Compile(`^admin@example\.org$`)
	input, out, err, sendErr := sendHelper(&origin, from, to, data, &setName, regexp, nil, nil)
	if input != nil {
		t.Errorf(
			"Unexpected number of destinations: %d. Expected: %d",
			len(input.Destination.ToAddresses),
			0,
		)
	}
	if sendErr != filter.ErrDeniedSender {
		t.Errorf("Unexpected error: %s. Expected: %s", sendErr, filter.ErrDeniedSender)
	}
	if len(out) == 0 {
		t.Error("Unexpected empty stdout")
	}
	if len(err) != 0 {
		t.Errorf("Unexpected stderr: %s", err)
	}
}

func TestSendWithDeniedRecipient(t *testing.T) {
	origin := net.TCPAddr{IP: []byte{127, 0, 0, 1}}
	from := "alice@example.org"
	to := []string{"bob@example.org", "charlie@example.org"}
	data := []byte{'T', 'E', 'S', 'T'}
	setName := ""
	regexp, _ := regexp.Compile(`^bob@example\.org$`)
	input, out, err, sendErr := sendHelper(&origin, from, to, data, &setName, nil, regexp, nil)
	if len(input.Destination.ToAddresses) != 1 {
		t.Errorf(
			"Unexpected number of destinations: %d. Expected: %d",
			len(input.Destination.ToAddresses),
			1,
		)
	}
	if input.Destination.ToAddresses[0] != to[1] {
		t.Errorf(
			"Unexpected destination: %s. Expected: %s",
			input.Destination.ToAddresses[0],
			to[1],
		)
	}
	if sendErr != filter.ErrDeniedRecipients {
		t.Errorf("Unexpected error: %s. Expected: %s", sendErr, filter.ErrDeniedRecipients)
	}
	if len(out) == 0 {
		t.Error("Unexpected empty stdout")
	}
	if len(err) != 0 {
		t.Errorf("Unexpected stderr: %s", err)
	}
}

func TestSendWithApiError(t *testing.T) {
	origin := net.TCPAddr{IP: []byte{127, 0, 0, 1}}
	from := "alice@example.org"
	to := []string{"bob@example.org"}
	data := []byte{'T', 'E', 'S', 'T'}
	setName := ""
	apiErr := errors.New("API failure")
	input, out, err, sendErr := sendHelper(&origin, from, to, data, &setName, nil, nil, apiErr)
	if *input.FromEmailAddress != from {
		t.Errorf(
			"Unexpected source: %s. Expected: %s",
			*input.FromEmailAddress,
			from,
		)
	}
	if len(input.Destination.ToAddresses) != 1 {
		t.Errorf(
			"Unexpected number of destinations: %d. Expected: %d",
			len(input.Destination.ToAddresses),
			1,
		)
	}
	if input.Destination.ToAddresses[0] != to[0] {
		t.Errorf(
			"Unexpected destination: %s. Expected: %s",
			input.Destination.ToAddresses[0],
			to[0],
		)
	}
	inputData := string(input.Content.Raw.Data)
	if inputData != "TEST" {
		t.Errorf("Unexpected data: %s. Expected: %s", inputData, "TEST")
	}
	if sendErr != apiErr {
		t.Errorf("Send did not report API error: %s. Expected: %s", sendErr, apiErr)
	}
	if len(out) == 0 {
		t.Error("Unexpected empty stdout")
	}
	if len(err) != 0 {
		t.Errorf("Unexpected stderr: %s", err)
	}
}

func TestNew(t *testing.T) {
	setName := ""
	allowFromRegExp, _ := regexp.Compile(`^admin@example\.org$`)
	denyToRegExp, _ := regexp.Compile(`^bob@example\.org$`)
	client := New(&setName, allowFromRegExp, denyToRegExp)
	typ := reflect.TypeOf(client).String()
	if typ != "pinpoint.Client" {
		t.Errorf("Unexpected: client is not a relay.Client:%v", typ)
	}
	if client.setName != &setName {
		t.Errorf("Unexpected setName: %s", *client.setName)
	}
	if client.allowFromRegExp != allowFromRegExp {
		t.Errorf("Unexpected allowFromRegExp: %s", client.allowFromRegExp)
	}
	if client.denyToRegExp != denyToRegExp {
		t.Errorf("Unexpected denyToRegExp: %s", client.denyToRegExp)
	}
}
