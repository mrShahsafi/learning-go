package main

import (
	"regexp"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

var zipRegex = regexp.MustCompile(`^\d{5}$`)

// Address is an optional nested struct.
// It implements validation.Validatable so ozzo validates it automatically
// when it appears as a field in a parent struct.
type Address struct {
	Street string `json:"street"`
	City   string `json:"city"`
	Zip    string `json:"zip"`
}

func (a Address) Validate() error {
	return validation.ValidateStruct(&a,
		validation.Field(&a.Street, validation.Required),
		validation.Field(&a.City, validation.Required),
		validation.Field(&a.Zip, validation.Required, validation.Match(zipRegex).Error("must be a 5-digit US zip")),
	)
}

// CreateUserRequest is the body for POST /api/users.
type CreateUserRequest struct {
	Email    string   `json:"email"`
	Password string   `json:"password"`
	Name     string   `json:"name"`
	Address  *Address `json:"address"` // pointer = optional; nil skips nested validation
}

func (r CreateUserRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Email,
			validation.Required,
			is.Email,
		),
		validation.Field(&r.Password,
			validation.Required,
			StrongPassword{},
		),
		validation.Field(&r.Name,
			validation.Required,
			validation.Length(2, 100),
		),
		// Address is optional but, if provided, must pass its own Validate().
		// ozzo calls addr.Validate() automatically because *Address implements
		// validation.Validatable.
		validation.Field(&r.Address),
	)
}
