package relay

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net"
	"os"
	"testing"
	"time"
)

func pointersToValues(pointers []*string) []string {
	values := []string{}
	for k := range pointers {
		if pointers[k] != nil {
			values = append(values, *(pointers)[k])
		}
	}
	return values
}

func logHelper(addr net.Addr, from *string, to []*string, err error) (
	[]byte,
	[]byte,
) {
	outReader, outWriter, _ := os.Pipe()
	errReader, errWriter, _ := os.Pipe()
	originalOut := os.Stdout
	originalErr := os.Stderr
	defer func() {
		os.Stdout = originalOut
		os.Stderr = originalErr
	}()
	os.Stdout = outWriter
	os.Stderr = errWriter
	func() {
		Log(addr, from, to, err)
		outWriter.Close()
		errWriter.Close()
	}()
	stdout, _ := ioutil.ReadAll(outReader)
	stderr, _ := ioutil.ReadAll(errReader)
	return stdout, stderr
}

func TestLog(t *testing.T) {
	origin := net.TCPAddr{IP: []byte{127, 0, 0, 1}}
	emails := []string{
		"alice@example.org",
		"bob@example.org",
		"charlie@example.org",
	}
	from := &emails[0]
	to := []*string{&emails[1], &emails[2]}
	timeBefore := time.Now()
	out, err := logHelper(&origin, from, to, nil)
	timeAfter := time.Now()
	var entry logEntry
	json.Unmarshal(out, &entry)
	if entry.Time.Before(timeBefore) {
		t.Errorf("Unexpected 'Time' log: %s", entry.Time)
	}
	if entry.Time.After(timeAfter) {
		t.Errorf("Unexpected 'Time' log: %s", entry.Time)
	}
	if entry.IP == nil {
		t.Errorf("Unexpected 'IP' log: %v. Expected: %s", nil, "127.0.0.1")
	} else if *entry.IP != "127.0.0.1" {
		t.Errorf("Unexpected 'IP' log: %s. Expected: %s", *entry.IP, "127.0.0.1")
	}
	if entry.From == nil {
		t.Errorf("Unexpected 'From' log: %v. Expected: %s", nil, *from)
	} else if *entry.From != *from {
		t.Errorf("Unexpected 'From' log: %s. Expected: %s", *entry.From, *from)
	}
	toVals := pointersToValues(entry.To)
	expectedToVals := pointersToValues(to)
	if len(toVals) != len(expectedToVals) ||
		toVals[0] != expectedToVals[0] || toVals[1] != expectedToVals[1] {
		t.Errorf("Unexpected 'To' log: %s. Expected: %s", toVals, expectedToVals)
	}
	if entry.Error != nil {
		t.Errorf("Unexpected 'Error' log: %v. Expected: %v", entry.Error, nil)
	}
	if len(err) != 0 {
		t.Errorf("Unexpected stderr: %s", err)
	}
}

func TestLogWithOriginIPv6(t *testing.T) {
	origin := net.TCPAddr{IP: []byte{
		0x20, 0x01, 0x48, 0x60, 0, 0, 0x20, 0x01, 0, 0, 0, 0, 0, 0, 0x00, 0x68,
	}}
	emails := []string{
		"alice@example.org",
		"bob@example.org",
		"charlie@example.org",
	}
	from := &emails[0]
	to := []*string{&emails[1], &emails[2]}
	out, err := logHelper(&origin, from, to, nil)
	var entry logEntry
	json.Unmarshal(out, &entry)
	if *entry.IP != "2001:4860:0:2001::68" {
		t.Errorf(
			"Unexpected 'IP' log: %s. Expected: %s",
			*entry.IP,
			"2001:4860:0:2001::68",
		)
	}
	if entry.Error != nil {
		t.Errorf("Unexpected 'Error' log: %v. Expected: %v", entry.Error, nil)
	}
	if len(err) != 0 {
		t.Errorf("Unexpected stderr: %s", err)
	}
}

func TestLogWithError(t *testing.T) {
	origin := net.TCPAddr{IP: []byte{127, 0, 0, 1}}
	emails := []string{
		"alice@example.org",
		"bob@example.org",
		"charlie@example.org",
	}
	from := &emails[0]
	to := []*string{&emails[1], &emails[2]}
	out, err := logHelper(&origin, from, to, errors.New("ERROR"))
	var entry logEntry
	json.Unmarshal(out, &entry)
	if entry.Error == nil {
		t.Errorf("Unexpected 'Error' log: %v. Expected: %s", nil, "ERROR")
	} else if *entry.Error != "ERROR" {
		t.Errorf("Unexpected 'Error' log: %s. Expected: %s", *entry.Error, "ERROR")
	}
	if len(err) != 0 {
		t.Errorf("Unexpected stderr: %s", err)
	}
}
