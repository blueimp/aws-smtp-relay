package relay_test

import (
	"encoding/json"
	"io/ioutil"
	"net"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/aws/aws-sdk-go/service/ses/sesiface"
	"github.com/blueimp/aws-smtp-relay/internal/relay"
)

var testData = struct{ input *ses.SendRawEmailInput }{}

type mockSESAPI struct {
	sesiface.SESAPI
}

func (m *mockSESAPI) SendRawEmail(input *ses.SendRawEmailInput) (
	*ses.SendRawEmailOutput,
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
) (email *ses.SendRawEmailInput, out []byte, err []byte) {
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
		relay.Send(&mockSESAPI{}, origin, from, to, data, setName)
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
	if *input.Source != from {
		t.Errorf(
			"Unexpected source: %s. Expected: %s",
			*input.Source,
			from,
		)
	}
	if *input.Destinations[0] != to[0] {
		t.Errorf(
			"Unexpected destination: %s. Expected: %s",
			*input.Destinations[0],
			to[0],
		)
	}
	inputData := string(input.RawMessage.Data)
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
	input, out, err := sendHelper(&origin, &from, &to, &data, &setName)
	if len(input.Destinations) != 2 {
		t.Errorf(
			"Unexpected number of destinations: %d. Expected: %d",
			len(input.Destinations),
			2,
		)
	}
	if *input.Destinations[0] != to[0] {
		t.Errorf(
			"Unexpected destination: %s. Expected: %s",
			*input.Destinations[0],
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
