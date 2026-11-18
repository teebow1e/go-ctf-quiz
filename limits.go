package main

import (
	"bufio"
	"crypto/sha256"
	"errors"
	"os"
	"strconv"
	"sync"
	"time"
)

const (
	// maxLineBytes bounds every line read from a player, so a client that never
	// sends a newline cannot grow server memory.
	maxLineBytes    = 4096
	maxLoggedAnswer = 256

	authCacheEntries = 10000
	authAcceptedTTL  = 5 * time.Minute
	authRejectedTTL  = 30 * time.Second
	authMaxQueueWait = 8 * time.Second
	authConcurrency  = 8
)

func envInt(name string, fallback int) int {
	if parsed, err := strconv.Atoi(os.Getenv(name)); err == nil && parsed >= 0 {
		return parsed
	}
	return fallback
}

func readLine(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadSlice('\n')
	if err != nil {
		return "", err
	}
	return string(line), nil
}

func truncateForLog(answer string) string {
	if len(answer) <= maxLoggedAnswer {
		return answer
	}
	return answer[:maxLoggedAnswer] + "...[truncated]"
}

// TeamGate limits how many sessions a team holds at once and how soon it can
// retry after a failed attempt.
type TeamGate struct {
	mu           sync.Mutex
	active       map[string]int
	blockedUntil map[string]time.Time
	maxSessions  int
	cooldown     time.Duration
}

func newTeamGate(maxSessions int, cooldown time.Duration) *TeamGate {
	return &TeamGate{
		active:       make(map[string]int),
		blockedUntil: make(map[string]time.Time),
		maxSessions:  maxSessions,
		cooldown:     cooldown,
	}
}

var (
	errTooManySessions = errors.New("too many sessions for this team")
	errCoolingDown     = errors.New("team is cooling down")
)

// Acquire reserves a session. On errCoolingDown the duration is the time left.
func (gate *TeamGate) Acquire(team string) (func(), time.Duration, error) {
	gate.mu.Lock()
	defer gate.mu.Unlock()

	if left := time.Until(gate.blockedUntil[team]); left > 0 {
		return nil, left, errCoolingDown
	}
	if gate.maxSessions > 0 && gate.active[team] >= gate.maxSessions {
		return nil, 0, errTooManySessions
	}
	gate.active[team]++
	released := false
	return func() {
		gate.mu.Lock()
		defer gate.mu.Unlock()
		if released {
			return
		}
		released = true
		gate.active[team]--
		if gate.active[team] <= 0 {
			delete(gate.active, team)
		}
	}, 0, nil
}

// Penalize starts the retry cooldown after a failed attempt.
func (gate *TeamGate) Penalize(team string) {
	if gate.cooldown <= 0 {
		return
	}
	gate.mu.Lock()
	defer gate.mu.Unlock()
	gate.blockedUntil[team] = time.Now().Add(gate.cooldown)
}

type cachedIdentity struct {
	identity Identity
	err      error
	expires  time.Time
}

// VerifyLimiter caches token checks and paces the ones that reach the CTF
// platform, so reconnects and junk tokens cannot flood it. Keys are hashes:
// team tokens are credentials and are not kept in memory.
type VerifyLimiter struct {
	verify   func(string) (Identity, error)
	interval time.Duration
	slots    chan struct{}

	mu       sync.Mutex
	cache    map[[sha256.Size]byte]cachedIdentity
	nextSlot time.Time
}

func newVerifyLimiter(verify func(string) (Identity, error), perSecond int) *VerifyLimiter {
	if perSecond <= 0 {
		perSecond = 1
	}
	return &VerifyLimiter{
		verify:   verify,
		interval: time.Second / time.Duration(perSecond),
		slots:    make(chan struct{}, authConcurrency),
		cache:    make(map[[sha256.Size]byte]cachedIdentity),
	}
}

func (limiter *VerifyLimiter) reserve(key [sha256.Size]byte) (cachedIdentity, bool, time.Duration) {
	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	now := time.Now()
	if hit, ok := limiter.cache[key]; ok && now.Before(hit.expires) {
		return hit, true, 0
	}
	if limiter.nextSlot.Before(now) {
		limiter.nextSlot = now
	}
	wait := limiter.nextSlot.Sub(now)
	if wait <= authMaxQueueWait {
		limiter.nextSlot = limiter.nextSlot.Add(limiter.interval)
	}
	return cachedIdentity{}, false, wait
}

func (limiter *VerifyLimiter) store(key [sha256.Size]byte, entry cachedIdentity) {
	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	if len(limiter.cache) >= authCacheEntries {
		limiter.cache = make(map[[sha256.Size]byte]cachedIdentity)
	}
	limiter.cache[key] = entry
}

// Verify returns errAuthUnavailable when the queue is too long.
func (limiter *VerifyLimiter) Verify(token string) (Identity, error) {
	key := sha256.Sum256([]byte(token))
	hit, cached, wait := limiter.reserve(key)
	if cached {
		return hit.identity, hit.err
	}
	if wait > authMaxQueueWait {
		return Identity{}, errAuthUnavailable
	}
	time.Sleep(wait)

	limiter.slots <- struct{}{}
	defer func() { <-limiter.slots }()

	identity, err := limiter.verify(token)
	switch {
	case err == nil:
		limiter.store(key, cachedIdentity{identity: identity, expires: time.Now().Add(authAcceptedTTL)})
	case !errors.Is(err, errAuthUnavailable):
		limiter.store(key, cachedIdentity{err: err, expires: time.Now().Add(authRejectedTTL)})
	}
	return identity, err
}
