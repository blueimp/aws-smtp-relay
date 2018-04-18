package main

import (
	"os"
	"testing"
)

func TestOptions(t *testing.T) {
	os.Args = []string{"noop"}
	addr, _, name, host := options()
	if addr != ":1025" {
		t.Errorf("Unexpected addr: %s. Expected: %s", addr, ":1025")
	}
	if name != "AWS SMTP Relay" {
		t.Errorf("Unexpected addr: %s. Expected: %s", name, "AWS SMTP Relay")
	}
	if host != "" {
		t.Errorf("Unexpected host: %s. Expected host to be empty.", host)
	}
	os.Args = append(
		[]string{"noop"},
		"-a",
		"127.0.0.1:25",
		"-n",
		"BANANA",
		"-h",
		"localhost",
	)
	addr, _, name, host = options()
	if addr != os.Args[2] {
		t.Errorf("Unexpected addr: %s. Expected: %s", addr, os.Args[2])
	}
	if name != os.Args[4] {
		t.Errorf("Unexpected addr: %s. Expected: %s", name, os.Args[4])
	}
	if host != os.Args[6] {
		t.Errorf("Unexpected host: %s. Expected: %s", host, os.Args[6])
	}
}
