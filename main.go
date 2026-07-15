package main

import (
	"DomainManager/config"
	"DomainManager/database"
	"DomainManager/router"
	"embed"
	"io/fs"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed web/dist/*
var frontendFS embed.FS

func main() {
	config.Load()
	database.Init()

	r := router.Setup()

	sub, err := fs.Sub(frontendFS, "web/dist")
	if err != nil {
		log.Fatalf("failed to load embedded frontend: %v", err)
	}

	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		path = strings.TrimPrefix(path, "/")
		if path == "" {
			path = "index.html"
		}
		if f, err := sub.Open(path); err == nil {
			f.Close()
			c.FileFromFS(path, http.FS(sub))
			return
		}
		c.FileFromFS("index.html", http.FS(sub))
	})

	log.Printf("Server starting on port %s", config.AppConfig.Port)
	if err := r.Run(":" + config.AppConfig.Port); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
