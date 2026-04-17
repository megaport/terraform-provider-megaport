package provider

import (
	"context"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
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
