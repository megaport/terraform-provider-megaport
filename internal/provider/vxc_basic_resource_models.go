package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	megaport "github.com/megaport/megaportgo"
)

var (
	vxcBasicEndConfigurationAttrs = map[string]attr.Type{
		"requested_product_uid": types.StringType,
		"current_product_uid":   types.StringType,
		"vlan":                  types.Int64Type,
		"inner_vlan":            types.Int64Type,
		"vnic_index":            types.Int64Type,
	}
	vxcBasicPartnerConfigAttrs = map[string]attr.Type{
		"partner":       types.StringType,
		"aws_config":    types.ObjectType{}.WithAttributeTypes(vxcPartnerConfigAWSAttrs),
		"azure_config":  types.ObjectType{}.WithAttributeTypes(vxcPartnerConfigAzureAttrs),
		"google_config": types.ObjectType{}.WithAttributeTypes(vxcPartnerConfigGoogleAttrs),
		"oracle_config": types.ObjectType{}.WithAttributeTypes(vxcPartnerConfigOracleAttrs),
		"mcr_config":    types.ObjectType{}.WithAttributeTypes(vxcPartnerConfigVrouterAttrs),
		"ibm_config":    types.ObjectType{}.WithAttributeTypes(vxcPartnerConfigIbmAttrs),
	}
	vxcPartnerConfigMCRAttrs = map[string]attr.Type{
		"interfaces": types.ListType{}.WithElementType(types.ObjectType{}.WithAttributeTypes(vxcVrouterInterfaceAttrs)),
	}
)

func (orm *vxcBasicResourceModel) fromAPIVXC(ctx context.Context, v *megaport.VXC, tags map[string]string) diag.Diagnostics {
	apiDiags := diag.Diagnostics{}

	orm.UID = types.StringValue(v.UID)
	orm.Name = types.StringValue(v.Name)
	orm.RateLimit = types.Int64Value(int64(v.RateLimit))
	orm.ProvisioningStatus = types.StringValue(v.ProvisioningStatus)
	orm.ContractTermMonths = types.Int64Value(int64(v.ContractTermMonths))
	orm.Shutdown = types.BoolValue(v.Shutdown)
	orm.CostCentre = types.StringValue(v.CostCentre)
	orm.Locked = types.BoolValue(v.Locked)
	orm.AdminLocked = types.BoolValue(v.AdminLocked)
	orm.Cancelable = types.BoolValue(v.Cancelable)

	var aEndRequestedProductUID, bEndRequestedProductUID string
	if !orm.AEndConfiguration.IsNull() {
		existingAEnd := &vxcBasicEndConfigurationModel{}
		aEndDiags := orm.AEndConfiguration.As(ctx, existingAEnd, basetypes.ObjectAsOptions{})
		apiDiags = append(apiDiags, aEndDiags...)
		aEndRequestedProductUID = existingAEnd.RequestedProductUID.ValueString()
	}

	aEndModel := &vxcBasicEndConfigurationModel{
		RequestedProductUID:   types.StringValue(aEndRequestedProductUID),
		CurrentProductUID:     types.StringValue(v.AEndConfiguration.UID),
		NetworkInterfaceIndex: types.Int64Value(int64(v.AEndConfiguration.NetworkInterfaceIndex)),
	}
	if v.AEndConfiguration.InnerVLAN == 0 {
		aEndModel.InnerVLAN = types.Int64PointerValue(nil)
	} else {
		aEndModel.InnerVLAN = types.Int64Value(int64(v.AEndConfiguration.InnerVLAN))
	}
	if v.AEndConfiguration.VLAN == 0 {
		aEndModel.VLAN = types.Int64PointerValue(nil)
	} else {
		aEndModel.VLAN = types.Int64Value(int64(v.AEndConfiguration.VLAN))
	}
	aEnd, aEndDiags := types.ObjectValueFrom(ctx, vxcBasicEndConfigurationAttrs, aEndModel)
	apiDiags = append(apiDiags, aEndDiags...)
	orm.AEndConfiguration = aEnd

	if !orm.BEndConfiguration.IsNull() {
		existingBEnd := &vxcBasicEndConfigurationModel{}
		bEndDiags := orm.BEndConfiguration.As(ctx, existingBEnd, basetypes.ObjectAsOptions{})
		apiDiags = append(apiDiags, bEndDiags...)
		bEndRequestedProductUID = existingBEnd.RequestedProductUID.ValueString()
	}

	bEndModel := &vxcBasicEndConfigurationModel{
		RequestedProductUID:   types.StringValue(bEndRequestedProductUID),
		CurrentProductUID:     types.StringValue(v.BEndConfiguration.UID),
		NetworkInterfaceIndex: types.Int64Value(int64(v.BEndConfiguration.NetworkInterfaceIndex)),
	}
	if v.BEndConfiguration.InnerVLAN == 0 {
		bEndModel.InnerVLAN = types.Int64PointerValue(nil)
	} else {
		bEndModel.InnerVLAN = types.Int64Value(int64(v.BEndConfiguration.InnerVLAN))
	}
	if v.BEndConfiguration.VLAN == 0 {
		bEndModel.VLAN = types.Int64PointerValue(nil)
	} else {
		bEndModel.VLAN = types.Int64Value(int64(v.BEndConfiguration.VLAN))
	}
	bEnd, bEndDiags := types.ObjectValueFrom(ctx, vxcBasicEndConfigurationAttrs, bEndModel)
	apiDiags = append(apiDiags, bEndDiags...)
	orm.BEndConfiguration = bEnd

	if len(tags) > 0 {
		resourceTags, tagDiags := types.MapValueFrom(ctx, types.StringType, tags)
		apiDiags = append(apiDiags, tagDiags...)
		orm.ResourceTags = resourceTags
	} else {
		orm.ResourceTags = types.MapNull(types.StringType)
	}

	return apiDiags
}

