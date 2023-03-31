package relay

import (
	"fmt"
	"io"
	"net"
)

type Client interface {
	Annotate(clt Client) Client
	Send(
		origin net.Addr,
		from string,
		to []string,
		data io.Reader,
	) error
}

func ConsumeToBytes(dr io.Reader, maxMessageSize uint) ([]byte, error) {
	if maxMessageSize == 0 {
		maxMessageSize = 1024 * 1024
	}
	data := make([]byte, maxMessageSize)
	n, err := dr.Read(data)
	if err != nil && err != io.EOF {
		return data, err
	} else {
		err = nil
	}
	if n >= len(data) {
		return data, fmt.Errorf("message size exceeds limit of %d bytes", maxMessageSize)
	}
	return data[:n], nil
}
