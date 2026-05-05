package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// unknownWhenNoPriorStatePlanModifier marks a Computed attribute as unknown when
// there is no prior state value. This is needed for Computed-only attributes inside
// ListNestedAttribute elements: the TF Plugin Framework does not automatically set
// them to unknown when a new list element is added during an update, so without this
// modifier the provider cannot write a real value (inconsistency error).
type unknownWhenNoPriorStatePlanModifier struct{}

func (m unknownWhenNoPriorStatePlanModifier) Description(_ context.Context) string {
	return "Sets the planned value to unknown when there is no prior state value"
}

func (m unknownWhenNoPriorStatePlanModifier) MarkdownDescription(_ context.Context) string {
	return "Sets the planned value to unknown when there is no prior state value"
}

func (m unknownWhenNoPriorStatePlanModifier) PlanModifyInt64(_ context.Context, req planmodifier.Int64Request, resp *planmodifier.Int64Response) {
	if req.StateValue.IsNull() {
		resp.PlanValue = types.Int64Unknown()
	}
}

func unknownWhenNoPriorState() planmodifier.Int64 {
	return unknownWhenNoPriorStatePlanModifier{}
}

type emptyPrefixFilterListPlanModifier struct{}

// Description returns a plain text description of the validator's behavior.
func (m emptyPrefixFilterListPlanModifier) Description(_ context.Context) string {
	return "If the list is null or unknown, it will be set to an empty list"
}

// MarkdownDescription returns a markdown description of the validator's behavior.
func (m emptyPrefixFilterListPlanModifier) MarkdownDescription(_ context.Context) string {
	return "If the list is null or unknown, it will be set to an empty list"
}

// PlanModifyList sets null lists to empty lists during planning.
func (m emptyPrefixFilterListPlanModifier) PlanModifyList(ctx context.Context, req planmodifier.ListRequest, resp *planmodifier.ListResponse) {
	// If list is null or unknown in the plan, set it to an empty list
	if req.PlanValue.IsNull() || req.PlanValue.IsUnknown() {
		resp.PlanValue = types.ListValueMust(
			types.ObjectType{}.WithAttributeTypes(mcrPrefixFilterListModelAttributes),
			[]attr.Value{},
		)
	}
}

// EmptyPrefixFilterListIfNull returns a plan modifier that sets null lists to empty lists.
func EmptyPrefixFilterListIfNull() planmodifier.List {
	return emptyPrefixFilterListPlanModifier{}
}
