package filter

import (
	"regexp"
	"testing"
)

func TestFilterAddresses(t *testing.T) {
	from := "alice@example.org"
	to := []string{
		"bob@example.org",
		"charlie@example.org",
	}
	allowedRecipients, deniedRecipients, err := FilterAddresses(
		from,
		to,
		nil,
		nil,
	)
	if err != nil {
		t.Errorf("Unexpected error: %s. Expected: %v", err, nil)
	}
	allowedRecipientsValues := allowedRecipients
	if len(allowedRecipients) != 2 || allowedRecipientsValues[0] != to[0] ||
		allowedRecipientsValues[1] != to[1] {
		t.Errorf(
			"Unexpected allowed recipients: %s. Expected: %s",
			allowedRecipientsValues,
			to,
		)
	}
	if len(deniedRecipients) != 0 {
		t.Errorf(
			"Unexpected denied recipients: %s. Expected: %s",
			deniedRecipients,
			[]string{},
		)
	}
}

func TestFilterAddressesWithDeniedSender(t *testing.T) {
	from := "alice@example.org"
	to := []string{
		"bob@example.org",
		"charlie@example.org",
	}
	allowFromRegExp, _ := regexp.Compile(`^admin@example\.org$`)
	denyToRegExp, _ := regexp.Compile("^david@example.org$")
	expectedError := ErrDeniedSender
	allowedRecipients, deniedRecipients, err := FilterAddresses(
		from,
		to,
		allowFromRegExp,
		denyToRegExp,
	)
	if err == nil {
		t.Errorf("Unexpected error: %v. Expected: %s", nil, expectedError)
	} else if err.Error() != expectedError.Error() {
		t.Errorf(
			"Unexpected error: `%s`. Expected: `%s`",
			err.Error(),
			expectedError.Error(),
		)
	}
	if len(allowedRecipients) != 0 {
		t.Errorf(
			"Unexpected allowed recipients: %s. Expected: %s",
			allowedRecipients,
			[]string{},
		)
	}
	deniedRecipientsValues := deniedRecipients
	if len(deniedRecipients) != 2 || deniedRecipientsValues[0] != to[0] ||
		deniedRecipientsValues[1] != to[1] {
		t.Errorf(
			"Unexpected denied recipients: %s. Expected: %s",
			deniedRecipientsValues,
			to,
		)
	}
}

func TestFilterAddressesWithDeniedRecipients(t *testing.T) {
	from := "alice@example.org"
	to := []string{
		"bob@example.org",
		"charlie@example.org",
	}
	allowFromRegExp, _ := regexp.Compile(`^[^@]+@example\.org$`)
	denyToRegExp, _ := regexp.Compile(`^bob@example\.org$`)
	expectedError := ErrDeniedRecipients
	allowedRecipients, deniedRecipients, err := FilterAddresses(
		from,
		to,
		allowFromRegExp,
		denyToRegExp,
	)
	if err == nil {
		t.Errorf("Unexpected error: %v. Expected: %s", nil, expectedError)
	} else if err.Error() != expectedError.Error() {
		t.Errorf(
			"Unexpected error: `%s`. Expected: `%s`",
			err.Error(),
			expectedError.Error(),
		)
	}
	allowedRecipientsValues := allowedRecipients
	if len(allowedRecipients) != 1 || allowedRecipientsValues[0] != to[1] {
		t.Errorf(
			"Unexpected allowed recipients: %s. Expected: %s",
			allowedRecipientsValues,
			[]string{"charlie@example.org"},
		)
	}
	deniedRecipientsValues := deniedRecipients
	if len(deniedRecipients) != 1 || deniedRecipientsValues[0] != to[0] {
		t.Errorf(
			"Unexpected denied recipients: %s. Expected: %s",
			deniedRecipientsValues,
			[]string{"bob@example.org"},
		)
	}
}
