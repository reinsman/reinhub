package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/reinsman/reinhub/pkg/api/errs"

	"github.com/gin-gonic/gin"
)

func TestAuthnRejectsRequestWithoutTrustedHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(Authn())
	engine.GET("/private", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/private", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("HTTP status = %d, want %d", recorder.Code, http.StatusOK)
	}
	var response struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Code != errs.AUTH_FAIL.Int() {
		t.Fatalf("business code = %d, want %d", response.Code, errs.AUTH_FAIL)
	}
}

func TestAuthnAndAuthzPermitTrustedRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(Authn(), Authz())
	engine.GET("/private", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "/private", nil)
	request.Header.Set(trustedHeaderName, "1")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("HTTP status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
}

func TestAuthzPlaceholderPermitsRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(Authz())
	engine.GET("/private", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/private", nil))

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("HTTP status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
}
