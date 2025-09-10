package api

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimiter manages rate limiting for different endpoints
type RateLimiter struct {
	visitors map[string]*rate.Limiter
	mu       sync.RWMutex

	// Different limits for different endpoint types
	authLimit    rate.Limit // Login, registration, password reset
	apiLimit     rate.Limit // General API endpoints
	uploadLimit  rate.Limit // File uploads
	messageLimit rate.Limit // Messaging endpoints

	authBurst    int
	apiBurst     int
	uploadBurst  int
	messageBurst int

	cleanupInterval time.Duration
}

// RateLimitConfig holds configuration for rate limiting
type RateLimitConfig struct {
	// Auth endpoints (login, register, password reset)
	AuthRequestsPerMinute int
	AuthBurst             int

	// General API endpoints
	APIRequestsPerMinute int
	APIBurst             int

	// File upload endpoints
	UploadRequestsPerMinute int
	UploadBurst             int

	// Message endpoints
	MessageRequestsPerMinute int
	MessageBurst             int

	// Cleanup interval for removing old visitors
	CleanupInterval time.Duration
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config RateLimitConfig) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*rate.Limiter),

		authLimit:    rate.Limit(config.AuthRequestsPerMinute) / 60,    // per second
		apiLimit:     rate.Limit(config.APIRequestsPerMinute) / 60,     // per second
		uploadLimit:  rate.Limit(config.UploadRequestsPerMinute) / 60,  // per second
		messageLimit: rate.Limit(config.MessageRequestsPerMinute) / 60, // per second

		authBurst:    config.AuthBurst,
		apiBurst:     config.APIBurst,
		uploadBurst:  config.UploadBurst,
		messageBurst: config.MessageBurst,

		cleanupInterval: config.CleanupInterval,
	}

	// Start cleanup goroutine
	go rl.cleanupVisitors()

	return rl
}

// getVisitor returns the rate limiter for a specific IP and endpoint type
func (rl *RateLimiter) getVisitor(ip string, endpointType string) *rate.Limiter {
	key := fmt.Sprintf("%s:%s", ip, endpointType)

	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.visitors[key]
	if !exists {
		var limit rate.Limit
		var burst int

		switch endpointType {
		case "auth":
			limit = rl.authLimit
			burst = rl.authBurst
		case "upload":
			limit = rl.uploadLimit
			burst = rl.uploadBurst
		case "message":
			limit = rl.messageLimit
			burst = rl.messageBurst
		default:
			limit = rl.apiLimit
			burst = rl.apiBurst
		}

		limiter = rate.NewLimiter(limit, burst)
		rl.visitors[key] = limiter
	}

	return limiter
}

// cleanupVisitors removes old visitors to prevent memory leaks
func (rl *RateLimiter) cleanupVisitors() {
	ticker := time.NewTicker(rl.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.mu.Lock()
			// Remove visitors that haven't been used recently
			for key, limiter := range rl.visitors {
				// If limiter allows all requests, it means it's been idle
				if limiter.Allow() {
					delete(rl.visitors, key)
				}
			}
			rl.mu.Unlock()
		}
	}
}

// getClientIP extracts the real client IP from the request
func getClientIP(c *gin.Context) string {
	// Check X-Forwarded-For header first (for proxies)
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}

	// Check X-Real-IP header
	if xri := c.GetHeader("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		return c.Request.RemoteAddr
	}
	return ip
}

// RateLimitMiddleware creates a rate limiting middleware for specific endpoint types
func (rl *RateLimiter) RateLimitMiddleware(endpointType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := getClientIP(c)
		limiter := rl.getVisitor(ip, endpointType)

		if !limiter.Allow() {
			// Get rate limit info for headers
			limit := rl.getLimit(endpointType)
			burst := rl.getBurst(endpointType)

			c.Header("X-RateLimit-Limit", strconv.Itoa(int(limit*60))) // requests per minute
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(time.Minute).Unix(), 10))
			c.Header("Retry-After", "60")

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded",
				"message": fmt.Sprintf("Too many requests. Limit: %d requests per minute with burst of %d",
					int(limit*60), burst),
				"retry_after": 60,
			})
			c.Abort()
			return
		}

		// Add rate limit headers for successful requests
		limit := rl.getLimit(endpointType)
		c.Header("X-RateLimit-Limit", strconv.Itoa(int(limit*60)))

		c.Next()
	}
}

// getLimit returns the rate limit for an endpoint type
func (rl *RateLimiter) getLimit(endpointType string) rate.Limit {
	switch endpointType {
	case "auth":
		return rl.authLimit
	case "upload":
		return rl.uploadLimit
	case "message":
		return rl.messageLimit
	default:
		return rl.apiLimit
	}
}

// getBurst returns the burst limit for an endpoint type
func (rl *RateLimiter) getBurst(endpointType string) int {
	switch endpointType {
	case "auth":
		return rl.authBurst
	case "upload":
		return rl.uploadBurst
	case "message":
		return rl.messageBurst
	default:
		return rl.apiBurst
	}
}

// Default rate limit configurations
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		// Strict limits for auth endpoints to prevent brute force
		AuthRequestsPerMinute: 5,
		AuthBurst:             3,

		// Moderate limits for general API usage
		APIRequestsPerMinute: 100,
		APIBurst:             20,

		// Lower limits for file uploads (resource intensive)
		UploadRequestsPerMinute: 10,
		UploadBurst:             5,

		// Higher limits for messaging (core functionality)
		MessageRequestsPerMinute: 60,
		MessageBurst:             10,

		CleanupInterval: 5 * time.Minute,
	}
}

// StrictRateLimitConfig returns a stricter configuration for production
func StrictRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		AuthRequestsPerMinute: 3,
		AuthBurst:             2,

		APIRequestsPerMinute: 60,
		APIBurst:             10,

		UploadRequestsPerMinute: 5,
		UploadBurst:             3,

		MessageRequestsPerMinute: 30,
		MessageBurst:             5,

		CleanupInterval: 3 * time.Minute,
	}
}

// PermissiveRateLimitConfig returns a more permissive configuration for development
func PermissiveRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		AuthRequestsPerMinute: 20,
		AuthBurst:             10,

		APIRequestsPerMinute: 300,
		APIBurst:             50,

		UploadRequestsPerMinute: 30,
		UploadBurst:             10,

		MessageRequestsPerMinute: 200,
		MessageBurst:             30,

		CleanupInterval: 10 * time.Minute,
	}
}
