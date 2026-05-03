package share

import (
	"crypto/rand"
	"fmt"

	"github.com/gin-gonic/gin"
)

const RequestIDKey = "requestID"

// RequestID is a Gin middleware that generates or forwards a unique request ID.
// If the incoming request carries an X-Request-ID header its value is reused;
// otherwise a new random ID is generated.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = newRequestID()
		}
		c.Set(RequestIDKey, requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// GetRequestID retrieves the request ID stored by the RequestID middleware.
// Returns an empty string when the middleware was not applied.
func GetRequestID(c *gin.Context) string {
	if v, exists := c.Get(RequestIDKey); exists {
		if id, ok := v.(string); ok {
			return id
		}
	}
	return ""
}

// newRequestID generates a random UUID v4-like string using crypto/rand.
func newRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "00000000-0000-0000-0000-000000000000"
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant bits
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
