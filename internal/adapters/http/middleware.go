package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/henriquemontalione/ratelimiter/internal/limiter"
)

const errMessage = "you have reached the maximum number of requests or actions allowed within a certain time frame"

func RateLimit(rl *limiter.RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		token := c.GetHeader("API_KEY")

		allowed, err := rl.Allow(c.Request.Context(), ip, token)
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		if !allowed {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"message": errMessage})
			return
		}

		c.Next()
	}
}
