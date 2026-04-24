package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	megaport "github.com/megaport/megaportgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- RetryWithBackoff tests ---

func TestRetryWithBackoff_SuccessOnFirstAttempt(t *testing.T) {
	calls := 0
	err := RetryWithBackoff(context.Background(), RetryConfig{
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
		Multiplier:     2.0,
		MaxAttempts:    5,
	}, func(_ context.Context) error {
		calls++
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, 1, calls)
}

func TestRetryWithBackoff_SuccessAfterRetries(t *testing.T) {
	calls := 0
	err := RetryWithBackoff(context.Background(), RetryConfig{
		InitialBackoff: 1 * time.Millisecond,
		MaxBackoff:     10 * time.Millisecond,
		Multiplier:     2.0,
		MaxAttempts:    5,
	}, func(_ context.Context) error {
		calls++
		if calls < 3 {
			return errors.New("transient")
		}
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, 3, calls)
}

func TestRetryWithBackoff_MaxAttemptsExhausted(t *testing.T) {
	calls := 0
	err := RetryWithBackoff(context.Background(), RetryConfig{
		InitialBackoff: 1 * time.Millisecond,
		MaxBackoff:     10 * time.Millisecond,
		Multiplier:     2.0,
		MaxAttempts:    3,
	}, func(_ context.Context) error {
		calls++
		return errors.New("always fails")
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "max attempts (3) exceeded")
	assert.Equal(t, 3, calls)
}

func TestRetryWithBackoff_TimeoutExceeded(t *testing.T) {
	calls := 0
	err := RetryWithBackoff(context.Background(), RetryConfig{
		InitialBackoff: 50 * time.Millisecond,
		MaxBackoff:     50 * time.Millisecond,
		Multiplier:     1.0,
		Timeout:        120 * time.Millisecond,
	}, func(_ context.Context) error {
		calls++
		return errors.New("always fails")
	})

	require.Error(t, err)
	// Should have been called at least once before timeout
	assert.GreaterOrEqual(t, calls, 1)
}

func TestRetryWithBackoff_NonRetryableError(t *testing.T) {
	calls := 0
	permanent := errors.New("permanent")
	err := RetryWithBackoff(context.Background(), RetryConfig{
		InitialBackoff: 1 * time.Millisecond,
		MaxBackoff:     10 * time.Millisecond,
		Multiplier:     2.0,
		MaxAttempts:    10,
		RetryableFunc:  func(err error) bool { return err.Error() == "transient" },
	}, func(_ context.Context) error {
		calls++
		return permanent
	})

	require.ErrorIs(t, err, permanent)
	assert.Equal(t, 1, calls, "non-retryable error should stop immediately")
}

func TestRetryWithBackoff_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	err := RetryWithBackoff(ctx, RetryConfig{
		InitialBackoff: 500 * time.Millisecond,
		MaxBackoff:     500 * time.Millisecond,
		Multiplier:     1.0,
		MaxAttempts:    100,
	}, func(_ context.Context) error {
		calls++
		if calls == 1 {
			cancel()
		}
		return errors.New("fail")
	})

	require.ErrorIs(t, err, context.Canceled)
}

// --- PollWithBackoff tests ---

func TestPollWithBackoff_ConditionMet(t *testing.T) {
	calls := 0
	val, err := PollWithBackoff(context.Background(), RetryConfig{
		InitialBackoff: 1 * time.Millisecond,
		MaxBackoff:     10 * time.Millisecond,
		Multiplier:     2.0,
		Timeout:        5 * time.Second,
	}, func(_ context.Context) (string, bool, error) {
		calls++
		if calls >= 3 {
			return "done", true, nil
		}
		return "", false, nil
	})

	require.NoError(t, err)
	assert.Equal(t, "done", val)
	assert.Equal(t, 3, calls)
}

func TestPollWithBackoff_Timeout(t *testing.T) {
	calls := 0
	_, err := PollWithBackoff(context.Background(), RetryConfig{
		InitialBackoff: 30 * time.Millisecond,
		MaxBackoff:     30 * time.Millisecond,
		Multiplier:     1.0,
		Timeout:        100 * time.Millisecond,
	}, func(_ context.Context) (int, bool, error) {
		calls++
		return 0, false, nil
	})

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrPollTimeout), "should wrap ErrPollTimeout")
	assert.GreaterOrEqual(t, calls, 1)
}

func TestPollWithBackoff_InitialDelay(t *testing.T) {
	start := time.Now()
	calls := 0
	_, err := PollWithBackoff(context.Background(), RetryConfig{
		InitialBackoff: 1 * time.Millisecond,
		MaxBackoff:     10 * time.Millisecond,
		Multiplier:     2.0,
		Timeout:        5 * time.Second,
		InitialDelay:   50 * time.Millisecond,
	}, func(_ context.Context) (struct{}, bool, error) {
		calls++
		return struct{}{}, true, nil
	})

	require.NoError(t, err)
	assert.Equal(t, 1, calls)
	assert.GreaterOrEqual(t, time.Since(start), 40*time.Millisecond, "should have waited for initial delay")
}

