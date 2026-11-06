package user

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/reinsman/reinhub/pkg/api/errs"
	userapi "github.com/reinsman/reinhub/pkg/api/v1/user"

	"github.com/gin-gonic/gin"
)

func TestHandlerUserInfo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &fakeRepository{user: &User{
		ID:       7,
		UserName: "alice",
		Email:    "alice@example.com",
	}}
	initTestDomain(t, repo)
	router := gin.New()
	router.GET("/user", userInfo)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/user?uid=7", nil)
	type contextKey string
	const requestIDKey contextKey = "request-id"
	request = request.WithContext(context.WithValue(request.Context(), requestIDKey, "request-1"))
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	var response struct {
		Code int                  `json:"code"`
		Data userapi.UserInfoData `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Code != errs.SUCC.Int() {
		t.Fatalf("response code = %d, want %d", response.Code, errs.SUCC)
	}
	if response.Data.UID != 7 || response.Data.Name != "alice" {
		t.Fatalf("response data = %+v", response.Data)
	}
	if repo.ctx == nil || repo.ctx.Value(requestIDKey) != "request-1" {
		t.Fatal("request context was not propagated to repository")
	}
}

func TestHandlerUserInfoRejectsInvalidUID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	initTestDomain(t, &fakeRepository{})
	router := gin.New()
	router.GET("/user", userInfo)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/user?uid=invalid", nil)
	router.ServeHTTP(recorder, request)

	var response struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Code != errs.BAD_PARAS.Int() {
		t.Fatalf("response code = %d, want %d", response.Code, errs.BAD_PARAS)
	}
}
