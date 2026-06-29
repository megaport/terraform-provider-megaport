package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// API defaults applied when a lifetime is omitted. Kept in sync with the
// attribute descriptions and the megaportgo IPsecTunnelConfig docs.
const (
	ipSecPhase1LifetimeDefault = 28800
	ipSecPhase2LifetimeDefault = 3600
)

// ipSecPhaseLifetimeValidator enforces the API rule that an IPsec tunnel's
// phase 2 lifetime must be shorter than its phase 1 lifetime. It runs at plan
// time on the ip_sec_tunnel_options object so users get a clear error instead
// of an order rejection. A null lifetime takes the API default, so the defaults
// are folded in before comparing (this catches a high phase2 paired with an
// omitted phase1). The absolute range checks live on the individual attributes;
// this only covers the cross-field relationship.
type ipSecPhaseLifetimeValidator struct{}

func (v ipSecPhaseLifetimeValidator) Description(_ context.Context) string {
	return "phase2_lifetime must be less than phase1_lifetime"
}

func (v ipSecPhaseLifetimeValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v ipSecPhaseLifetimeValidator) ValidateObject(ctx context.Context, req validator.ObjectRequest, resp *validator.ObjectResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	var tunnel ipSecTunnelOptionsModel
	resp.Diagnostics.Append(req.ConfigValue.As(ctx, &tunnel, basetypes.ObjectAsOptions{})...)
	if resp.Diagnostics.HasError() {
		return
	}

	p1, p2 := tunnel.Phase1Lifetime, tunnel.Phase2Lifetime
	// Unknown values (e.g. interpolated from another resource) can't be checked
	// at plan time; the API enforces the relationship in that case.
	if p1.IsUnknown() || p2.IsUnknown() {
		return
	}

	effectiveP1 := int64(ipSecPhase1LifetimeDefault)
	if !p1.IsNull() {
		effectiveP1 = p1.ValueInt64()
	}
	effectiveP2 := int64(ipSecPhase2LifetimeDefault)
	if !p2.IsNull() {
		effectiveP2 = p2.ValueInt64()
	}

	if effectiveP2 >= effectiveP1 {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("phase2_lifetime"),
			"Invalid IPsec tunnel lifetimes",
			fmt.Sprintf("phase2_lifetime (%d) must be less than phase1_lifetime (%d). Unset lifetimes use the API defaults (phase1 %d, phase2 %d).", effectiveP2, effectiveP1, ipSecPhase1LifetimeDefault, ipSecPhase2LifetimeDefault),
		)
	}
}
