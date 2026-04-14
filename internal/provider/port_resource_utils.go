package provider

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	megaport "github.com/megaport/megaportgo"
)

// portResourcesAttrs and portInterfaceAttrs define the Terraform type maps used when
// converting API port responses into typed Terraform object values.
var (
	portResourcesAttrs = map[string]attr.Type{
		"interface": types.ObjectType{}.WithAttributeTypes(portInterfaceAttrs),
	}

	portInterfaceAttrs = map[string]attr.Type{
		"demarcation": types.StringType,
		"up":          types.Int64Type,
	}

	// contractTermMonthsValidator is the shared validator for port contract terms.
	contractTermMonthsValidator = []validator.Int64{
		int64validator.OneOf(1, 12, 24, 36, 48, 60),
	}
)

type portResourcesModel struct {
	Interface types.Object `tfsdk:"interface"`
}

type portInterfaceModel struct {
	Demarcation types.String `tfsdk:"demarcation"`
	Up          types.Int64  `tfsdk:"up"`
}

// fromAPIPortInterface converts a megaportgo PortInterface into a Terraform object value.
func fromAPIPortInterface(ctx context.Context, p *megaport.PortInterface) (types.Object, diag.Diagnostics) {
	m := &portInterfaceModel{
		Demarcation: types.StringValue(p.Demarcation),
		Up:          types.Int64Value(int64(p.Up)),
	}
	return types.ObjectValueFrom(ctx, portInterfaceAttrs, m)
}

// portModifyParams holds resolved field values for a ModifyPort API call.
type portModifyParams struct {
	name                  string
	costCentre            string
	marketplaceVisibility bool
	contractTermMonths    *int
}

// resolvePortModifyParams computes the effective field values for a ModifyPort request by
// comparing plan and state. costCentre always comes from the plan to support clearing the value.
func resolvePortModifyParams(
	planName, stateName types.String,
	planVisibility, stateVisibility types.Bool,
	planCostCentre types.String,
	planTerm, stateTerm types.Int64,
) portModifyParams {
	p := portModifyParams{
		costCentre: planCostCentre.ValueString(),
	}
	if !planName.Equal(stateName) {
		p.name = planName.ValueString()
	} else {
		p.name = stateName.ValueString()
	}
	if !planVisibility.Equal(stateVisibility) {
		p.marketplaceVisibility = planVisibility.ValueBool()
	} else {
		p.marketplaceVisibility = stateVisibility.ValueBool()
	}
	if !planTerm.Equal(stateTerm) {
		months := int(planTerm.ValueInt64())
		p.contractTermMonths = &months
	}
	return p
}

// lagPortUIDsList converts a slice of UID strings into a Terraform list value.
// Returns a null list when uids is empty.
func lagPortUIDsList(uids []string) (types.List, diag.Diagnostics) {
	if len(uids) == 0 {
		return types.ListNull(types.StringType), nil
	}
	vals := make([]attr.Value, len(uids))
	for i, uid := range uids {
		vals[i] = types.StringValue(uid)
	}
	return types.ListValue(types.StringType, vals)
}

// isPortNotFoundError reports whether an API error indicates the port no longer exists.
func isPortNotFoundError(err error) bool {
	mpErr, ok := err.(*megaport.ErrorResponse)
	if !ok || mpErr.Response == nil {
		return false
	}
	return mpErr.Response.StatusCode == http.StatusNotFound ||
		(mpErr.Response.StatusCode == http.StatusBadRequest &&
			strings.Contains(mpErr.Message, "Could not find a service with UID"))
}

// syncPortResourceTags updates resource tags on a port when the plan value differs from state.
func syncPortResourceTags(ctx context.Context, plan, state types.Map, uid string, client *megaport.Client) diag.Diagnostics {
	diags := diag.Diagnostics{}
	if plan.Equal(state) {
		return diags
	}
	tagMap, tagDiags := toResourceTagMap(ctx, plan)
	diags.Append(tagDiags...)
	if diags.HasError() {
		return diags
	}
	if err := client.PortService.UpdatePortResourceTags(ctx, uid, tagMap); err != nil {
		diags.AddError(
			"Error updating port resource tags",
			fmt.Sprintf("Could not update resource tags for port %s: %s", uid, err),
		)
	}
	return diags
}

// commonPortSchemaAttrs returns schema attributes shared by both port resource types.
// Callers add their own product_name, port_speed, and marketplace_visibility.
func commonPortSchemaAttrs() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"product_uid": schema.StringAttribute{
			Description: "The unique identifier for the resource.",
			Computed:    true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"contract_term_months": schema.Int64Attribute{
			Description: "The term of the contract in months: valid values are 1, 12, 24, 36, 48, and 60. To set the product to a month-to-month contract with no minimum term, set the value to 1.",
			Required:    true,
			Validators:  contractTermMonthsValidator,
		},
		"company_uid": schema.StringAttribute{
			Description: "The unique identifier of the company.",
			Computed:    true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"cost_centre": schema.StringAttribute{
			Description: "A customer reference number to be included in billing information and invoices. Also known as the service level reference (SLR) number. Specify a unique identifying number for the product to be used for billing purposes, such as a cost center number or a unique customer ID. The service level reference number appears for each service under the Product section of the invoice. You can also edit this field for an existing service.",
			Optional:    true,
			Computed:    true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"promo_code": schema.StringAttribute{
			Description: "An optional promotional code for the service order. The code is not validated — if it doesn't exist or doesn't apply, the request will still succeed.",
			Optional:    true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"location_id": schema.Int64Attribute{
			Description: "The numeric location ID of the product. This value can be retrieved from the data source megaport_location.",
			Required:    true,
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.RequiresReplace(),
			},
		},
		"diversity_zone": schema.StringAttribute{
			Description: "The diversity zone of the product.",
			Optional:    true,
			Computed:    true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
				stringplanmodifier.RequiresReplace(),
			},
		},
		"resources": schema.SingleNestedAttribute{
			Description: "Resources attached to port.",
			Computed:    true,
			Attributes: map[string]schema.Attribute{
				"interface": schema.SingleNestedAttribute{
					Description: "Port interface details.",
					Optional:    true,
					Computed:    true,
					Attributes: map[string]schema.Attribute{
						"demarcation": schema.StringAttribute{
							Description: "The demarcation of the interface.",
							Computed:    true,
						},
						"up": schema.Int64Attribute{
							Description: "The up status of the interface.",
							Computed:    true,
						},
					},
				},
			},
		},
		"resource_tags": schema.MapAttribute{
			Description: "The resource tags associated with the product.",
			Optional:    true,
			ElementType: types.StringType,
			PlanModifiers: []planmodifier.Map{
				mapplanmodifier.UseStateForUnknown(),
			},
		},
	}
}
