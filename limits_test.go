package main

import (
	"bufio"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestReadLineRejectsOversizedInput(t *testing.T) {
	long := bufio.NewReaderSize(strings.NewReader(strings.Repeat("a", maxLineBytes*3)+"\n"), maxLineBytes)
	if _, err := readLine(long); !errors.Is(err, bufio.ErrBufferFull) {
		t.Fatalf("expected ErrBufferFull, got %v", err)
	}
	short := bufio.NewReaderSize(strings.NewReader("answer\n"), maxLineBytes)
	if got, err := readLine(short); err != nil || got != "answer\n" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestTeamGateLimitsSessions(t *testing.T) {
	gate := newTeamGate(2, 0)
	first, _, err := gate.Acquire("a")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = gate.Acquire("a"); err != nil {
		t.Fatal(err)
	}
	if _, _, err = gate.Acquire("a"); !errors.Is(err, errTooManySessions) {
		t.Fatalf("expected limit, got %v", err)
	}
	if _, _, err = gate.Acquire("b"); err != nil {
		t.Fatalf("other team must not be affected: %v", err)
	}
	first()
	first()
	if _, _, err = gate.Acquire("a"); err != nil {
		t.Fatalf("slot should be free: %v", err)
	}
	if _, _, err = gate.Acquire("a"); !errors.Is(err, errTooManySessions) {
		t.Fatalf("double release must free one slot only, got %v", err)
	}
}

func TestTeamGateCooldown(t *testing.T) {
	gate := newTeamGate(0, 50*time.Millisecond)
	gate.Penalize("a")
	if _, left, err := gate.Acquire("a"); !errors.Is(err, errCoolingDown) || left <= 0 {
		t.Fatalf("expected cooldown, got %v %v", left, err)
	}
	time.Sleep(60 * time.Millisecond)
	if _, _, err := gate.Acquire("a"); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyLimiterCachesResults(t *testing.T) {
	var calls atomic.Int32
	rejected := errors.New("rejected")
	limiter := newVerifyLimiter(func(token string) (Identity, error) {
		calls.Add(1)
		if token == "bad" {
			return Identity{}, rejected
		}
		return Identity{Name: token, LogKey: token}, nil
	}, 1000)
	for range 5 {
		if id, err := limiter.Verify("good"); err != nil || id.Name != "good" {
			t.Fatal(id, err)
		}
		if _, err := limiter.Verify("bad"); !errors.Is(err, rejected) {
			t.Fatal(err)
		}
	}
	if calls.Load() != 2 {
		t.Fatalf("expected 2 upstream calls, got %d", calls.Load())
	}
}

func TestVerifyLimiterDoesNotCacheOutages(t *testing.T) {
	var calls atomic.Int32
	limiter := newVerifyLimiter(func(string) (Identity, error) {
		calls.Add(1)
		return Identity{}, errAuthUnavailable
	}, 1000)
	limiter.Verify("x")
	limiter.Verify("x")
	if calls.Load() != 2 {
		t.Fatalf("outages must be retried, got %d calls", calls.Load())
	}
}

func TestVerifyLimiterShedsLoad(t *testing.T) {
	limiter := newVerifyLimiter(func(string) (Identity, error) { return Identity{}, nil }, 1)
	limiter.nextSlot = time.Now().Add(authMaxQueueWait + time.Second)
	if _, err := limiter.Verify("new"); !errors.Is(err, errAuthUnavailable) {
		t.Fatalf("expected shed, got %v", err)
	}
}
