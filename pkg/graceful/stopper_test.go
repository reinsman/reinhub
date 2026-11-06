package graceful

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"
)

func TestStopperStopCancelsRunningTask(t *testing.T) {
	stopper := NewStopper(100 * time.Millisecond)
	started := make(chan struct{})
	result := make(chan error, 1)

	go func() {
		result <- stopper.Run(func(ctx context.Context) error {
			close(started)
			<-ctx.Done()
			return nil
		})
	}()

	<-started
	stopper.Stop()

	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("Run() error = %v, want nil", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Run() did not return after Stop()")
	}
	if !stopper.IsStopped() {
		t.Fatal("IsStopped() = false after Stop()")
	}
}

func TestRunManyReturnsTaskErrorAndRunsFinalizersInReverseOrder(t *testing.T) {
	stopper := NewStopper(100 * time.Millisecond)
	wantErr := errors.New("task failed")
	started := make(chan struct{})

	var mu sync.Mutex
	var finalized []int
	stopper.AddFinalizer(
		func() error {
			mu.Lock()
			defer mu.Unlock()
			finalized = append(finalized, 1)
			return nil
		},
		func() error {
			mu.Lock()
			defer mu.Unlock()
			finalized = append(finalized, 2)
			return nil
		},
	)

	err := stopper.RunMany(
		func(ctx context.Context) error {
			close(started)
			<-ctx.Done()
			return nil
		},
		func(context.Context) error {
			<-started
			return wantErr
		},
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("RunMany() error = %v, want %v", err, wantErr)
	}

	mu.Lock()
	defer mu.Unlock()
	if !slices.Equal(finalized, []int{2, 1}) {
		t.Fatalf("finalizer order = %v, want [2 1]", finalized)
	}
}

func TestRunManyTimesOutWhenTaskIgnoresCancellation(t *testing.T) {
	stopper := NewStopper(10 * time.Millisecond)
	started := make(chan struct{})
	release := make(chan struct{})
	result := make(chan error, 1)

	go func() {
		result <- stopper.Run(func(context.Context) error {
			close(started)
			<-release
			return nil
		})
	}()

	<-started
	stopper.Stop()
	err := <-result
	close(release)

	if !IsTimeoutError(err) {
		t.Fatalf("Run() error = %v, want context deadline exceeded", err)
	}
}
