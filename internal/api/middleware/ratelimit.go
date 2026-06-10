package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type attemptEntry struct {
	count   int
	blocked time.Time
}

type rateLimiter struct {
	mu   sync.Mutex
	data map[string]*attemptEntry
}

var loginLimiter = &rateLimiter{
	data: make(map[string]*attemptEntry),
}

func LoginRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		loginLimiter.mu.Lock()
		ip := c.ClientIP()
		entry, ok := loginLimiter.data[ip]
		if !ok {
			entry = &attemptEntry{}
			loginLimiter.data[ip] = entry
		}
		if time.Now().Before(entry.blocked) {
			loginLimiter.mu.Unlock()
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many login attempts, try again later"})
			return
		}
		loginLimiter.mu.Unlock()

		c.Next()

		if c.Writer.Status() == http.StatusUnauthorized {
			loginLimiter.mu.Lock()
			entry.count++
			if entry.count >= 5 {
				entry.blocked = time.Now().Add(time.Duration(entry.count-4) * time.Second)
			}
			loginLimiter.mu.Unlock()
		} else {
			loginLimiter.mu.Lock()
			entry.count = 0
			entry.blocked = time.Time{}
			loginLimiter.mu.Unlock()
		}
	}
}
