package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Envelope is the consistent success response shape for the API.
//
//	{ "success": true, "message": "", "data": {} }
type Envelope struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
	Meta    *Meta       `json:"meta,omitempty"`
}

// ErrorEnvelope is the consistent error response shape for the API.
//
//	{ "success": false, "message": "", "errors": [] }
type ErrorEnvelope struct {
	Success bool     `json:"success"`
	Message string   `json:"message"`
	Errors  []string `json:"errors"`
}

// Meta carries pagination metadata for list responses.
type Meta struct {
	Total int64 `json:"total"`
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
}

func OK(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, Envelope{Success: true, Message: message, Data: data})
}

func Created(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusCreated, Envelope{Success: true, Message: message, Data: data})
}

func Paginated(c *gin.Context, message string, data interface{}, meta Meta) {
	c.JSON(http.StatusOK, Envelope{Success: true, Message: message, Data: data, Meta: &meta})
}

func Error(c *gin.Context, status int, message string, errs ...string) {
	if errs == nil {
		errs = []string{}
	}
	c.JSON(status, ErrorEnvelope{Success: false, Message: message, Errors: errs})
}
