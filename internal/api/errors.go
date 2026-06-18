package api

import (
	"github.com/gin-gonic/gin"
)

// ErrorBody is the standard error envelope returned by every error response.
//
//	{"error": {"code": "not_found", "message": "job not found"}}
//
// `code` is a stable machine-readable string clients can branch on; `message`
// is human-readable and may change. `request_id` is set when the request_id
// middleware ran so callers can quote it in support tickets.
type ErrorBody struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

// RespondError writes a standard error envelope at the given status code.
// Keep code values stable across releases; clients depend on them.
func RespondError(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, gin.H{
		"error": ErrorBody{
			Code:      code,
			Message:   message,
			RequestID: RequestIDFromContext(c),
		},
	})
}
