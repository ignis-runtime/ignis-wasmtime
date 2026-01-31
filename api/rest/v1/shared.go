package v1

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// APIError represents an error response from the API
// @Description API Error Response
type APIError struct {
	Code int    `json:"code"`
	Err  string `json:"err"`
	Data any    `json:"data,omitempty"`
}

func (e APIError) Error() string {
	return e.Err
}

// APIResponse represents a success response from the API
// @Description API Success Response
type APIResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data,omitempty"`
}

func (r APIResponse) Error() string {
	return r.Msg
}

func ErrorHandler(fn func(c *gin.Context) error) gin.HandlerFunc {
	return func(c *gin.Context) {
		err := fn(c)
		var apiErr APIError
		var apiResp APIResponse
		if err != nil {
			if errors.As(err, &apiErr) {
				c.AbortWithStatusJSON(apiErr.Code, apiErr)
				return
			} else if errors.As(err, &apiResp) {
				c.AbortWithStatusJSON(apiResp.Code, apiResp)
			} else {
				c.AbortWithStatusJSON(http.StatusInternalServerError, APIError{
					Code: http.StatusInternalServerError,
					Err:  err.Error(),
				})
			}
		}
	}
}
