package internal

import (
	"encoding/json"
	"fmt"
	"net"
	"time"
)

func Log(component string, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	entry := errLog{
		Time:      time.Now().UTC(),
		Component: component,
		Msg:       &msg,
	}
	b, _ := json.Marshal(entry)
	fmt.Println(string(b))
}

type logEntry struct {
	Time  time.Time
	IP    string
	From  string
	To    []string
	Error *string
}

// LogEmail creates a log entry and prints it as JSON to STDOUT.
func LogEmail(origin net.Addr, from string, to []string, err error) {
	ip := origin.(*net.TCPAddr).IP.String()
	entry := &logEntry{
		Time: time.Now().UTC(),
		IP:   ip,
		From: from,
		To:   to,
	}
	if err != nil {
		errString := err.Error()
		entry.Error = &errString
	}
	b, _ := json.Marshal(entry)
	fmt.Println(string(b))
}

type errLog struct {
	Time      time.Time
	Component string
	Msg       *string `json:"Msg,omitempty"`
	Error     *string `json:"Error,omitempty"`
}

func LogError(component string, format string, args ...any) error {
	err := fmt.Errorf(format, args...)
	errStr := err.Error()
	entry := errLog{
		Time:      time.Now().UTC(),
		Component: component,
		Error:     &errStr,
	}
	b, _ := json.Marshal(entry)
	fmt.Println(string(b))
	return err
}
