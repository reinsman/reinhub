package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/reinsman/reinhub/pkg/api/errs"
)

var (
	testRouterOnce sync.Once
	testRouter     http.Handler
)

// routerForTest 让路由测试共享一个应用实例，符合每个领域只初始化一次的约束。
func routerForTest() http.Handler {
	testRouterOnce.Do(func() {
		testRouter = newRouter(nil, true, nil)
	})
	return testRouter
}

func TestPingRouteDoesNotRequireAuthentication(t *testing.T) {
	recorder := httptest.NewRecorder()
	routerForTest().ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodGet, "/self/ping", nil),
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("HTTP status = %d, want %d", recorder.Code, http.StatusOK)
	}
	var response struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Code != errs.SUCC.Int() {
		t.Fatalf("business code = %d, want %d", response.Code, errs.SUCC)
	}
}

func TestSwaggerRouteDoesNotRequireAuthentication(t *testing.T) {
	recorder := httptest.NewRecorder()
	routerForTest().ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodGet, "/self/swagger/index.html", nil),
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("HTTP status = %d, want %d", recorder.Code, http.StatusOK)
	}
}

func TestUserRoutesRequireAuthentication(t *testing.T) {
	recorder := httptest.NewRecorder()
	routerForTest().ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodGet, "/api/v1/user", nil),
	)

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

func TestUserAndSelfDomainsCallEachOther(t *testing.T) {
	request := httptest.NewRequest(
		http.MethodPut,
		"/api/v1/dummy/1?status=active",
		strings.NewReader(`{"id":10}`),
	)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("_HCID", "1")
	recorder := httptest.NewRecorder()

	routerForTest().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("HTTP status = %d, want %d", recorder.Code, http.StatusOK)
	}
	var response struct {
		Code int `json:"code"`
		Data struct {
			Echo int `json:"echo"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Code != errs.SUCC.Int() {
		t.Fatalf("business code = %d, want %d", response.Code, errs.SUCC)
	}
	if response.Data.Echo != 11 {
		t.Fatalf("echo = %d, want 11", response.Data.Echo)
	}
}
