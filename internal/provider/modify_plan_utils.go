package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// restoreComputedOnNoOpPlan converges a plan whose only content is unknowns.
// Removing an Optional+Computed nested attribute from a configuration makes the
// framework mark every Computed attribute with a null config value as unknown,
// and it does that before any plan modifier runs. Restoring prior state here is
// the only place left. A real change anywhere returns the plan untouched:
// Update rewrites those attributes, so pinning them would fail the apply with
// an inconsistent result.
func restoreComputedOnNoOpPlan(plan, state, config tftypes.Value) (tftypes.Value, error) {
	// Nothing to restore when creating (no prior state) or destroying (no plan).
	if state.IsNull() || plan.IsNull() {
		return plan, nil
	}

	var planAttrs, stateAttrs, configAttrs map[string]tftypes.Value
	if err := plan.As(&planAttrs); err != nil {
		return plan, fmt.Errorf("could not read the planned values: %w", err)
	}
	if err := state.As(&stateAttrs); err != nil {
		return plan, fmt.Errorf("could not read the prior state values: %w", err)
	}
	if err := config.As(&configAttrs); err != nil {
		return plan, fmt.Errorf("could not read the configured values: %w", err)
	}

	restorable := map[string]tftypes.Value{}
	for name, planValue := range planAttrs {
		if planValue.Equal(stateAttrs[name]) {
			continue
		}
		// Same test the framework used to mark it: unknown in the plan, null
		// in the config. An attribute wired to another resource's unknown
		// output fails this and stays unknown.
		if !planValue.IsKnown() && configAttrs[name].IsNull() {
			restorable[name] = stateAttrs[name]
			continue
		}
		// A real change. Leave the whole plan as it is.
		return plan, nil
	}

	if len(restorable) == 0 {
		return plan, nil
	}

	for name, stateValue := range restorable {
		planAttrs[name] = stateValue
	}
	return tftypes.NewValue(plan.Type(), planAttrs), nil
}
