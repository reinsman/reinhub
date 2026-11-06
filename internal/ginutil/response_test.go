package ginutil

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/reinsman/reinhub/pkg/api/errs"

	"github.com/gin-gonic/gin"
)

func TestDoneSetsBusinessCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	Done(c, errs.FAIL, "failed")

	value, exists := c.Get("code")
	if !exists {
		t.Fatal("business code was not set")
	}
	if code, ok := value.(int); !ok || code != errs.FAIL.Int() {
		t.Fatalf("business code = %v, want %d", value, errs.FAIL.Int())
	}
}
