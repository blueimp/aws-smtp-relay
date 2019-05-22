package relay

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"time"
)

// Request object
type Request struct {
	Time  time.Time
	Addr  net.Addr
	IP    string
	From  string
	To    []string
	Error string
	Err   *error
}

// Log prints the Request data to Stdout/Stderr
func (req *Request) Log() {
	req.IP = req.Addr.(*net.TCPAddr).IP.String()
	var out *os.File
	err := *req.Err
	if err != nil {
		req.Error = err.Error()
		out = os.Stderr
	} else {
		out = os.Stdout
	}
	b, _ := json.Marshal(req)
	fmt.Fprintln(out, string(b))
}