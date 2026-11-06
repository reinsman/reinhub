package user

import (
	"context"
	"errors"
	"testing"

	"github.com/reinsman/reinhub/pkg/api/errs"
)

type fakeRepository struct {
	user *User
	err  error
	ctx  context.Context
}

type fakePingGetter struct {
	value  int
	err    error
	called bool
}

func initTestDomain(t *testing.T, repo Repository) *Domain {
	t.Helper()
	domain := initDomain(repo)
	t.Cleanup(func() {
		currentDomain = nil
	})
	return domain
}

func (f *fakePingGetter) GetPing(_ context.Context) (int, error) {
	f.called = true
	return f.value, f.err
}

func (f *fakeRepository) Insert(ctx context.Context, _ string, _ string, _ string, _ string) (uint, error) {
	f.ctx = ctx
	return 0, f.err
}

func (f *fakeRepository) GetByOAuthID(ctx context.Context, _ string) (*User, error) {
	f.ctx = ctx
	return f.user, f.err
}

func (f *fakeRepository) GetByUID(ctx context.Context, _ uint) (*User, error) {
	f.ctx = ctx
	return f.user, f.err
}

func (f *fakeRepository) GetByName(ctx context.Context, _ string) (*User, error) {
	f.ctx = ctx
	return f.user, f.err
}

func TestGetByUID(t *testing.T) {
	databaseErr := errors.New("database unavailable")
	tests := []struct {
		name     string
		repo     Repository
		wantNil  bool
		wantCode errs.Code
	}{
		{
			name: "found",
			repo: &fakeRepository{user: &User{
				ID:        7,
				UserName:  "alice",
				AccountID: "account-1",
				Email:     "alice@example.com",
			}},
			wantCode: errs.SUCC,
		},
		{
			name:     "not found",
			repo:     &fakeRepository{},
			wantNil:  true,
			wantCode: errs.SUCC,
		},
		{
			name:     "repository error",
			repo:     &fakeRepository{err: databaseErr},
			wantNil:  true,
			wantCode: errs.DB_FAIL,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			initTestDomain(t, test.repo)
			data, err := getByUID(context.Background(), 7)
			if test.wantCode == errs.SUCC {
				if err != nil {
					t.Fatalf("GetByUID() error = %v", err)
				}
			} else {
				var codeErr *errs.CodeErr
				if !errors.As(err, &codeErr) {
					t.Fatalf("GetByUID() error type = %T, want *errs.CodeErr", err)
				}
				if codeErr.C != test.wantCode {
					t.Fatalf("GetByUID() error code = %d, want %d", codeErr.C, test.wantCode)
				}
			}
			if test.wantNil {
				if data != nil {
					t.Fatalf("GetByUID() data = %+v, want nil", data)
				}
				return
			}
			if data == nil || data.UID != 7 || data.Name != "alice" || data.Email != "alice@example.com" {
				t.Fatalf("GetByUID() data = %+v", data)
			}
		})
	}
}

func TestDoDummy(t *testing.T) {
	pingGetter := &fakePingGetter{}
	domain := initTestDomain(t, &fakeRepository{})
	if err := domain.WithPingGetter(pingGetter); err != nil {
		t.Fatal(err)
	}

	data, err := doDummy(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if !pingGetter.called {
		t.Fatal("DoDummy()没有调用Self领域的GetPing能力")
	}
	if data.Echo != 11 {
		t.Fatalf("DoDummy().Echo = %d, want 11", data.Echo)
	}
}

func TestGetUserNum(t *testing.T) {
	domain := initTestDomain(t, &fakeRepository{})
	num, err := domain.GetUserNum(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if num != 1 {
		t.Fatalf("GetUserNum() = %d, want 1", num)
	}
}

func TestDoDummyPropagatesPingError(t *testing.T) {
	wantErr := errors.New("self unavailable")
	domain := initTestDomain(t, &fakeRepository{})
	if err := domain.WithPingGetter(&fakePingGetter{err: wantErr}); err != nil {
		t.Fatal(err)
	}

	_, err := doDummy(context.Background(), 10)
	if !errors.Is(err, wantErr) {
		t.Fatalf("DoDummy() error = %v, want %v", err, wantErr)
	}
}
