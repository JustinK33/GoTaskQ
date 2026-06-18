package api

import (
	"crypto/rand"
	"encoding/hex"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// HeaderRequestID is the canonical request-id header used both for accepting
// caller-supplied ids and for echoing the chosen id back in the response.
const HeaderRequestID = "X-Request-Id"

// requestIDKey is the gin-context key used to retrieve the request id from
// downstream handlers. Use RequestIDFromContext for typed access.
const requestIDKey = "request_id"

// RequestID returns a middleware that ensures every request carries an
// X-Request-Id, generating one if the caller didn't provide it. The id is
// echoed in the response header, stored in gin context, and added to a
// per-request zerolog sub-logger that downstream handlers can pull out
// with zerolog.Ctx(c.Request.Context()).
func RequestID(base zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(HeaderRequestID)
		if id == "" {
			id = newRequestID()
		}
		c.Set(requestIDKey, id)
		c.Writer.Header().Set(HeaderRequestID, id)

		log := base.With().Str("request_id", id).Logger()
		ctx := log.WithContext(c.Request.Context())
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// RequestLogger logs one structured line per request after it completes.
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		log := zerolog.Ctx(c.Request.Context())
		evt := log.Info()
		if c.Writer.Status() >= 500 {
			evt = log.Error()
		} else if c.Writer.Status() >= 400 {
			evt = log.Warn()
		}
		evt.
			Str("method", c.Request.Method).
			Str("path", c.FullPath()).
			Int("status", c.Writer.Status()).
			Dur("latency", time.Since(start)).
			Str("client_ip", c.ClientIP()).
			Msg("http_request")
	}
}

// RequestIDFromContext returns the request id stashed by RequestID middleware,
// or empty if the middleware hasn't run.
func RequestIDFromContext(c *gin.Context) string {
	if v, ok := c.Get(requestIDKey); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func newRequestID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "req-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return hex.EncodeToString(b)
}
