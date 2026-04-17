package provider

import (
	"context"
	crand "crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	megaport "github.com/megaport/megaportgo"
	"golang.org/x/time/rate"
)

// jitterRand is a process-local PRNG used to compute backoff jitter. Seeding
// explicitly from crypto/rand ensures parallel provider processes get distinct
// jitter schedules regardless of Go toolchain defaults for math/rand's global
// source, preserving the intended thundering-herd protection.
var (
	jitterRandMu sync.Mutex
	jitterRand   = rand.New(rand.NewSource(seedFromCryptoRand())) //nolint:gosec // not used for security
)

func seedFromCryptoRand() int64 {
	var b [8]byte
	if _, err := crand.Read(b[:]); err != nil {
		return time.Now().UnixNano()
	}
	return int64(binary.LittleEndian.Uint64(b[:]))
}

func jitterInt63n(n int64) int64 {
	jitterRandMu.Lock()
	defer jitterRandMu.Unlock()
	return jitterRand.Int63n(n)
}

// ErrPollTimeout is returned by PollWithBackoff when the timeout expires before
// the poll condition is met. Callers can use errors.Is(err, ErrPollTimeout) to
// distinguish timeouts from other errors.
var ErrPollTimeout = errors.New("poll timed out")

// RetryConfig holds parameters for exponential backoff retry and poll loops.
type RetryConfig struct {
	InitialBackoff time.Duration    // starting backoff duration (e.g. 2s)
	MaxBackoff     time.Duration    // upper bound on backoff (e.g. 30s)
	Multiplier     float64          // backoff growth factor (e.g. 1.5)
	MaxRetries     int              // 0 = unlimited (use Timeout instead)
	Timeout        time.Duration    // overall deadline; 0 = unlimited (use MaxRetries)
	InitialDelay   time.Duration    // delay before the first attempt (useful for poll loops)
	RetryableFunc  func(error) bool // predicate: should we retry this error? nil = retry all
}

// DefaultRetryConfig returns sensible defaults matching the existing VXC polling parameters.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		InitialBackoff: 2 * time.Second,
		MaxBackoff:     10 * time.Second,
		Multiplier:     1.5,
		Timeout:        120 * time.Second,
		InitialDelay:   1 * time.Second,
	}
}

// normalizeRetryConfig clamps invalid RetryConfig values to sensible minimums
// so that RetryWithBackoff / PollWithBackoff cannot spin on 0-duration timers
// or call jitterInt63n with a non-positive bound (which would panic).
func normalizeRetryConfig(cfg RetryConfig) RetryConfig {
	if cfg.InitialBackoff <= 0 {
		cfg.InitialBackoff = 1 * time.Second
	}
	if cfg.MaxBackoff <= 0 {
		cfg.MaxBackoff = 30 * time.Second
	}
	if cfg.MaxBackoff < cfg.InitialBackoff {
		cfg.MaxBackoff = cfg.InitialBackoff
	}
	if cfg.Multiplier <= 0 {
		cfg.Multiplier = 1.5
	}
	return cfg
}

