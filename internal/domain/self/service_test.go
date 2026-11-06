package self

import (
	"context"
	"errors"
	"testing"

	domaincontract "github.com/reinsman/reinhub/internal/domain"
)

type fakeUserNumberGetter struct {
	value  int
	err    error
	ctx    context.Context
	called bool
}

func initTestDomain(t *testing.T, getter domaincontract.UserCounter) *Domain {
	t.Helper()
	domain := InitDomain(getter)
	t.Cleanup(func() {
		currentDomain = nil
	})
	return domain
}

func (f *fakeUserNumberGetter) GetUserNum(ctx context.Context) (int, error) {
	f.called = true
	f.ctx = ctx
	return f.value, f.err
}

func TestGetPing(t *testing.T) {
	userNumberGetter := &fakeUserNumberGetter{value: 3}
	initTestDomain(t, userNumberGetter)
	ctx := context.WithValue(context.Background(), struct{}{}, "test")
	health, err := getPing(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if health != 0 {
		t.Fatalf("GetPing() = %d, want 0", health)
	}
	if !userNumberGetter.called {
		t.Fatal("GetPing()没有调用User领域的GetUserNum能力")
	}
	if userNumberGetter.ctx != ctx {
		t.Fatal("GetPing()没有透传context")
	}
}

func TestGetPingPropagatesUserNumberError(t *testing.T) {
	wantErr := errors.New("user unavailable")
	initTestDomain(t, &fakeUserNumberGetter{err: wantErr})
	_, err := getPing(context.Background())
	if !errors.Is(err, wantErr) {
		t.Fatalf("GetPing() error = %v, want %v", err, wantErr)
	}
}
