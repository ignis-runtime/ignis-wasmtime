package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func UUIDValidator() gin.HandlerFunc {
	return func(c *gin.Context) {
		runtimeID := c.Param("uuid")

		// Validate UUID format
		parsedUUID, err := uuid.Parse(runtimeID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
				"error": "invalid runtime identifier format",
			})
			return
		}
		c.Set("uuid", parsedUUID)
		c.Next()
	}
}
