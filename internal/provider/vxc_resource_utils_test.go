package provider

import (
	"errors"
	"strings"
	"testing"
)

// TestMapVXCUpdateError covers the diagnostic wrapping that turns NetAuto's
// raw 400 message ("VRouter: localAsn may not change the neighbour
// relationship from IBGP to EBGP…") into provider-side guidance the operator
// can act on. See ESD-1185.
func TestMapVXCUpdateError(t *testing.T) {
	const vxcUID = "36cfd12e-ba4a-465e-8c05-c0803ee8bc22"

	// Mirrors the format the megaportgo SDK returns for a 400 from
	// PUT /v3/product/vxc/{uid}, as reported in github issue #383.
	ibgpEbgpAPIErr := errors.New(
		`PUT https://api-staging.megaport.com/v3/product/vxc/36cfd12e-ba4a-465e-8c05-c0803ee8bc22: ` +
			`400 (trace_id "7a7670f5b04e2af7d0837fc4d7f27309") ` +
			`VRouter: localAsn may not change the neighbour relationship from IBGP to EBGP, ` +
			`Validation of csp_request failed`,
	)

	t.Run("iBGP-to-eBGP transition 400 is wrapped with provider guidance", func(t *testing.T) {
		summary, detail := mapVXCUpdateError(ibgpEbgpAPIErr, vxcUID)

		if !strings.Contains(strings.ToLower(summary), "bgp") {
			t.Errorf("summary should mention BGP, got %q", summary)
		}
		if !strings.Contains(detail, vxcUID) {
			t.Errorf("detail should reference the VXC UID %q, got %q", vxcUID, detail)
		}
		if !strings.Contains(detail, "iBGP") && !strings.Contains(detail, "IBGP") {
			t.Errorf("detail should explain the iBGP/eBGP constraint, got %q", detail)
		}
		// The original API message must remain so trace_id and underlying
		// cause stay visible.
		if !strings.Contains(detail, "localAsn may not change the neighbour relationship") {
			t.Errorf("detail should retain the original API error message, got %q", detail)
		}
	})

	t.Run("unrelated error falls through to generic Update diagnostic", func(t *testing.T) {
		genericErr := errors.New("some other API failure")
		summary, detail := mapVXCUpdateError(genericErr, vxcUID)

		if summary != "Error Updating VXC" {
			t.Errorf("expected generic summary, got %q", summary)
		}
		if !strings.Contains(detail, vxcUID) {
			t.Errorf("detail should include the VXC UID, got %q", detail)
		}
		if !strings.Contains(detail, "some other API failure") {
			t.Errorf("detail should include the underlying error, got %q", detail)
		}
	})
}
