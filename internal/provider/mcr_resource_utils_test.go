package provider

import (
	"errors"
	"strings"
	"testing"
)

// TestMapMCRUpdateError covers the diagnostic wrapping that turns the platform's
// raw 400 message ("Cannot update ASN while VXCs are attached to this MCR…")
// into provider-side guidance the operator can act on. See ESD-1185.
func TestMapMCRUpdateError(t *testing.T) {
	const mcrUID = "e4c9ef77-b4e9-438a-bf1e-6d14de49a78f"

	// Mirrors the format the megaportgo SDK returns for a 400 from the
	// PUT /v2/product/mcr2/{uid} endpoint, as reported in github issue #383.
	asnAttachedAPIErr := errors.New(
		`PUT https://api-staging.megaport.com/v2/product/mcr2/e4c9ef77-b4e9-438a-bf1e-6d14de49a78f: ` +
			`400 (trace_id "e1b2d2ee44b49acfb1cb02f5bef0866c") ` +
			`Cannot update ASN while VXCs are attached to this MCR. ` +
			`Please remove all VXC connections before changing the ASN.`,
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
	})

	t.Run("unrelated error falls through to generic Update diagnostic", func(t *testing.T) {
		genericErr := errors.New("some other API failure")
		summary, detail := mapMCRUpdateError(genericErr, mcrUID)

		if summary != "Error Updating MCR" {
			t.Errorf("expected generic summary, got %q", summary)
		}
		if !strings.Contains(detail, "some other API failure") {
			t.Errorf("detail should include the underlying error, got %q", detail)
		}
	})
}
