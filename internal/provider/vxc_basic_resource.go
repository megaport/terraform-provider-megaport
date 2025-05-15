package provider

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	megaport "github.com/megaport/megaportgo"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &vxcBasicResource{}
	_ resource.ResourceWithConfigure   = &vxcBasicResource{}
	_ resource.ResourceWithImportState = &vxcBasicResource{}

	vxcBasicEndConfigurationAttrs = map[string]attr.Type{
		"requested_product_uid": types.StringType,
		"current_product_uid":   types.StringType,
		"vlan":                  types.Int64Type,
		"inner_vlan":            types.Int64Type,
		"vnic_index":            types.Int64Type,
	}
)

// vxcBasicResourceModel maps the resource schema data.
type vxcBasicResourceModel struct {
	LastUpdated types.String `tfsdk:"last_updated"`

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

// NewVXCBasicResource is a helper function to simplify the provider implementation.
func NewVXCBasicResource() resource.Resource {
	return &vxcBasicResource{}
}

// vxcBasicResource is the resource implementation.
type vxcBasicResource struct {
	client *megaport.Client
}

// Configure adds the provider configured client to the resource.
func (r *vxcBasicResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*megaport.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *megaport.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

// Schema defines the schema for the resource.
func (r *vxcBasicResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Virtual Cross Connect (VXC) Resource for the Megaport Terraform Provider. This resource allows you to create, modify, and update VXCs. VXCs are Layer 2 Ethernet circuits providing private, flexible, and on-demand connections between any of the locations on the Megaport network with 1 Mbps to 100 Gbps of capacity. This is a basic resource for VXC management.",
		Attributes: map[string]schema.Attribute{
			"last_updated": schema.StringAttribute{
				Description: "The last time the resource was updated.",
				Computed:    true,
			},
			"product_uid": schema.StringAttribute{
				Description: "The unique identifier for the resource.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"service_key": schema.StringAttribute{
				Description: "The service key of the VXC.",
				Optional:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"product_name": schema.StringAttribute{
				Description: "The name of the product.",
				Required:    true,
			},
			"rate_limit": schema.Int64Attribute{
				Description: "The rate limit of the product.",
				Required:    true,
			},
			"provisioning_status": schema.StringAttribute{
				Description: "The provisioning status of the product.",
				Computed:    true,
			},
			"promo_code": schema.StringAttribute{
				Description: "Promo code is an optional string that can be used to enter a promotional code for the service order. The code is not validated, so if the code doesn't exist or doesn't work for the service, the request will still be successful.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"contract_term_months": schema.Int64Attribute{
				Description: "The term of the contract in months: valid values are 1, 12, 24, and 36. To set the product to a month-to-month contract with no minimum term, set the value to 1.",
				Required:    true,
				Validators: []validator.Int64{
					int64validator.OneOf(1, 12, 24, 36),
				},
			},
			"shutdown": schema.BoolAttribute{
				Description: "Temporarily shut down and re-enable the VXC. Valid values are true (shut down) and false (enabled). If not provided, it defaults to false (enabled).",
				Computed:    true,
				Optional:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"cost_centre": schema.StringAttribute{
				Description: "A customer reference number to be included in billing information and invoices. Also known as the service level reference (SLR) number. Specify a unique identifying number for the product to be used for billing purposes, such as a cost center number or a unique customer ID. The service level reference number appears for each service under the Product section of the invoice. You can also edit this field for an existing service.",
				Computed:    true,
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
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
			"locked": schema.BoolAttribute{
				Description: "Whether the product is locked.",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"admin_locked": schema.BoolAttribute{
				Description: "Whether the product is admin locked.",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"cancelable": schema.BoolAttribute{
				Description: "Whether the product is cancelable.",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"a_end": getEndConfigSchema(VXC_A_END),
			"b_end": getEndConfigSchema(VXC_B_END),
			"a_end_partner_config": schema.SingleNestedAttribute{
				Description: `The partner configuration of the A-End order configuration. Contains CSP and/or BGP Configuration settings. For any partner configuration besides "vrouter", this configuration cannot be changed after the VXC is created and if it is modified, the VXC will be deleted and re-created. Imported VXCs do not have this field populated by the API, so the initially provided configuration will be ignored as it can't be verified to be correct. If the user wants to change the configuration after importing the resource, they can then do so by changing the field after importing the resource and running terraform apply.`,
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"partner":              partnerTypeSchema,
					"aws_config":           awsPartnerConfigSchema,
					"azure_config":         azurePartnerConfigSchema,
					"google_config":        googlePartnerConfigSchema,
					"ibm_config":           ibmPartnerConfigSchema,
					"oracle_config":        oraclePartnerConfigSchema,
					"vrouter_config":       vrouterPartnerConfigSchema,
					"partner_a_end_config": aEndPartnerConfigSchema,
				},
			},
			"b_end_partner_config": schema.SingleNestedAttribute{
				Description: `The partner configuration of the B-End order configuration. Contains CSP and/or BGP Configuration settings. For any partner configuration besides "vrouter", this configuration cannot be changed after the VXC is created and if it is modified, the VXC will be deleted and re-created. Imported VXCs do not have this field populated by the API, so the initially provided configuration will be ignored as it can't be verified to be correct. If the user wants to change the configuration after importing the resource, they can then do so by changing the field after importing the resource and running terraform apply.`,
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"partner":              partnerTypeSchema,
					"aws_config":           awsPartnerConfigSchema,
					"azure_config":         azurePartnerConfigSchema,
					"google_config":        googlePartnerConfigSchema,
					"ibm_config":           ibmPartnerConfigSchema,
					"oracle_config":        oraclePartnerConfigSchema,
					"vrouter_config":       vrouterPartnerConfigSchema,
					"partner_a_end_config": aEndPartnerConfigSchema,
				},
			},
		},
	}
}

// Create a new resource.
func (r *vxcBasicResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan vxcBasicResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	buyReq := &megaport.BuyVXCRequest{
		VXCName:    plan.Name.ValueString(),
		Term:       int(plan.ContractTermMonths.ValueInt64()),
		RateLimit:  int(plan.RateLimit.ValueInt64()),
		PromoCode:  plan.PromoCode.ValueString(),
		CostCentre: plan.CostCentre.ValueString(),
		ServiceKey: plan.ServiceKey.ValueString(),

		WaitForProvision: true,
		WaitForTime:      waitForTime,
	}

	if !plan.Shutdown.IsNull() {
		buyReq.Shutdown = plan.Shutdown.ValueBool()
	}

	if !plan.ResourceTags.IsNull() {
		tagMap, tagDiags := toResourceTagMap(ctx, plan.ResourceTags)
		resp.Diagnostics.Append(tagDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		buyReq.ResourceTags = tagMap
	}

	aEndObj := plan.AEndConfiguration
	bEndObj := plan.BEndConfiguration

	var a vxcBasicEndConfigurationModel
	aEndDiags := aEndObj.As(ctx, &a, basetypes.ObjectAsOptions{})
	if aEndDiags.HasError() {
		resp.Diagnostics.Append(aEndDiags...)
		return
	}

	buyReq.PortUID = a.RequestedProductUID.ValueString()

	endDiags, aEndMegaportConfig, aEndPartnerConfig := r.createVXCBasicEndConfiguration(ctx, plan.Name.ValueString(), int(plan.RateLimit.ValueInt64()), a, plan.AEndPartnerConfig)
	resp.Diagnostics.Append(endDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.AEndPartnerConfig = aEndPartnerConfig
	buyReq.AEndConfiguration = aEndMegaportConfig

	var b vxcBasicEndConfigurationModel
	bEndDiags := bEndObj.As(ctx, &b, basetypes.ObjectAsOptions{})
	if bEndDiags.HasError() {
		resp.Diagnostics.Append(bEndDiags...)
		return
	}

	endDiags, bEndMegaportConfig, bEndPartnerConfig := r.createVXCBasicEndConfiguration(ctx, plan.Name.ValueString(), int(plan.RateLimit.ValueInt64()), b, plan.BEndPartnerConfig)
	resp.Diagnostics.Append(endDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.BEndPartnerConfig = bEndPartnerConfig
	buyReq.BEndConfiguration = bEndMegaportConfig

	err := r.client.VXCService.ValidateVXCOrder(ctx, buyReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Validation error while attempting to create VXC",
			"Validation error while attempting to create VXC with name "+plan.Name.ValueString()+": "+err.Error(),
		)
		return
	}

	createdVXC, err := r.client.VXCService.BuyVXC(ctx, buyReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating VXC",
			"Could not order VXC with name "+plan.Name.ValueString()+": "+err.Error(),
		)
		return
	}

	createdID := createdVXC.TechnicalServiceUID

	// get the created VXC
	vxc, err := r.client.VXCService.GetVXC(ctx, createdID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading newly created VXC",
			"Could not read newly created VXC with ID "+createdID+": "+err.Error(),
		)
		return
	}

	tags, err := r.client.VXCService.ListVXCResourceTags(ctx, createdID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading tags for newly created VXC",
			"Could not read tags for newly created VXC with ID "+createdID+": "+err.Error(),
		)
		return
	}

	// update the plan with the VXC info
	apiDiags := plan.fromAPIVXC(ctx, vxc, tags)
	resp.Diagnostics.Append(apiDiags...)

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read resource information.
func (r *vxcBasicResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state vxcBasicResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get refreshed vxc value from API
	vxc, err := r.client.VXCService.GetVXC(ctx, state.UID.ValueString())
	if err != nil {
		// VXC has been deleted or is not found
		if mpErr, ok := err.(*megaport.ErrorResponse); ok {
			if mpErr.Response.StatusCode == http.StatusNotFound ||
				(mpErr.Response.StatusCode == http.StatusBadRequest && strings.Contains(mpErr.Message, "Could not find a service with UID")) {
				resp.State.RemoveResource(ctx)
				return
			}
		}

		resp.Diagnostics.AddError(
			"Error Reading VXC",
			"Could not read VXC with ID "+state.UID.ValueString()+": "+err.Error(),
		)
		return
	}

	// If the vxc has been deleted
	if vxc.ProvisioningStatus == megaport.STATUS_DECOMMISSIONED {
		resp.State.RemoveResource(ctx)
		return
	}

	// Get tags
	tags, err := r.client.VXCService.ListVXCResourceTags(ctx, state.UID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading tags for VXC",
			"Could not read tags for VXC with ID "+state.UID.ValueString()+": "+err.Error(),
		)
		return
	}

	apiDiags := state.fromAPIVXC(ctx, vxc, tags)
	resp.Diagnostics.Append(apiDiags...)

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)

	aEndConfig := &vxcBasicEndConfigurationModel{}
	bEndConfig := &vxcBasicEndConfigurationModel{}
	aEndConfigDiags := state.AEndConfiguration.As(ctx, aEndConfig, basetypes.ObjectAsOptions{})
	bEndConfigDiags := state.BEndConfiguration.As(ctx, bEndConfig, basetypes.ObjectAsOptions{})
	resp.Diagnostics.Append(aEndConfigDiags...)
	resp.Diagnostics.Append(bEndConfigDiags...)
}

func (r *vxcBasicResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state vxcBasicResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := &megaport.UpdateVXCRequest{
		WaitForUpdate: true,
		WaitForTime:   waitForTime,
	}

	if !plan.Name.Equal(state.Name) {
		updateReq.Name = megaport.PtrTo(plan.Name.ValueString())
	}

	aEndDiags, aEndState, aEndPartnerObj, aEndMegaportPartnerConfig, aEndRequestedProductUID, aEndVLAN, aEndInnerVLAN, aEndCSP := r.makeUpdateEndConfig(ctx, plan.Name.ValueString(), plan.AEndConfiguration, state.AEndConfiguration, plan.AEndPartnerConfig, state.AEndPartnerConfig)
	resp.Diagnostics.Append(aEndDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !aEndCSP && !aEndPartnerObj.IsNull() {
		// Only update the partner config if it is not a CSP partner config.
		state.AEndPartnerConfig = aEndPartnerObj
	}

	updateReq.AEndPartnerConfig = aEndMegaportPartnerConfig
	updateReq.AEndVLAN = aEndVLAN
	updateReq.AEndInnerVLAN = aEndInnerVLAN
	updateReq.AEndProductUID = aEndRequestedProductUID

	bEndDiags, bEndState, bEndPartnerObj, bEndMegaportPartnerConfig, bEndRequestedProductUID, bEndVLAN, bEndInnerVLAN, bEndCSP := r.makeUpdateEndConfig(ctx, plan.Name.ValueString(), plan.BEndConfiguration, state.BEndConfiguration, plan.BEndPartnerConfig, state.BEndPartnerConfig)
	resp.Diagnostics.Append(bEndDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Only update the partner config if it is not a CSP partner config.
	if !bEndCSP && !bEndPartnerObj.IsNull() {
		state.BEndPartnerConfig = bEndPartnerObj
	}
	updateReq.BEndPartnerConfig = bEndMegaportPartnerConfig
	updateReq.BEndVLAN = bEndVLAN
	updateReq.BEndInnerVLAN = bEndInnerVLAN
	updateReq.BEndProductUID = bEndRequestedProductUID

	if !plan.RateLimit.IsNull() && !plan.RateLimit.Equal(state.RateLimit) {
		updateReq.RateLimit = megaport.PtrTo(int(plan.RateLimit.ValueInt64()))
	}

	if !plan.CostCentre.IsNull() && !plan.CostCentre.Equal(state.CostCentre) {
		updateReq.CostCentre = megaport.PtrTo(plan.CostCentre.ValueString())
	}

	if !plan.Shutdown.IsNull() && !plan.Shutdown.Equal(state.Shutdown) {
		updateReq.Shutdown = megaport.PtrTo(plan.Shutdown.ValueBool())
	}

	if !plan.ContractTermMonths.IsNull() && !plan.ContractTermMonths.Equal(state.ContractTermMonths) {
		updateReq.Term = megaport.PtrTo(int(plan.ContractTermMonths.ValueInt64()))
	}

	_, err := r.client.VXCService.UpdateVXC(ctx, plan.UID.ValueString(), updateReq)

	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating VXC",
			"Could not update VXC with ID "+state.UID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Set updated state
	state.AEndConfiguration = aEndState
	state.BEndConfiguration = bEndState
	state.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	// Get refreshed vxc value from API
	vxc, err := r.client.VXCService.GetVXC(ctx, state.UID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading VXC",
			"Could not read VXC with ID "+state.UID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Save partner configurations before calling fromAPIVXC
	savedAEndPartnerConfig := state.AEndPartnerConfig
	savedBEndPartnerConfig := state.BEndPartnerConfig

	if !plan.ResourceTags.Equal(state.ResourceTags) {
		tagMap, tagDiags := toResourceTagMap(ctx, plan.ResourceTags)
		resp.Diagnostics.Append(tagDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		err := r.client.VXCService.UpdateVXCResourceTags(ctx, state.UID.ValueString(), tagMap)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating tags for VXC",
				"Could not update tags for VXC with ID "+state.UID.ValueString()+": "+err.Error(),
			)
			return
		}
	}

	// Get resource tags
	tags, err := r.client.VXCService.ListVXCResourceTags(ctx, state.UID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading VXC Tags",
			"Could not read VXC tags with ID "+state.UID.ValueString()+": "+err.Error(),
		)
		return
	}

	apiDiags := state.fromAPIVXC(ctx, vxc, tags)
	state.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	resp.Diagnostics.Append(apiDiags...)

	state.AEndPartnerConfig = savedAEndPartnerConfig
	state.BEndPartnerConfig = savedBEndPartnerConfig

	// Set refreshed state
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Delete deletes the resource and removes the Terraform state on success.
func (r *vxcBasicResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state vxcBasicResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete existing order
	err := r.client.VXCService.DeleteVXC(ctx, state.UID.ValueString(), &megaport.DeleteVXCRequest{
		DeleteNow: true,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting VXC",
			"Could not delete VXC, unexpected error: "+err.Error(),
		)
		return
	}
}

func (r *vxcBasicResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Get current state
	var plan, state vxcBasicResourceModel
	diags := diag.Diagnostics{}

	if !req.Plan.Raw.IsNull() {
		planDiags := req.Plan.Get(ctx, &plan)
		resp.Diagnostics.Append(planDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	if !req.State.Raw.IsNull() {
		stateDiags := req.State.Get(ctx, &state)
		resp.Diagnostics.Append(stateDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// If VXC is not yet created, return
	if !state.UID.IsNull() {
		if !req.Plan.Raw.IsNull() {
			aEndStateObj := state.AEndConfiguration
			bEndStateObj := state.BEndConfiguration

			diags, endPlanObj, statePlanObj, statePartner, requiresReplace := modifyPlanBasicEndConfig(ctx, plan.AEndConfiguration, aEndStateObj, plan.AEndPartnerConfig, state.AEndPartnerConfig)
			resp.Diagnostics.Append(diags...)
			if diags.HasError() {
				return
			}
			plan.AEndConfiguration = endPlanObj
			state.AEndConfiguration = statePlanObj
			state.AEndPartnerConfig = statePartner
			if requiresReplace != nil {
				resp.RequiresReplace = append(resp.RequiresReplace, requiresReplace...)
			}
			diags, endPlanObj, statePlanObj, statePartner, requiresReplace = modifyPlanBasicEndConfig(ctx, plan.BEndConfiguration, bEndStateObj, plan.BEndPartnerConfig, state.BEndPartnerConfig)
			resp.Diagnostics.Append(diags...)
			if diags.HasError() {
				return
			}
			if requiresReplace != nil {
				resp.RequiresReplace = append(resp.RequiresReplace, requiresReplace...)
			}
			plan.BEndConfiguration = endPlanObj
			state.BEndConfiguration = statePlanObj
			state.BEndPartnerConfig = statePartner

			req.Plan.Set(ctx, &plan)
			resp.Plan.Set(ctx, &plan)
			stateDiags := req.State.Set(ctx, &state)
			diags = append(diags, stateDiags...)
			resp.Diagnostics.Append(diags...)
		}
	}

	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *vxcBasicResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Retrieve import ID and save to id attribute
	resource.ImportStatePassthroughID(ctx, path.Root("product_uid"), req, resp)
}

// Metadata returns the resource type name.
func (r *vxcBasicResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vxc_basic"
}
