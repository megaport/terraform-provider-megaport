package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	megaport "github.com/megaport/megaportgo"
)

// mcrAsnAttachedVXCSentinel is megalith's 400 message for an MCR ASN change while live VXCs are attached.
// The NAT Gateway message ends "to this NAT Gateway" instead.
const mcrAsnAttachedVXCSentinel = "Cannot update ASN while VXCs are attached to this MCR"

func isMCRAsnAttachedVXCError(err error) bool {
	var apiErr *megaport.ErrorResponse
	if !errors.As(err, &apiErr) || apiErr.Response == nil {
		return false
	}
	return apiErr.Response.StatusCode == http.StatusBadRequest &&
		strings.Contains(apiErr.Message+" "+apiErr.Data, mcrAsnAttachedVXCSentinel)
}

// mapMCRUpdateError adds the workaround to a known ASN rejection and keeps the generic diagnostic for anything else.
func mapMCRUpdateError(err error, mcrUID string) (summary, detail string) {
	if isMCRAsnAttachedVXCError(err) {
		return "Cannot update MCR ASN while VXCs are attached",
			fmt.Sprintf(
				"The Megaport API rejected the ASN change on MCR %s because it has live VXCs attached. "+
					"Delete the attached VXCs, then apply the ASN change again. "+
					"Original API error: %s",
				mcrUID, err.Error(),
			)
	}
	return "Error Updating MCR", "Could not update MCR, unexpected error: " + err.Error()
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
