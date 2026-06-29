package main

import (
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	h := &UserHandler{}
	r.POST("/api/users", h.CreateUser)

	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
