package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// Recover ловит panic и превращает в 500.
func Recover() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.Error().
					Interface("panic", r).
					Str("path", c.Request.URL.Path).
					Bytes("stack", debug.Stack()).
					Msg("panic")
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": gin.H{
						"code":    "internal",
						"message": "internal server error",
					},
				})
			}
		}()
		c.Next()
	}
}
