/*
Package relay provides an interface to relay emails via Amazon SES/Pinpoint API.
*/
package relay

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"regexp"
	"time"
)

// Client provides an interface to send emails.
type Client interface {
	Send(
		origin net.Addr,
		from string,
		to []string,
		data []byte,
	) error
}

type logEntry struct {
	Time  time.Time
	IP    *string
	From  *string
	To    []*string
	Error *string
}

// Log creates a log entry and prints it as JSON to STDOUT.
func Log(origin net.Addr, from *string, to []*string, err error) {
	ip := origin.(*net.TCPAddr).IP.String()
	entry := &logEntry{
		Time: time.Now().UTC(),
		IP:   &ip,
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

// FilterAddresses validates sender and recipients and returns lists for allowed
// and denied recipients.
// If the sender is denied, all recipients are denied and an error is returned.
// If the sender is allowed, but some of the recipients are denied, an error
// will also be returned.
func FilterAddresses(
	from string,
	to []string,
	allowFromRegExp *regexp.Regexp,
	denyToRegExp *regexp.Regexp,
) (allowedRecipients []*string, deniedRecipients []*string, err error) {
	allowedRecipients = []*string{}
	deniedRecipients = []*string{}
	if allowFromRegExp != nil && !allowFromRegExp.MatchString(from) {
		err = errors.New(
			"Denied sender: sender does not match the allowed emails regexp",
		)
	}
	for k := range to {
		recipient := &(to)[k]
		// Deny all recipients if the sender address is not allowed
		if err != nil ||
			(denyToRegExp != nil && denyToRegExp.MatchString(*recipient)) {
			deniedRecipients = append(deniedRecipients, recipient)
		} else {
			allowedRecipients = append(allowedRecipients, recipient)
		}
	}
	if err == nil && len(deniedRecipients) > 0 {
		err = errors.New(
			"Denied recipients: recipients match the denied emails regexp",
		)
	}
	return
}
