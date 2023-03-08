package filter

import (
	"errors"
	"regexp"
)

var (
	ErrDeniedSender = errors.New(
		"denied sender: sender does not match the allowed emails regexp",
	)

	ErrDeniedRecipients = errors.New(
		"denied recipients: recipients match the denied emails regexp",
	)
)

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
) (allowedRecipients []string, deniedRecipients []string, err error) {
	allowedRecipients = []string{}
	deniedRecipients = []string{}
	if allowFromRegExp != nil && !allowFromRegExp.MatchString(from) {
		err = ErrDeniedSender
	}
	for k := range to {
		recipient := &(to)[k]
		// Deny all recipients if the sender address is not allowed
		if err != nil ||
			(denyToRegExp != nil && denyToRegExp.MatchString(*recipient)) {
			deniedRecipients = append(deniedRecipients, *recipient)
		} else {
			allowedRecipients = append(allowedRecipients, *recipient)
		}
	}
	if err == nil && len(deniedRecipients) > 0 {
		err = ErrDeniedRecipients
	}
	return
}
