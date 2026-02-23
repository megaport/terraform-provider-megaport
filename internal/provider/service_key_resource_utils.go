package provider

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	megaport "github.com/megaport/megaportgo"
)

type serviceKeyResourceModel struct {
	ProductUID types.String `tfsdk:"product_uid"`
	MaxSpeed    types.Int64  `tfsdk:"max_speed"`
	SingleUse   types.Bool   `tfsdk:"single_use"`
	Active      types.Bool   `tfsdk:"active"`
	Description types.String `tfsdk:"description"`
	VLAN        types.Int64  `tfsdk:"vlan"`
	PreApproved types.Bool   `tfsdk:"pre_approved"`
	ValidFor    types.Object `tfsdk:"valid_for"`
	Key         types.String `tfsdk:"key"`
	ProductName types.String `tfsdk:"product_name"`
	CompanyID   types.Int64  `tfsdk:"company_id"`
	CompanyUID  types.String `tfsdk:"company_uid"`
	CompanyName types.String `tfsdk:"company_name"`
	CreateDate  types.String `tfsdk:"create_date"`
	LastUsed    types.String `tfsdk:"last_used"`
	Expired     types.Bool   `tfsdk:"expired"`
	Valid       types.Bool   `tfsdk:"valid"`
	PromoCode   types.String `tfsdk:"promo_code"`
	LastUpdated types.String `tfsdk:"last_updated"`
}

type serviceKeyValidForModel struct {
	StartTime types.String `tfsdk:"start_time"`
	EndTime   types.String `tfsdk:"end_time"`
}

// fromAPI maps an API ServiceKey response to the Terraform model.
func (m *serviceKeyResourceModel) fromAPI(ctx context.Context, apiKey *megaport.ServiceKey) diag.Diagnostics {
	diags := diag.Diagnostics{}

	m.Key = types.StringValue(apiKey.Key)
	m.ProductUID = types.StringValue(apiKey.ProductUID)
	m.ProductName = types.StringValue(apiKey.ProductName)
	m.MaxSpeed = types.Int64Value(int64(apiKey.MaxSpeed))
	m.SingleUse = types.BoolValue(apiKey.SingleUse)
	m.Active = types.BoolValue(apiKey.Active)
	m.Description = types.StringValue(apiKey.Description)
	m.PreApproved = types.BoolValue(apiKey.PreApproved)
	m.CompanyID = types.Int64Value(int64(apiKey.CompanyID))
	m.CompanyUID = types.StringValue(apiKey.CompanyUID)
	m.CompanyName = types.StringValue(apiKey.CompanyName)
	m.Expired = types.BoolValue(apiKey.Expired)
	m.Valid = types.BoolValue(apiKey.Valid)
	m.PromoCode = types.StringValue(apiKey.PromoCode)

	if apiKey.VLAN != 0 {
		m.VLAN = types.Int64Value(int64(apiKey.VLAN))
	} else {
		m.VLAN = types.Int64Null()
	}

	if apiKey.CreateDate != nil {
		m.CreateDate = types.StringValue(apiKey.CreateDate.Time.UTC().Format(time.RFC3339))
	} else {
		m.CreateDate = types.StringNull()
	}

	if apiKey.LastUsed != nil {
		m.LastUsed = types.StringValue(apiKey.LastUsed.Time.UTC().Format(time.RFC3339))
	} else {
		m.LastUsed = types.StringNull()
	}

	if apiKey.ValidFor != nil {
		validForModel := &serviceKeyValidForModel{}
		if apiKey.ValidFor.StartTime != nil {
			validForModel.StartTime = types.StringValue(apiKey.ValidFor.StartTime.Time.UTC().Format(time.RFC3339))
		} else {
			validForModel.StartTime = types.StringNull()
		}
		if apiKey.ValidFor.EndTime != nil {
			validForModel.EndTime = types.StringValue(apiKey.ValidFor.EndTime.Time.UTC().Format(time.RFC3339))
		} else {
			validForModel.EndTime = types.StringNull()
		}
		validForObj, objDiags := types.ObjectValueFrom(ctx, serviceKeyValidForAttrs, validForModel)
		diags.Append(objDiags...)
		m.ValidFor = validForObj
	} else {
		m.ValidFor = types.ObjectNull(serviceKeyValidForAttrs)
	}

	return diags
}

