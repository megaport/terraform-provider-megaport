package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// mcrAsnAttachedVXCSentinel is the substring the Megaport platform returns
// (verbatim, from MegaPortService.validateMcrAsn in Megalith) when an MCR
// ASN update is rejected because live VXCs are still attached to the MCR.
const mcrAsnAttachedVXCSentinel = "Cannot update ASN while VXCs are attached"

// mapMCRUpdateError translates a megaportgo MCR update error into a Terraform
// diagnostic (summary, detail). Known platform constraints get richer
// provider-side guidance; unrecognised errors fall through to the historical
// generic Update diagnostic so we don't hide novel failure modes from users.
func mapMCRUpdateError(err error, mcrUID string) (summary, detail string) {
	msg := err.Error()
	if strings.Contains(msg, mcrAsnAttachedVXCSentinel) {
		return "Cannot update MCR ASN while VXCs are attached",
			fmt.Sprintf(
				"The Megaport API rejected the ASN update on MCR %s because it has live VXC connections. "+
					"The platform does not currently support changing an MCR's ASN while VXCs are attached; "+
					"all attached VXCs must be deleted before the ASN can be changed. "+
					"This is a platform-side constraint that the Terraform provider cannot work around. "+
					"Original API error: %s",
				mcrUID, msg,
			)
	}
	return "Error Updating MCR", "Could not update MCR, unexpected error: " + msg
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