func TestPollWithBackoff_ErrorStopsPolling(t *testing.T) {
	calls := 0
	apiErr := errors.New("api failure")
	_, err := PollWithBackoff(context.Background(), RetryConfig{
		InitialBackoff: 1 * time.Millisecond,
		MaxBackoff:     10 * time.Millisecond,
		Multiplier:     2.0,
		Timeout:        5 * time.Second,
	}, func(_ context.Context) (string, bool, error) {
		calls++
		if calls == 2 {
			return "", false, apiErr
		}
		return "", false, nil
	})

	require.ErrorIs(t, err, apiErr)
	assert.Equal(t, 2, calls, "should stop on first error")
}

func TestPollWithBackoff_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	_, err := PollWithBackoff(ctx, RetryConfig{
		InitialBackoff: 500 * time.Millisecond,
		MaxBackoff:     500 * time.Millisecond,
		Multiplier:     1.0,
		Timeout:        10 * time.Second,
	}, func(_ context.Context) (int, bool, error) {
		calls++
		if calls == 1 {
			cancel()
		}
		return 0, false, nil
	})

	require.ErrorIs(t, err, context.Canceled)
}

// --- IsRetryableHTTPError tests ---

func TestIsRetryableHTTPError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "409 Conflict is retryable",
			err:      &megaport.ErrorResponse{Response: &http.Response{StatusCode: http.StatusConflict}},
			expected: true,
		},
		{
			name:     "429 Too Many Requests is retryable",
			err:      &megaport.ErrorResponse{Response: &http.Response{StatusCode: http.StatusTooManyRequests}},
			expected: true,
		},
		{
			name:     "500 Internal Server Error is retryable",
			err:      &megaport.ErrorResponse{Response: &http.Response{StatusCode: http.StatusInternalServerError}},
			expected: true,
		},
		{
			name:     "502 Bad Gateway is retryable",
			err:      &megaport.ErrorResponse{Response: &http.Response{StatusCode: http.StatusBadGateway}},
			expected: true,
		},
		{
			name:     "503 Service Unavailable is retryable",
			err:      &megaport.ErrorResponse{Response: &http.Response{StatusCode: http.StatusServiceUnavailable}},
			expected: true,
		},
		{
			name:     "404 Not Found is NOT retryable",
			err:      &megaport.ErrorResponse{Response: &http.Response{StatusCode: http.StatusNotFound}},
			expected: false,
		},
		{
			name:     "400 Bad Request is NOT retryable",
			err:      &megaport.ErrorResponse{Response: &http.Response{StatusCode: http.StatusBadRequest}},
			expected: false,
		},
		{
			name:     "401 Unauthorized is NOT retryable",
			err:      &megaport.ErrorResponse{Response: &http.Response{StatusCode: http.StatusUnauthorized}},
			expected: false,
		},
		{
			name:     "non-API error is NOT retryable",
			err:      fmt.Errorf("network timeout"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsRetryableHTTPError(tt.err))
		})
	}
}

// --- IsNotFoundError tests ---

func TestIsNotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "404 is not found",
			err:      &megaport.ErrorResponse{Response: &http.Response{StatusCode: http.StatusNotFound}},
			expected: true,
		},
		{
			name:     "409 is not 'not found'",
			err:      &megaport.ErrorResponse{Response: &http.Response{StatusCode: http.StatusConflict}},
			expected: false,
		},
		{
			name: "400 with 'Could not find' is not found",
			err: &megaport.ErrorResponse{
				Response: &http.Response{StatusCode: http.StatusBadRequest},
				Message:  "Could not find prefix filter list with id 12345",
			},
			expected: true,
		},
		{
			name: "400 with other message is not 'not found'",
			err: &megaport.ErrorResponse{
				Response: &http.Response{StatusCode: http.StatusBadRequest},
				Message:  "Invalid VLAN value",
			},
			expected: false,
		},
		{
			name:     "non-API error",
			err:      errors.New("something else"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsNotFoundError(tt.err))
		})
	}
}

// --- NewAPIRateLimiter tests ---

func TestNewAPIRateLimiter(t *testing.T) {
	limiter := NewAPIRateLimiter(5, 100*time.Millisecond)
	ctx := context.Background()

	// Should be able to burst 5 tokens immediately.
	for i := 0; i < 5; i++ {
		err := limiter.Wait(ctx)
		require.NoError(t, err, "burst token %d should be available", i)
	}

	// Context cancellation should stop Wait.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := limiter.Wait(ctx)
	require.ErrorIs(t, err, context.Canceled)
}
