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
	Field   string `json:"field"`
	Message string `json:"message"`
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
		"code":    code,
		"message": message,
	}
	if len(details) > 0 {
		errObj["details"] = details
	}
	c.JSON(status, gin.H{
		"success": false,
		"error":   errObj,
	})
}