// vxcBasicResourceModel maps the resource schema data.
type vxcBasicResourceModel struct {
	UID                types.String `tfsdk:"product_uid"`
	Name               types.String `tfsdk:"product_name"`
	RateLimit          types.Int64  `tfsdk:"rate_limit"`
	ProvisioningStatus types.String `tfsdk:"provisioning_status"`
	PromoCode          types.String `tfsdk:"promo_code"`
	ServiceKey         types.String `tfsdk:"service_key"`

	ContractTermMonths types.Int64  `tfsdk:"contract_term_months"`
	Locked             types.Bool   `tfsdk:"locked"`
	AdminLocked        types.Bool   `tfsdk:"admin_locked"`
	Cancelable         types.Bool   `tfsdk:"cancelable"`
	CostCentre         types.String `tfsdk:"cost_centre"`
	Shutdown           types.Bool   `tfsdk:"shutdown"`

	AEndConfiguration types.Object `tfsdk:"a_end"`
	BEndConfiguration types.Object `tfsdk:"b_end"`

	AEndPartnerConfig types.Object `tfsdk:"a_end_partner_config"`
	BEndPartnerConfig types.Object `tfsdk:"b_end_partner_config"`

	ResourceTags types.Map `tfsdk:"resource_tags"`
}

// vxcBasicEndConfigurationModel maps the end configuration schema data.
type vxcBasicEndConfigurationModel struct {
	RequestedProductUID   types.String `tfsdk:"requested_product_uid"`
	CurrentProductUID     types.String `tfsdk:"current_product_uid"`
	VLAN                  types.Int64  `tfsdk:"vlan"`
	InnerVLAN             types.Int64  `tfsdk:"inner_vlan"`
	NetworkInterfaceIndex types.Int64  `tfsdk:"vnic_index"`
}

type vxcBasicPartnerConfigurationModel struct {
	Partner             types.String `tfsdk:"partner"`
	AWSPartnerConfig    types.Object `tfsdk:"aws_config"`
	AzurePartnerConfig  types.Object `tfsdk:"azure_config"`
	GooglePartnerConfig types.Object `tfsdk:"google_config"`
	OraclePartnerConfig types.Object `tfsdk:"oracle_config"`
	MCRPartnerConfig    types.Object `tfsdk:"mcr_config"`
	IBMPartnerConfig    types.Object `tfsdk:"ibm_config"`
}