// RetryWithBackoff calls fn repeatedly with exponential backoff and full jitter
// until it succeeds, the context is cancelled, MaxRetries is exhausted, or
// Timeout expires.
//
// Full jitter: sleep = rand(0, min(maxBackoff, initialBackoff * multiplier^attempt))
// using a process-local PRNG seeded from crypto/rand (see jitterRand) so that
// parallel provider processes get distinct backoff schedules.
//
// If RetryableFunc is set, only errors matching the predicate are retried;
// all others are returned immediately.
//
// If InitialDelay > 0, it is applied before the first attempt.
func RetryWithBackoff(ctx context.Context, cfg RetryConfig, fn func(ctx context.Context) error) error {
	cfg = normalizeRetryConfig(cfg)
	var deadline time.Time
	if cfg.Timeout > 0 {
		deadline = time.Now().Add(cfg.Timeout)
	}

	if cfg.InitialDelay > 0 {
		timer := time.NewTimer(cfg.InitialDelay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}

	var lastErr error
	for attempt := 0; ; attempt++ {
		if !deadline.IsZero() && time.Now().After(deadline) {
			if lastErr != nil {
				return fmt.Errorf("timeout (%v) exceeded: %w", cfg.Timeout, lastErr)
			}
			return fmt.Errorf("timeout (%v) exceeded", cfg.Timeout)
		}

		lastErr = fn(ctx)
		if lastErr == nil {
			return nil
		}

		// Check if the error is retryable.
		if cfg.RetryableFunc != nil && !cfg.RetryableFunc(lastErr) {
			return lastErr
		}

		// If no retries remain, return immediately — don't sleep before
		// giving up, which would add up to MaxBackoff of pointless delay.
		if cfg.MaxRetries > 0 && attempt+1 >= cfg.MaxRetries {
			return fmt.Errorf("max retries (%d) exceeded: %w", cfg.MaxRetries, lastErr)
		}

		// Full jitter: sleep = rand(0, min(maxBackoff, initial * multiplier^attempt))
		calculated := float64(cfg.InitialBackoff) * math.Pow(cfg.Multiplier, float64(attempt))
		capped := math.Min(calculated, float64(cfg.MaxBackoff))
		jittered := time.Duration(jitterInt63n(int64(capped) + 1))

		// Cap the sleep to the remaining time before the deadline so we
		// don't overshoot Timeout by up to a full backoff interval.
		if !deadline.IsZero() {
			remaining := time.Until(deadline)
			if remaining <= 0 {
				return fmt.Errorf("timeout (%v) exceeded: %w", cfg.Timeout, lastErr)
			}
			if jittered > remaining {
				jittered = remaining
			}
		}

		timer := time.NewTimer(jittered)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

// PollWithBackoff calls fn repeatedly with deterministic exponential backoff
// until fn signals completion, the context is cancelled, or the timeout expires.
//
// fn returns (value, done, error):
//   - done=true: polling stops, value is returned
//   - error!=nil: polling stops immediately with the error
//   - done=false, error==nil: keep polling
//
// On timeout, the returned error wraps ErrPollTimeout so callers can use
// errors.Is(err, ErrPollTimeout) to distinguish timeouts from API errors.
func PollWithBackoff[T any](ctx context.Context, cfg RetryConfig, fn func(ctx context.Context) (T, bool, error)) (T, error) {
	cfg = normalizeRetryConfig(cfg)
	var zero T
	var deadline time.Time
	if cfg.Timeout > 0 {
		deadline = time.Now().Add(cfg.Timeout)
	}

	// Optional initial delay before the first poll.
	if cfg.InitialDelay > 0 {
		timer := time.NewTimer(cfg.InitialDelay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return zero, ctx.Err()
		case <-timer.C:
		}
	}

	backoff := cfg.InitialBackoff
	for {
		if !deadline.IsZero() && time.Now().After(deadline) {
			return zero, fmt.Errorf("%w after %v", ErrPollTimeout, cfg.Timeout)
		}

		val, done, err := fn(ctx)
		if err != nil {
			return zero, err
		}
		if done {
			return val, nil
		}

		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return zero, ctx.Err()
		case <-timer.C:
			backoff = time.Duration(float64(backoff) * cfg.Multiplier)
			if backoff > cfg.MaxBackoff {
				backoff = cfg.MaxBackoff
			}
		}
	}
}

// IsRetryableHTTPError returns true for HTTP status codes that warrant a retry:
// 409 Conflict, 429 Too Many Requests, and 500+.
func IsRetryableHTTPError(err error) bool {
	var apiErr *megaport.ErrorResponse
	if !errors.As(err, &apiErr) || apiErr.Response == nil {
		return false
	}
	code := apiErr.Response.StatusCode
	return code == http.StatusConflict ||
		code == http.StatusTooManyRequests ||
		code >= http.StatusInternalServerError
}

// IsNotFoundError returns true if the error represents "resource not found".
// The Megaport API normally signals this with HTTP 404, but some endpoints
// (notably MCR prefix filter lists) return 400 Bad Request with a
// "Could not find" message instead — both are treated as not-found here so
// callers can uniformly remove the resource from state.
func IsNotFoundError(err error) bool {
	var apiErr *megaport.ErrorResponse
	if !errors.As(err, &apiErr) || apiErr.Response == nil {
		return false
	}
	code := apiErr.Response.StatusCode
	if code == http.StatusNotFound {
		return true
	}
	if code == http.StatusBadRequest && strings.Contains(apiErr.Message, "Could not find") {
		return true
	}
	return false
}

// NewAPIRateLimiter creates a context-aware rate limiter using golang.org/x/time/rate.
// Unlike the previous channel-based implementation, this does not spawn background
// goroutines and will not leak resources.
//
// Usage: call limiter.Wait(ctx) before each API request.
func NewAPIRateLimiter(burst int, refillInterval time.Duration) *rate.Limiter {
	return rate.NewLimiter(rate.Every(refillInterval), burst)
}

// retryTransientDelete retries a delete operation when the API returns transient
// backend errors like "Transaction silently rolled back because it has been marked
// as rollback-only". These are server-side transaction conflicts that typically
// succeed on retry.
//
//nolint:unparam // maxAttempts is constant today but callers may vary it in future
func retryTransientDelete(ctx context.Context, maxAttempts int, fn func() error) error {
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	var err error
	for attempt := range maxAttempts {
		err = fn()
		if err == nil {
			return nil
		}
		if !isTransientDeleteError(err) {
			return err
		}
		if attempt < maxAttempts-1 {
			tflog.Debug(ctx, "Transient delete error, retrying",
				map[string]interface{}{
					"attempt": attempt + 1,
					"error":   err.Error(),
				})
			select {
			case <-time.After(time.Duration(attempt+1) * 2 * time.Second):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	return err
}

func isTransientDeleteError(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "rollback-only") ||
		strings.Contains(msg, "Transaction silently rolled back")
}
