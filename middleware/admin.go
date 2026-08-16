package middleware

import (
	"net/http"

	"DomainManager/database"
	"DomainManager/models"

	"github.com/gin-gonic/gin"
)

// AdminRequired ensures the authenticated user has the admin role.
// Must be used after AuthRequired so that user_id is present.
func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := c.Get("user_id")
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			c.Abort()
			return
		}

		var user models.User
		if err := database.DB.First(&user, userID.(uint)).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
			c.Abort()
			return
		}

		if user.Role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "admin permission required"})
			c.Abort()
			return
		}

		c.Next()
	}
}
