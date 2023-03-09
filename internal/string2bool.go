package internal

import "strings"

func String2bool(str string) bool {
	matcher := strings.ToLower(strings.TrimSpace(str))
	switch matcher {
	case "1", "t", "true", "yes", "y", "on":
		return true
	case "0", "f", "false", "no", "n", "off":
		return false
	}
	return false
}
