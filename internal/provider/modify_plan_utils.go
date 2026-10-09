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
// an inconsistent result. It walks nested objects too, since the framework
// marks their Computed children the same way.
func restoreComputedOnNoOpPlan(plan, state, config tftypes.Value) (tftypes.Value, error) {
	noOp, err := onlyRestorableUnknowns(plan, state, config)
	if err != nil || !noOp {
		return plan, err
	}
	return state, nil
}

// onlyRestorableUnknowns reports whether plan differs from state only in
// unknowns that the config leaves null.
func onlyRestorableUnknowns(plan, state, config tftypes.Value) (bool, error) {
	if plan.Equal(state) {
		return true, nil
	}
	// Same test the framework used to mark it: unknown in the plan, null in
	// the config. An attribute wired to another resource's unknown output
	// fails this and stays unknown.
	if !plan.IsKnown() {
		return config.IsNull(), nil
	}
	// Create, destroy, and an added or removed object are real changes.
	if _, isObject := plan.Type().(tftypes.Object); !isObject || plan.IsNull() || state.IsNull() {
		return false, nil
	}

	var planAttrs, stateAttrs, configAttrs map[string]tftypes.Value
	if err := plan.As(&planAttrs); err != nil {
		return false, fmt.Errorf("could not read the planned values: %w", err)
	}
	if err := state.As(&stateAttrs); err != nil {
		return false, fmt.Errorf("could not read the prior state values: %w", err)
	}
	if err := config.As(&configAttrs); err != nil {
		return false, fmt.Errorf("could not read the configured values: %w", err)
	}

	// A null config object reads as an empty map, and a missing child as null.
	for name, planValue := range planAttrs {
		noOp, err := onlyRestorableUnknowns(planValue, stateAttrs[name], configAttrs[name])
		if err != nil || !noOp {
			return false, err
		}
	}
	return true, nil
}
