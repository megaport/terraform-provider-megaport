package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// restoreComputedOnNoOpPlan returns prior state when plan differs from it only
// in unknowns that the config leaves null, and returns plan otherwise. The
// framework marks those unknowns before any plan modifier runs. On a real
// change, Update rewrites them, and a pinned value would fail the apply.
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
	// The framework marks an attribute unknown only when its config is null.
	// An attribute wired to another resource's unknown output stays unknown.
	if !plan.IsKnown() {
		return config.IsNull(), nil
	}
	// A changed value that isn't an object is a real change. An added or
	// removed object is one too, and that covers create and destroy.
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
