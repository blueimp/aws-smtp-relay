/*
Package auth provides an IP and user credentials based authentication handler.
*/
package auth

import (
	"errors"
	"net"

	"golang.org/x/crypto/bcrypt"
)

// Authentication implements the AuthHandler interface.
type Authentication struct {
	ips  map[string]bool
	user string
	hash []byte
}

// Handler validates remote IPs and user credentials.
func (a Authentication) Handler(
	remoteAddr net.Addr,
	mechanism string,
	username []byte,
	password []byte,
	shared []byte,
) (bool, error) {
	if a.ips != nil {
		ip := remoteAddr.(*net.TCPAddr).IP.String()
		if a.ips[ip] != true {
			return false, errors.New("Invalid client IP: " + ip)
		}
	}
	if a.user != "" {
		if string(username) != a.user {
			return false, errors.New("Invalid username: " + string(username))
		}
		err := bcrypt.CompareHashAndPassword(a.hash, password)
		return err == nil, err
	}
	return true, nil
}

// New creates a new Authentication config.
func New(ips map[string]bool, user string, hash []byte) Authentication {
	return Authentication{
		ips:  ips,
		user: user,
		hash: hash,
	}
}
