package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type UserHandler struct{}

// CreateUser handles POST /api/users.
//
// Step 1: ShouldBindJSON — deserialize (shape + JSON syntax only, no domain rules)
// Step 2: req.Validate()  — validate domain rules (required, format, strength, nesting)
//
// This two-step split is the Go equivalent of DRF's is_valid().
// The trap: forgetting step 2 means unvalidated data silently reaches your service.
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ProblemDetail{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: "Request body is not valid JSON.",
		})
		return
	}

	if err := req.Validate(); err != nil {
		RespondValidationError(c, err)
		return
	}

	// Beyond this point req is guaranteed valid — pass to service/repo.
	c.JSON(http.StatusCreated, gin.H{
		"message": "user would be created here",
		"email":   req.Email,
		"name":    req.Name,
	})
}
