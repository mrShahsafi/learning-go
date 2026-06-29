package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

// ProblemDetail is an RFC 9457 problem detail envelope.
// Use this as the error body for every non-2xx response.
type ProblemDetail struct {
	Type       string         `json:"type"`
	Title      string         `json:"title"`
	Status     int            `json:"status"`
	Detail     string         `json:"detail"`
	Instance   string         `json:"instance,omitempty"`
	Extensions map[string]any `json:"extensions,omitempty"`
}

// RespondValidationError converts an ozzo validation.Errors into an RFC 9457
// 422 response and writes it to c.
func RespondValidationError(c *gin.Context, err error) {
	errs, ok := err.(validation.Errors)
	if !ok {
		// Not a field-level error — return a generic 422.
		c.JSON(http.StatusUnprocessableEntity, ProblemDetail{
			Type:   "about:blank",
			Title:  "Unprocessable Entity",
			Status: http.StatusUnprocessableEntity,
			Detail: err.Error(),
		})
		return
	}

	c.JSON(http.StatusUnprocessableEntity, ProblemDetail{
		Type:     "https://coordeck.com/problems/validation-error",
		Title:    "Validation Error",
		Status:   http.StatusUnprocessableEntity,
		Detail:   "One or more fields failed validation.",
		Instance: c.Request.URL.Path,
		Extensions: map[string]any{
			"errors": flattenErrors(errs),
		},
	})
}

// flattenErrors recursively converts ozzo's nested validation.Errors into a
// plain map so nested structs like "address.city" appear as nested JSON objects.
//
//	{"email": "cannot be blank", "address": {"zip": "must be a 5-digit US zip"}}
func flattenErrors(errs validation.Errors) map[string]any {
	out := make(map[string]any, len(errs))
	for field, err := range errs {
		if nested, ok := err.(validation.Errors); ok {
			out[field] = flattenErrors(nested)
		} else {
			out[field] = err.Error()
		}
	}
	return out
}
