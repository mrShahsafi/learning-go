// Package domain holds plain data types shared across every layer.
// It imports nothing from this module — that's what makes it safe for
// repository, service, and handler to all depend on it without a cycle.
package domain

type Booking struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}
