package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

// This wires the real app for a quick manual smoke test. The interesting
// part of this lesson is `go test ./...` — see user_service_test.go and
// user_handler_test.go.
func main() {
	repo := NewFakeUserRepo() // swap for NewUserRepo(dsn) against a real Postgres
	svc := NewUserService(repo)
	h := NewUserHandler(svc)

	r := gin.Default()
	r.GET("/users/:id", h.GetUser)
	r.POST("/users", h.CreateUser)

	fmt.Println("Run: curl -X POST localhost:8080/users -d '{\"email\":\"a@b.com\",\"name\":\"A\"}'")
	fmt.Println("Then: curl localhost:8080/users/1")
	_ = r.Run(":8080")
}
