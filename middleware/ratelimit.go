package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// simple in-memory per-IP rate limiter (auth brute-force protection).
type ipRateLimiter struct {
	mu       sync.Mutex
	window   time.Duration
	limit    int
	requests map[string][]time.Time
}

func newIPRateLimiter(limit int, window time.Duration) *ipRateLimiter {
	return &ipRateLimiter{
		window:   window,
		limit:    limit,
		requests: make(map[string][]time.Time),
	}
}

func (l *ipRateLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-l.window)

	times := l.requests[key]
	kept := times[:0]
	for _, t := range times {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	l.requests[key] = kept

	if len(kept) >= l.limit {
		return false
	}
	l.requests[key] = append(l.requests[key], now)
	return true
}

var authLimiter = newIPRateLimiter(10, time.Minute)

// RateLimitAuth limits login/register attempts per client IP.
func RateLimitAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !authLimiter.allow(ip) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many requests, please try again later"})
			c.Abort()
			return
		}
		c.Next()
	}
}
