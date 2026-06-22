package main

import (
	"log"

	"github.com/gin-gonic/gin"
)

// main() is the only place in the app that knows about ALL dependencies.
// Every other file only knows about what it needs — nothing more.
//
// Wiring order:  Config → DB → Repo → Service → Handler → Router
//
// In Django, this wiring is implicit — settings.py + INSTALLED_APPS + AppConfig.__init__
// does it for you. Go makes it explicit. That's the trade-off: more code, full control.
func main() {
	cfg := LoadConfig()

	// Swap NewFakeUserRepo() for NewUserRepo(cfg.DatabaseURL) when you have a real DB.
	repo := NewFakeUserRepo()
	svc := NewUserService(repo)
	handler := NewUserHandler(svc)

	r := gin.Default()
	r.GET("/users/:id", handler.GetUser)

	log.Printf("listening on :%s  — try: curl http://localhost:%s/users/1", cfg.Port, cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("server: %v", err)
	}
}
