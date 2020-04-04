/*
Package relay provides an interface to relay emails via Amazon SES/Pinpoint API.
*/
package relay

import (
	"encoding/json"
	"fmt"
	"net"
	"time"
)

// Client provides an interface to send emails.
type Client interface {
	Send(
		origin net.Addr,
		from string,
		to []string,
		data []byte,
	)
}

type logEntry struct {
	Time  time.Time
	IP    string
	From  string
	To    []string
	Error string
}

// Log creates a log entry and prints it as JSON to STDOUT.
func Log(origin net.Addr, from string, to []string, err error) {
	ip := origin.(*net.TCPAddr).IP.String()
	entry := &logEntry{
		Time: time.Now().UTC(),
		IP:   ip,
		From: from,
		To:   to,
	}
	if err != nil {
		entry.Error = err.Error()
	}
	b, _ := json.Marshal(entry)
	fmt.Println(string(b))
}
