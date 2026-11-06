package middleware

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

func recoveryHandler(c *gin.Context, recovered any) {
	if err, ok := recovered.(error); ok && errors.Is(err, http.ErrAbortHandler) {
		return
	}
	c.AbortWithStatus(http.StatusInternalServerError)
}
