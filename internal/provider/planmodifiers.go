package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

// preserveStateForComputedModifier preserves the state value for computed fields
// that may change via the API but shouldn't be considered as drift.
// This prevents "Provider produced inconsistent result after apply" errors
// when the API returns slightly different values (e.g., timestamps, status changes).
type preserveStateForComputedModifier struct{}

func (m preserveStateForComputedModifier) Description(ctx context.Context) string {
	return "Preserves the state value for computed fields that may change via API"
}

func (m preserveStateForComputedModifier) MarkdownDescription(ctx context.Context) string {
	return "Preserves the state value for computed fields that may change via API"
}

func (m preserveStateForComputedModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// If state is null (resource being created), use the plan value
	if req.StateValue.IsNull() {
		return
	}

	// If the plan value is unknown, use the state value
	if req.PlanValue.IsUnknown() {
		resp.PlanValue = req.StateValue
		return
	}

	// Otherwise keep the plan value (for explicit updates)
}

// PreserveStateForComputed returns a plan modifier that preserves the state value
// for computed fields that may change via the API.
func PreserveStateForComputed() planmodifier.String {
	return preserveStateForComputedModifier{}
}
