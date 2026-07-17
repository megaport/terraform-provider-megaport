package provider

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	megaport "github.com/megaport/megaportgo"
)

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

// retryTransientGet retries a read operation when the API returns a transient
// backend error (a 5xx or a transport-level failure). Without it a single blip
// on the post-buy read of a freshly provisioned, billable resource would be
// treated as a hard failure and trigger cleanup of a live resource.
//
//nolint:unparam // maxAttempts is constant today but callers may vary it in future
func retryTransientGet(ctx context.Context, maxAttempts int, fn func() error) error {
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	var err error
	for attempt := range maxAttempts {
		err = fn()
		if err == nil {
			return nil
		}
		if !isTransientGetError(err) {
			return err
		}
		if attempt < maxAttempts-1 {
			tflog.Debug(ctx, "Transient read error, retrying",
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

func isTransientGetError(err error) bool {
	if err == nil {
		return false
	}
	// A cancelled or timed-out context won't recover on retry.
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var apiErr *megaport.ErrorResponse
	if errors.As(err, &apiErr) {
		// 5xx are server-side blips that usually clear on retry; 4xx are
		// genuine failures that must fall through to the caller.
		return apiErr.Response != nil && apiErr.Response.StatusCode >= http.StatusInternalServerError
	}
	// Non-API errors here are transport-level failures (reset, timeout).
	return true
}
