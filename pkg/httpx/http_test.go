package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetStatusCallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if got := r.URL.Query().Get("keyword"); got != "a b" {
			t.Fatalf("query keyword = %q, want %q", got, "a b")
		}
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte(`{"message":"teapot"}`))
	}))
	defer server.Close()

	err := Get(context.Background(), server.URL+"/test?keyword=a%20b", func(code int, header http.Header, body []byte) error {
		if code != http.StatusTeapot {
			t.Fatalf("status = %d, want %d", code, http.StatusTeapot)
		}
		if header == nil {
			t.Fatal("response header is nil")
		}
		var response map[string]string
		return json.Unmarshal(body, &response)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestRequestReturnsStatusErrorWithoutCallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "failed", http.StatusInternalServerError)
	}))
	defer server.Close()

	err := Get(context.Background(), server.URL, nil)
	if !IsStatusErr(err) {
		t.Fatalf("error = %#v, want status error", err)
	}
	code, ok := StatusCode(err)
	if !ok || code != http.StatusInternalServerError {
		t.Fatalf("StatusCode(error) = %d, %t; want 500, true", code, ok)
	}
}

func TestPostJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("Content-Type = %q", got)
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	if err := Post(context.Background(), server.URL+"/", map[string]string{"key": "value"}, nil); err != nil {
		t.Fatal(err)
	}
}

func TestNetworkErrorPreservesCause(t *testing.T) {
	cause := errors.New("transport failed")
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, cause
	})}
	previousClient := defaultClient
	t.Cleanup(func() { defaultClient = previousClient })
	if err := SetClient(client); err != nil {
		t.Fatal(err)
	}

	err := Get(context.Background(), "http://example.com", nil)
	if !IsNetErr(err) {
		t.Fatalf("error = %v, want network error", err)
	}
	if !errors.Is(err, cause) {
		t.Fatalf("errors.Is(%v, cause) = false", err)
	}
}

func TestOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Test"); got != "value" {
			t.Fatalf("X-Test = %q, want value", got)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	err := Get(context.Background(), server.URL, nil,
		WithHeader("X-Test", "value"),
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSetInsecure(t *testing.T) {
	previousClient := defaultClient
	t.Cleanup(func() { defaultClient = previousClient })
	defaultClient = http.Client{
		Timeout:   defaultTimeout,
		Transport: http.DefaultTransport.(*http.Transport).Clone(),
	}

	if err := SetInsecure(true); err != nil {
		t.Fatal(err)
	}
	transport, ok := defaultClient.Transport.(*http.Transport)
	if !ok || transport.TLSClientConfig == nil || !transport.TLSClientConfig.InsecureSkipVerify {
		t.Fatal("TLS certificate verification was not disabled")
	}

	if err := SetInsecure(false); err != nil {
		t.Fatal(err)
	}
	transport = defaultClient.Transport.(*http.Transport)
	if transport.TLSClientConfig.InsecureSkipVerify {
		t.Fatal("TLS certificate verification was not enabled")
	}
}

func TestSetInsecureRejectsCustomTransport(t *testing.T) {
	previousClient := defaultClient
	t.Cleanup(func() { defaultClient = previousClient })
	defaultClient = http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, nil
	})}

	if err := SetInsecure(true); err == nil {
		t.Fatal("expected unsupported transport error")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}