// planToCreateRequest converts the Terraform plan to a CreateServiceKeyRequest.
func (m *serviceKeyResourceModel) planToCreateRequest(ctx context.Context) (*megaport.CreateServiceKeyRequest, diag.Diagnostics) {
	diags := diag.Diagnostics{}

	req := &megaport.CreateServiceKeyRequest{
		ProductUID:  m.ProductUID.ValueString(),
		MaxSpeed:    int(m.MaxSpeed.ValueInt64()),
		SingleUse:   m.SingleUse.ValueBool(),
		Active:      m.Active.ValueBool(),
		PreApproved: m.PreApproved.ValueBool(),
		Description: m.Description.ValueString(),
	}

	if !m.VLAN.IsNull() && !m.VLAN.IsUnknown() {
		req.VLAN = int(m.VLAN.ValueInt64())
	}

	if !m.ValidFor.IsNull() && !m.ValidFor.IsUnknown() {
		var validForModel serviceKeyValidForModel
		validDiags := m.ValidFor.As(ctx, &validForModel, basetypes.ObjectAsOptions{})
		diags.Append(validDiags...)
		if diags.HasError() {
			return nil, diags
		}

		validFor, parseDiags := parseValidFor(&validForModel)
		diags.Append(parseDiags...)
		if diags.HasError() {
			return nil, diags
		}
		req.ValidFor = validFor
	}

	return req, diags
}

// planToUpdateRequest builds an UpdateServiceKeyRequest from plan + state.
func planToUpdateRequest(ctx context.Context, plan, state *serviceKeyResourceModel) (*megaport.UpdateServiceKeyRequest, diag.Diagnostics) {
	diags := diag.Diagnostics{}

	req := &megaport.UpdateServiceKeyRequest{
		Key:        state.Key.ValueString(),
		ProductUID: state.ProductUID.ValueString(),
		SingleUse:  state.SingleUse.ValueBool(),
		Active:     plan.Active.ValueBool(),
	}

	if !plan.ValidFor.IsNull() && !plan.ValidFor.IsUnknown() {
		var validForModel serviceKeyValidForModel
		validDiags := plan.ValidFor.As(ctx, &validForModel, basetypes.ObjectAsOptions{})
		diags.Append(validDiags...)
		if diags.HasError() {
			return nil, diags
		}

		validFor, parseDiags := parseValidFor(&validForModel)
		diags.Append(parseDiags...)
		if diags.HasError() {
			return nil, diags
		}
		req.ValidFor = validFor
	}

	return req, diags
}

// parseValidFor converts the Terraform valid_for model into a megaport.ValidFor struct.
func parseValidFor(model *serviceKeyValidForModel) (*megaport.ValidFor, diag.Diagnostics) {
	diags := diag.Diagnostics{}
	validFor := &megaport.ValidFor{}

	if !model.StartTime.IsNull() && !model.StartTime.IsUnknown() && model.StartTime.ValueString() != "" {
		t, err := time.Parse(time.RFC3339, model.StartTime.ValueString())
		if err != nil {
			diags.AddError(
				"Invalid start_time",
				"Could not parse start_time as RFC3339: "+err.Error(),
			)
			return nil, diags
		}
		validFor.StartTime = &megaport.Time{Time: t}
	}

	if !model.EndTime.IsNull() && !model.EndTime.IsUnknown() && model.EndTime.ValueString() != "" {
		t, err := time.Parse(time.RFC3339, model.EndTime.ValueString())
		if err != nil {
			diags.AddError(
				"Invalid end_time",
				"Could not parse end_time as RFC3339: "+err.Error(),
			)
			return nil, diags
		}
		validFor.EndTime = &megaport.Time{Time: t}
	}

	return validFor, diags
}
