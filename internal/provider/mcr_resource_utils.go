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

// mcrAsnAttachedVXCSentinel is the platform's 400 message for a rejected MCR ASN update.
// NAT Gateway shares this error path but ends "to this NAT Gateway" instead, so the qualifier stays MCR-specific.
const mcrAsnAttachedVXCSentinel = "Cannot update ASN while VXCs are attached to this MCR"

// isMCRAsnAttachedVXCError reports whether err is the platform's HTTP 400
// rejecting an MCR ASN change because live VXCs are still attached. Anything
// that is not a 400 *megaport.ErrorResponse (e.g. a WaitForUpdate poll timeout)
// returns false and falls through to the generic diagnostic.
func isMCRAsnAttachedVXCError(err error) bool {
	var apiErr *megaport.ErrorResponse
	if !errors.As(err, &apiErr) || apiErr.Response == nil {
		return false
	}
	return apiErr.Response.StatusCode == http.StatusBadRequest &&
		strings.Contains(apiErr.Message+" "+apiErr.Data, mcrAsnAttachedVXCSentinel)
}

// mapMCRUpdateError turns an error from MCRService.ModifyMCR into a Terraform
// diagnostic (summary, detail). The known ASN-while-attached constraint gets
// richer provider-side guidance; everything else (including WaitForUpdate poll
// timeouts) falls through to the historical generic Update diagnostic so we
// don't hide novel failure modes from users. err must be non-nil; callers
// invoke this only on the error path.
func mapMCRUpdateError(err error, mcrUID string) (summary, detail string) {
	if isMCRAsnAttachedVXCError(err) {
		return "Cannot update MCR ASN while VXCs are attached",
			fmt.Sprintf(
				"The Megaport API rejected the ASN update on MCR %s because it has live VXC connections. "+
					"The platform does not currently support changing an MCR's ASN while VXCs are attached; "+
					"all attached VXCs must be deleted before the ASN can be changed. "+
					"This is a platform-side constraint that the Terraform provider cannot work around. "+
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
