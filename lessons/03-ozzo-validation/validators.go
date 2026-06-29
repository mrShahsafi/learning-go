package main

import (
	"errors"
	"unicode"

	validation "github.com/go-ozzo/ozzo-validation/v4"
)

// StrongPassword is a custom validation.Rule.
// Implement the single Validate(value interface{}) error method.
type StrongPassword struct{}

func (StrongPassword) Validate(value interface{}) error {
	s, _ := value.(string)

	var hasUpper, hasDigit bool
	for _, r := range s {
		if unicode.IsUpper(r) {
			hasUpper = true
		}
		if unicode.IsDigit(r) {
			hasDigit = true
		}
	}

	if len(s) < 8 || !hasUpper || !hasDigit {
		return errors.New("must be at least 8 characters with one uppercase letter and one digit")
	}
	return nil
}

// Compile-time check: StrongPassword must satisfy validation.Rule.
// If the interface changes (it won't), this line breaks the build immediately.
var _ validation.Rule = StrongPassword{}
