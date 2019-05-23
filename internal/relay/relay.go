/*
Package relay provides an interface to relay emails via Amazon SES/Pinpoint API.
*/
package relay

import "net"

// Client provides an interface to send emails.
type Client interface {
	Send(
		origin net.Addr,
		from string,
		to []string,
		data []byte,
	)
}
