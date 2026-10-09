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
	// Nothing to restore when creating (no prior state) or destroying (no plan).
	if state.IsNull() || plan.IsNull() {
		return plan, nil
	}

	restored, ok, err := restoreUnknowns(plan, state, config)
	if err != nil || !ok {
		return plan, err
	}
	return restored, nil
}

// restoreUnknowns returns plan with prior state in place of each restorable
// unknown. It returns false when plan holds a real change.
func restoreUnknowns(plan, state, config tftypes.Value) (tftypes.Value, bool, error) {
	if plan.Equal(state) {
		return plan, true, nil
	}
	// Same test the framework used to mark it: unknown in the plan, null in
	// the config. An attribute wired to another resource's unknown output
	// fails this and stays unknown.
	if !plan.IsKnown() {
		return state, config.IsNull(), nil
	}

	objType, isObject := plan.Type().(tftypes.Object)
	if !isObject || plan.IsNull() || state.IsNull() {
		return plan, false, nil
	}

	var planAttrs, stateAttrs, configAttrs map[string]tftypes.Value
	if err := plan.As(&planAttrs); err != nil {
		return plan, false, fmt.Errorf("could not read the planned values: %w", err)
	}
	if err := state.As(&stateAttrs); err != nil {
		return plan, false, fmt.Errorf("could not read the prior state values: %w", err)
	}
	if err := config.As(&configAttrs); err != nil {
		return plan, false, fmt.Errorf("could not read the configured values: %w", err)
	}

	// As shares the value's own map, so write into a new one.
	restoredAttrs := make(map[string]tftypes.Value, len(planAttrs))
	for name, planValue := range planAttrs {
		// A null config object reads as an empty map, so each child is null.
		configValue, found := configAttrs[name]
		if !found {
			configValue = tftypes.NewValue(objType.AttributeTypes[name], nil)
		}
		restored, ok, err := restoreUnknowns(planValue, stateAttrs[name], configValue)
		if err != nil || !ok {
			return plan, false, err
		}
		restoredAttrs[name] = restored
	}
	return tftypes.NewValue(objType, restoredAttrs), true, nil
}
