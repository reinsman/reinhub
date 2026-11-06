package retry

import (
	"errors"
	"testing"
	"time"

	"github.com/reinsman/reinhub/pkg/httpx"
)

type retryableError struct {
	description string
}

func (e *retryableError) Error() string {
	return e.description
}

func TestRetryDoesNotRetryNonNetworkError(t *testing.T) {
	wantErr := errors.New("business error")
	calls := 0

	err := Retry(func() error {
		calls++
		return wantErr
	}, 3, time.Duration(0))

	if !errors.Is(err, wantErr) {
		t.Fatalf("Retry() error = %v, want %v", err, wantErr)
	}
	if calls != 1 {
		t.Fatalf("Retry() calls = %d, want 1", calls)
	}
}

func TestRetrySucceedsAfterNetworkErrors(t *testing.T) {
	calls := 0

	err := Retry(func() error {
		calls++
		if calls < 3 {
			return httpx.NewNetErr(errors.New("temporary network error"))
		}
		return nil
	}, 3, time.Duration(0))

	if err != nil {
		t.Fatalf("Retry() error = %v, want nil", err)
	}
	if calls != 3 {
		t.Fatalf("Retry() calls = %d, want 3", calls)
	}
}

func TestRetryReturnsLastNetworkErrorAfterExhaustion(t *testing.T) {
	wantErr := errors.New("network unavailable")
	calls := 0

	err := Retry(func() error {
		calls++
		return httpx.NewNetErr(wantErr)
	}, 2, time.Duration(0))

	origin, ok := IsRetryErr(err)
	if !ok {
		t.Fatalf("Retry() error type = %T, want retry error", err)
	}
	if !errors.Is(origin, wantErr) {
		t.Fatalf("retry origin = %v, want %v", origin, wantErr)
	}
	if calls != 3 {
		t.Fatalf("Retry() calls = %d, want 3", calls)
	}
}

func TestRetryErrRetriesMatchingType(t *testing.T) {
	calls := 0

	err := RetryErr(func() error {
		calls++
		if calls < 3 {
			return &retryableError{description: "try again"}
		}
		return nil
	}, &retryableError{}, 2, time.Duration(0))

	if err != nil {
		t.Fatalf("RetryErr() error = %v, want nil", err)
	}
	if calls != 3 {
		t.Fatalf("RetryErr() calls = %d, want 3", calls)
	}
}

func TestRetryErrDoesNotRetryDifferentType(t *testing.T) {
	wantErr := errors.New("do not retry")
	calls := 0

	err := RetryErr(func() error {
		calls++
		return wantErr
	}, &retryableError{}, 2, time.Duration(0))

	if !errors.Is(err, wantErr) {
		t.Fatalf("RetryErr() error = %v, want %v", err, wantErr)
	}
	if calls != 1 {
		t.Fatalf("RetryErr() calls = %d, want 1", calls)
	}
}
