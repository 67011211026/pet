package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type clientLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

var (
	clientsMap = make(map[string]*clientLimiter)
	mu         sync.Mutex
)

func init() {
	// Cleanup old rate limiter entries every 10 minutes
	go func() {
		for {
			time.Sleep(10 * time.Minute)
			mu.Lock()
			for ip, client := range clientsMap {
				if time.Since(client.lastSeen) > 15*time.Minute {
					delete(clientsMap, ip)
				}
			}
			mu.Unlock()
		}
	}()
}

// RateLimitLogin limits requests to protect login/register from brute-force
func RateLimitLogin(reqPerMinute int, burst int) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()

		mu.Lock()
		client, exists := clientsMap[ip]
		if !exists {
			ratePerSec := rate.Limit(float64(reqPerMinute) / 60.0)
			limiter := rate.NewLimiter(ratePerSec, burst)
			client = &clientLimiter{limiter: limiter, lastSeen: time.Now()}
			clientsMap[ip] = client
		}
		client.lastSeen = time.Now()
		mu.Unlock()

		if !client.limiter.Allow() {
			c.Header("Retry-After", "60")
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests. Please wait a moment before trying again.",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
