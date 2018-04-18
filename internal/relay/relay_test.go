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
		relay.Send(&mockSESAPI{}, origin, from, to, data)
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
	timeBefore := time.Now()
	input, out, err := sendHelper(&origin, &from, &to, &data)
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
