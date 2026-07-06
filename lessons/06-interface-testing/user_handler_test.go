package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
)

// This is the Go equivalent of Django REST Framework's APITestCase / APIClient.
// httptest.NewRecorder() plays the role of the test client, and we wire a
// real gin.Engine with the fakeUserRepo behind it — no DB, no mocking
// framework, full HTTP round trip through routing + binding + JSON encoding.

func newTestRouter() (*gin.Engine, *UserHandler) {
	gin.SetMode(gin.TestMode)
	repo := NewFakeUserRepo()
	svc := NewUserService(repo)
	h := NewUserHandler(svc)

	r := gin.New()
	r.GET("/users/:id", h.GetUser)
	r.POST("/users", h.CreateUser)
	return r, h
}

func TestUserHandler_CreateUser(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "valid payload",
			body:       `{"email":"amir@coordeck.com","name":"Amir"}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "missing email",
			body:       `{"name":"Amir"}`,
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "malformed json",
			body:       `{"email":`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router, _ := newTestRouter()

			req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}

func TestUserHandler_GetUser_NotFound(t *testing.T) {
	router, _ := newTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/users/42", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}

	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["error"] == "" {
		t.Errorf("expected error message in body, got %v", body)
	}
}

func TestUserHandler_CreateThenGet(t *testing.T) {
	router, _ := newTestRouter()

	createReq := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBufferString(
		`{"email":"alice@example.com","name":"Alice"}`,
	))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	router.ServeHTTP(createW, createReq)

	if createW.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d", createW.Code, http.StatusCreated)
	}

	var created User
	if err := json.Unmarshal(createW.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to decode created user: %v", err)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/users/"+strconv.FormatInt(created.ID, 10), nil)
	getW := httptest.NewRecorder()
	router.ServeHTTP(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Fatalf("get status = %d, want %d (body: %s)", getW.Code, http.StatusOK, getW.Body.String())
	}
}
