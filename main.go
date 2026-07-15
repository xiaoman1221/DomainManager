package main

import (
	"DomainManager/config"
	"DomainManager/database"
	"DomainManager/router"
	"embed"
	"io"
	"io/fs"
	"log"
	"mime"
	"net/http"
	"path/filepath"
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

	serveFile := func(c *gin.Context, name string) {
		f, err := sub.Open(name)
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		defer f.Close()
		data, err := io.ReadAll(f)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		ext := filepath.Ext(name)
		c.Data(http.StatusOK, mime.TypeByExtension(ext), data)
	}

	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		clean := strings.TrimPrefix(path, "/")
		if clean == "" {
			clean = "index.html"
		}
		if _, err := fs.Stat(sub, clean); err == nil {
			serveFile(c, clean)
			return
		}
		serveFile(c, "index.html")
	})

	log.Printf("Server starting on port %s", config.AppConfig.Port)
	if err := r.Run(":" + config.AppConfig.Port); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
