package provider

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	megaport "github.com/megaport/megaportgo"
)

// newMegaportAPIError builds a *megaport.ErrorResponse shaped like the errors the
// megaportgo SDK returns for a non-2xx API response, so the diagnostic mappers
// can be exercised against the concrete error type they match on.
func newMegaportAPIError(path string, status int, traceID, message string) *megaport.ErrorResponse {
	return &megaport.ErrorResponse{
		Response: &http.Response{
			StatusCode: status,
			Request: &http.Request{
				Method: http.MethodPut,
				URL:    &url.URL{Scheme: "https", Host: "api-staging.megaport.com", Path: path},
			},
		},
		TraceID: traceID,
		Message: message,
	}
}

// TestMapMCRUpdateError covers the diagnostic wrapping that turns the platform's
// raw 400 message ("Cannot update ASN while VXCs are attached to this MCR…")
// into provider-side guidance the operator can act on. See ESD-1185.
func TestMapMCRUpdateError(t *testing.T) {
	const mcrUID = "e4c9ef77-b4e9-438a-bf1e-6d14de49a78f"

	// The verbatim 400 the platform returns when VXCs are still attached,
	// as reported in github issue #383.
	asnAttachedAPIErr := newMegaportAPIError(
		"/v2/product/mcr2/"+mcrUID, http.StatusBadRequest,
		"e1b2d2ee44b49acfb1cb02f5bef0866c",
		"Cannot update ASN while VXCs are attached to this MCR. "+
			"Please remove all VXC connections before changing the ASN.",
	)

	t.Run("ASN-while-attached 400 is wrapped with provider guidance", func(t *testing.T) {
		summary, detail := mapMCRUpdateError(asnAttachedAPIErr, mcrUID)

		if !strings.Contains(summary, "ASN") {
			t.Errorf("summary should mention ASN, got %q", summary)
		}
		if !strings.Contains(detail, mcrUID) {
			t.Errorf("detail should reference the MCR UID %q, got %q", mcrUID, detail)
		}
		if !strings.Contains(detail, "VXC") {
			t.Errorf("detail should mention VXC connections, got %q", detail)
		}
		// The original API message must remain in the diagnostic so the
		// trace_id and underlying cause are still visible to the operator.
		if !strings.Contains(detail, "Cannot update ASN while VXCs are attached") {
			t.Errorf("detail should retain the original API error message, got %q", detail)
		}
		if !strings.Contains(detail, "e1b2d2ee44b49acfb1cb02f5bef0866c") {
			t.Errorf("detail should retain the trace_id, got %q", detail)
		}
	})

	t.Run("wrapped API error is still unwrapped and matched", func(t *testing.T) {
		// A caller that wraps the SDK error (fmt.Errorf("...: %w", apiErr))
		// must still get the rich diagnostic; this is why the matcher uses
		// errors.As rather than a plain type assertion.
		wrapped := fmt.Errorf("modify MCR %s: %w", mcrUID, asnAttachedAPIErr)
		summary, detail := mapMCRUpdateError(wrapped, mcrUID)

		if !strings.Contains(summary, "ASN") {
			t.Errorf("summary should mention ASN for a wrapped API error, got %q", summary)
		}
		if !strings.Contains(detail, "Cannot update ASN while VXCs are attached") {
			t.Errorf("detail should retain the original API error message, got %q", detail)
		}
		if !strings.Contains(detail, "e1b2d2ee44b49acfb1cb02f5bef0866c") {
			t.Errorf("detail should retain the trace_id, got %q", detail)
		}
	})

	t.Run("non-API error (e.g. WaitForUpdate timeout) falls through to generic", func(t *testing.T) {
		// ModifyMCR runs with WaitForUpdate=true; poll timeouts are plain
		// errors, not *megaport.ErrorResponse, so they must not match.
		timeoutErr := errors.New("time expired waiting for MCR " + mcrUID + " to update")
		summary, detail := mapMCRUpdateError(timeoutErr, mcrUID)

		if summary != "Error Updating MCR" {
			t.Errorf("expected generic summary, got %q", summary)
		}
		if !strings.Contains(detail, "time expired waiting for MCR") {
			t.Errorf("detail should include the underlying error, got %q", detail)
		}
	})

	t.Run("sentinel message on a non-400 status falls through to generic", func(t *testing.T) {
		// The rich diagnostic is gated on HTTP 400; a 500 carrying the same
		// text is some other failure and must not be misreported.
		wrongStatus := newMegaportAPIError(
			"/v2/product/mcr2/"+mcrUID, http.StatusInternalServerError,
			"5f3c2b1a", "Cannot update ASN while VXCs are attached to this MCR.",
		)
		summary, _ := mapMCRUpdateError(wrongStatus, mcrUID)

		if summary != "Error Updating MCR" {
			t.Errorf("expected generic summary for a non-400 status, got %q", summary)
		}
	})
}
