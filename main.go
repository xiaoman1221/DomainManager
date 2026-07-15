package main

import (
	"DomainManager/config"
	"DomainManager/database"
	"DomainManager/router"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

func main() {
	config.Load()
	database.Init()

	r := router.Setup()

	frontendDir := "./web/dist"
	if _, err := os.Stat(frontendDir); err == nil {
		r.NoRoute(func(c *gin.Context) {
			path := c.Request.URL.Path
			if strings.HasPrefix(path, "/api") {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			filePath := filepath.Join(frontendDir, path)
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				c.File(filepath.Join(frontendDir, "index.html"))
			} else {
				c.File(filePath)
			}
		})
		log.Printf("Serving frontend from %s", frontendDir)
	}

	log.Printf("Server starting on port %s", config.AppConfig.Port)
	if err := r.Run(":" + config.AppConfig.Port); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
