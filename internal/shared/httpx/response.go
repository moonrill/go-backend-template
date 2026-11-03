package httpx

import "github.com/gin-gonic/gin"

// PageMeta describes pagination metadata for list responses.
type PageMeta struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalItems int64 `json:"total_items"`
	TotalPages int   `json:"total_pages"`
}

// FieldError describes a validation error on a specific request field or query parameter.
type FieldError struct {
	Field   string `json:"field" example:"email"`
	Message string `json:"message" example:"email is required"`
}

// ErrorResponse represents a standard error response envelope.
type ErrorResponse struct {
	Success bool      `json:"success" example:"false"`
	Error   ErrorInfo `json:"error"`
}

// HealthResponse represents a standard health check response envelope.
type HealthResponse struct {
	Success bool       `json:"success" example:"true"`
	Data    HealthData `json:"data"`
}

// ErrorInfo represents structured error details in an error envelope.
type ErrorInfo struct {
	Code      string       `json:"code" example:"VALIDATION_ERROR"`
	Message   string       `json:"message" example:"Request validation failed"`
	RequestID string       `json:"request_id" example:"req_12345678-abcd"`
	Details   []FieldError `json:"details,omitempty"`
}

// HealthData represents the status response payload of a health check.
type HealthData struct {
	Status string            `json:"status" example:"live"`
	Checks map[string]string `json:"checks,omitempty"`
}

// JSONSuccess writes a standard JSON success envelope.
func JSONSuccess(c *gin.Context, status int, data any) {
	c.JSON(status, gin.H{
		"success": true,
		"data":    data,
	})
}

// JSONPaginated writes a standard JSON success envelope with pagination metadata.
func JSONPaginated(c *gin.Context, status int, data any, meta PageMeta) {
	c.JSON(status, gin.H{
		"success": true,
		"data":    data,
		"meta":    meta,
	})
}

// JSONError writes a standard JSON error envelope.
func JSONError(c *gin.Context, status int, code, message string, details ...FieldError) {
	errObj := gin.H{
		"code":       code,
		"message":    message,
		"request_id": c.GetString("request_id"),
	}
	if len(details) > 0 {
		errObj["details"] = details
	}
	c.JSON(status, gin.H{
		"success": false,
		"error":   errObj,
	})
}
