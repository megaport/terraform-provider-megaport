package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	megaport "github.com/megaport/megaportgo"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &vxcResource{}
	_ resource.ResourceWithConfigure   = &vxcResource{}
	_ resource.ResourceWithImportState = &vxcResource{}
)

// vxcResourceModel maps the resource schema data.
type vxcResourceModel struct {
	LastUpdated types.String `tfsdk:"last_updated"`

	ID                 types.Int64  `tfsdk:"product_id"`
	UID                types.String `tfsdk:"product_uid"`
	ServiceID          types.Int64  `tfsdk:"service_id"`
	Name               types.String `tfsdk:"product_name"`
	Type               types.String `tfsdk:"product_type"`
	RateLimit          types.Int64  `tfsdk:"rate_limit"`
	DistanceBand       types.String `tfsdk:"distance_band"`
	ProvisioningStatus types.String `tfsdk:"provisioning_status"`

	SecondaryName  types.String `tfsdk:"secondary_name"`
	UsageAlgorithm types.String `tfsdk:"usage_algorithm"`
	CreatedBy      types.String `tfsdk:"created_by"`

	ContractTermMonths types.Int64                   `tfsdk:"contract_term_months"`
	CompanyUID         types.String                  `tfsdk:"company_uid"`
	CompanyName        types.String                  `tfsdk:"company_name"`
	Locked             types.Bool                    `tfsdk:"locked"`
	AdminLocked        types.Bool                    `tfsdk:"admin_locked"`
	AttributeTags      map[types.String]types.String `tfsdk:"attribute_tags"`
	Cancelable         types.Bool                    `tfsdk:"cancelable"`

	LiveDate          types.String `tfsdk:"live_date"`
	CreateDate        types.String `tfsdk:"create_date"`
	ContractStartDate types.String `tfsdk:"contract_start_date"`
	ContractEndDate   types.String `tfsdk:"contract_end_date"`

	PortUID types.String `tfsdk:"port_uid"`

	AEndConfiguration *vxcEndConfigurationModel `tfsdk:"a_end"`
	BEndConfiguration *vxcEndConfigurationModel `tfsdk:"b_end"`

	AEndOrderConfiguration *vxcOrderEndPointConfigurationModel `tfsdk:"a_end_order_configuration"`
	BEndOrderConfiguration *vxcOrderEndPointConfigurationModel `tfsdk:"b_end_order_configuration"`

	Resources   *vxcResourcesModel `tfsdk:"resources"`
	VXCApproval *vxcApprovalModel  `tfsdk:"vxc_approval"`
}

// vxcResourcesModel represents the resources associated with a VXC.
type vxcResourcesModel struct {
	Interface     []*portInterfaceModel `tfsdk:"interface"`
	VirtualRouter *virtualRouterModel   `tfsdk:"virtual_router"`
	VLL           *vllConfigModel       `tfsdk:"vll"`
	// TODO - CSPConnection struct
}

// virtualRouterModel maps the virtual router schema data.
type virtualRouterModel struct {
	MCRAsn             types.Int64  `tfsdk:"mcr_asn"`
	ResourceName       types.String `tfsdk:"resource_name"`
	ResourceType       types.String `tfsdk:"resource_type"`
	Speed              types.Int64  `tfsdk:"speed"`
	BGPShutdownDefault types.Bool   `tfsdk:"bgp_shutdown_default"`
}

// vllConfigModel maps the VLL configuration schema data.
type vllConfigModel struct {
	AEndVLAN      types.Int64  `tfsdk:"a_vlan"`
	BEndVLAN      types.Int64  `tfsdk:"b_vlan"`
	Description   types.String `tfsdk:"description"`
	ID            types.Int64  `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	RateLimitMBPS types.Int64  `tfsdk:"rate_limit_mbps"`
	ResourceName  types.String `tfsdk:"resource_name"`
	ResourceType  types.String `tfsdk:"resource_type"`
}

// vxcApprovalModel maps the approval schema data.
type vxcApprovalModel struct {
	Status   types.String `tfsdk:"status"`
	Message  types.String `tfsdk:"message"`
	UID      types.String `tfsdk:"uid"`
	Type     types.String `tfsdk:"type"`
	NewSpeed types.Int64  `tfsdk:"new_speed"`
}

// vxcOrderEndPointConfigurationModel maps the end point configuration schema data.
type vxcOrderEndPointConfigurationModel struct {
	ProductUID              types.String             `tfsdk:"product_uid,omitempty"`
	VLAN                    types.Int64              `tfsdk:"vlan,omitempty"`
	PartnerConfig           *vxcPartnerConfiguration `tfsdk:"partner_config,omitempty"`
	*vxcOrderMVEConfigModel `tfsdk:"mve_config,omitempty"`
}

// vxcOrderMVEConfigModel maps the MVE configuration schema data.
type vxcOrderMVEConfigModel struct {
	InnerVLAN             types.Int64 `tfsdk:"inner_vlan,omitempty"`
	NetworkInterfaceIndex types.Int64 `tfsdk:"vnic_index"`
}

// vxcPartnerConfiguration is an interface to ensure the partner configuration is implemented.
type vxcPartnerConfiguration interface {
	IsParnerConfiguration()
}

// vxcPartnerConfigAWSModel maps the partner configuration schema data for AWS.
type vxcPartnerConfigAWSModel struct {
	vxcPartnerConfiguration
	ConnectType       types.String `tfsdk:"connect_type"`
	Type              types.String `tfsdk:"type"`
	OwnerAccount      types.String `tfsdk:"owner_account"`
	ASN               types.Int64  `tfsdk:"asn,omitempty"`
	AmazonASN         types.Int64  `tfsdk:"amazon_asn,omitempty"`
	AuthKey           types.String `tfsdk:"auth_key,omitempty"`
	Prefixes          types.String `tfsdk:"prefixes,omitempty"`
	CustomerIPAddress types.String `tfsdk:"customer_ip_address,omitempty"`
	AmazonIPAddress   types.String `tfsdk:"amazon_ip_address,omitempty"`
	ConnectionName    types.String `tfsdk:"name,omitempty"`
}

// vxcPartnerConfigAzureModel maps the partner configuration schema data for Azure.
type vxcPartnerConfigAzureModel struct {
	vxcPartnerConfiguration
	ConnectType types.String `tfsdk:"connect_type"`
	ServiceKey  types.String `tfsdk:"service_key"`
}

// vxcPartnerConfigGoogleModel maps the partner configuration schema data for Google.
type vxcPartnerConfigGoogleModel struct {
	vxcPartnerConfiguration
	ConnectType types.String `tfsdk:"connect_type"`
	PairingKey  types.String `tfsdk:"pairing_key"`
}

// vxcPartnerConfigOracleModel maps the partner configuration schema data for Oracle.
type vxcPartnerConfigOracleModel struct {
	vxcPartnerConfiguration
	ConnectType      types.String `tfsdk:"connect_type"`
	VirtualCircuitId types.String `tfsdk:"virtual_circuit_id"`
}

// vxcEndConfigurationModel maps the end configuration schema data.
type vxcEndConfigurationModel struct {
	OwnerUID              types.String `tfsdk:"owner_uid"`
	UID                   types.String `tfsdk:"product_uid"`
	Name                  types.String `tfsdk:"product_name"`
	LocationID            types.Int64  `tfsdk:"location_id"`
	Location              types.String `tfsdk:"location"`
	VLAN                  int          `tfsdk:"vlan"`
	InnerVLAN             types.Int64  `tfsdk:"inner_vlan"`
	NetworkInterfaceIndex types.Int64  `tfsdk:"vnic_index"`
	SecondaryName         types.String `tfsdk:"secondary_name"`
}

func (orm *vxcResourceModel) fromAPIVXC(v *megaport.VXC) {
	orm.UID = types.StringValue(v.UID)
	orm.ID = types.Int64Value(int64(v.ID))
	orm.Name = types.StringValue(v.Name)
	orm.ServiceID = types.Int64Value(int64(v.ServiceID))
	orm.Type = types.StringValue(v.Type)
	orm.RateLimit = types.Int64Value(int64(v.RateLimit))
	orm.DistanceBand = types.StringValue(v.DistanceBand)
	orm.ProvisioningStatus = types.StringValue(v.ProvisioningStatus)
	orm.SecondaryName = types.StringValue(v.SecondaryName)
	orm.UsageAlgorithm = types.StringValue(v.UsageAlgorithm)
	orm.CreatedBy = types.StringValue(v.CreatedBy)
	orm.ContractTermMonths = types.Int64Value(int64(v.ContractTermMonths))
	orm.CompanyUID = types.StringValue(v.CompanyUID)
	orm.CompanyName = types.StringValue(v.CompanyName)
	orm.Locked = types.BoolValue(v.Locked)
	orm.AdminLocked = types.BoolValue(v.AdminLocked)
	orm.Cancelable = types.BoolValue(v.Cancelable)
	orm.LiveDate = types.StringValue(v.LiveDate.Format(time.RFC850))
	orm.CreateDate = types.StringValue(v.CreateDate.Format(time.RFC850))
	orm.ContractStartDate = types.StringValue(v.ContractStartDate.Format(time.RFC850))
	orm.ContractEndDate = types.StringValue(v.ContractEndDate.Format(time.RFC850))

	orm.AEndConfiguration = &vxcEndConfigurationModel{
		OwnerUID:              types.StringValue(v.AEndConfiguration.OwnerUID),
		UID:                   types.StringValue(v.AEndConfiguration.UID),
		Name:                  types.StringValue(v.AEndConfiguration.Name),
		LocationID:            types.Int64Value(int64(v.AEndConfiguration.LocationID)),
		Location:              types.StringValue(v.AEndConfiguration.Location),
		VLAN:                  v.AEndConfiguration.VLAN,
		InnerVLAN:             types.Int64Value(int64(v.AEndConfiguration.InnerVLAN)),
		NetworkInterfaceIndex: types.Int64Value(int64(v.AEndConfiguration.NetworkInterfaceIndex)),
		SecondaryName:         types.StringValue(v.AEndConfiguration.SecondaryName),
	}

	orm.BEndConfiguration = &vxcEndConfigurationModel{
		OwnerUID:              types.StringValue(v.BEndConfiguration.OwnerUID),
		UID:                   types.StringValue(v.BEndConfiguration.UID),
		Name:                  types.StringValue(v.BEndConfiguration.Name),
		LocationID:            types.Int64Value(int64(v.BEndConfiguration.LocationID)),
		Location:              types.StringValue(v.BEndConfiguration.Location),
		VLAN:                  v.BEndConfiguration.VLAN,
		InnerVLAN:             types.Int64Value(int64(v.BEndConfiguration.InnerVLAN)),
		NetworkInterfaceIndex: types.Int64Value(int64(v.BEndConfiguration.NetworkInterfaceIndex)),
		SecondaryName:         types.StringValue(v.BEndConfiguration.SecondaryName),
	}

	orm.VXCApproval = &vxcApprovalModel{
		Status:   types.StringValue(v.VXCApproval.Status),
		Message:  types.StringValue(v.VXCApproval.Message),
		UID:      types.StringValue(v.VXCApproval.UID),
		Type:     types.StringValue(v.VXCApproval.Type),
		NewSpeed: types.Int64Value(int64(v.VXCApproval.NewSpeed)),
	}

	resources := &vxcResourcesModel{
		VLL: &vllConfigModel{
			AEndVLAN:      types.Int64Value(int64(v.Resources.VLL.AEndVLAN)),
			BEndVLAN:      types.Int64Value(int64(v.Resources.VLL.BEndVLAN)),
			Description:   types.StringValue(v.Resources.VLL.Description),
			ID:            types.Int64Value(int64(v.Resources.VLL.ID)),
			Name:          types.StringValue(v.Resources.VLL.Name),
			RateLimitMBPS: types.Int64Value(int64(v.Resources.VLL.RateLimitMBPS)),
			ResourceName:  types.StringValue(v.Resources.VLL.ResourceName),
			ResourceType:  types.StringValue(v.Resources.VLL.ResourceType),
		},
		VirtualRouter: &virtualRouterModel{
			MCRAsn:             types.Int64Value(int64(v.Resources.VirtualRouter.MCRAsn)),
			ResourceName:       types.StringValue(v.Resources.VirtualRouter.ResourceName),
			ResourceType:       types.StringValue(v.Resources.VirtualRouter.ResourceType),
			Speed:              types.Int64Value(int64(v.Resources.VirtualRouter.Speed)),
			BGPShutdownDefault: types.BoolValue(v.Resources.VirtualRouter.BGPShutdownDefault),
		},
	}
	for _, i := range v.Resources.Interface {
		resources.Interface = append(resources.Interface, &portInterfaceModel{
			Demarcation:  types.StringValue(i.Demarcation),
			Description:  types.StringValue(i.Description),
			ID:           types.Int64Value(int64(i.ID)),
			LOATemplate:  types.StringValue(i.LOATemplate),
			Media:        types.StringValue(i.Media),
			Name:         types.StringValue(i.Name),
			PortSpeed:    types.Int64Value(int64(i.PortSpeed)),
			ResourceName: types.StringValue(i.ResourceName),
			ResourceType: types.StringValue(i.ResourceType),
			Up:           types.Int64Value(int64(i.Up)),
		})
	}
	// TODO - Struct Model Fields
}

// NewPortResource is a helper function to simplify the provider implementation.
func NewVxcResource() resource.Resource {
	return &vxcResource{}
}

// vxcResource is the resource implementation.
type vxcResource struct {
	client *megaport.Client
}

// Metadata returns the resource type name.
func (r *vxcResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vxc"
}

// Schema defines the schema for the resource.
func (r *vxcResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"last_updated": schema.StringAttribute{
				Description: "The last time the resource was updated.",
				Computed:    true,
			},
			"uid": schema.StringAttribute{
				Description: "The unique identifier for the resource.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"product_id": schema.Int64Attribute{
				Description: "The numeric ID of the product.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"product_name": schema.StringAttribute{
				Description: "The name of the product.",
				Required:    true,
			},

			"provisioning_status": schema.StringAttribute{
				Description: "The provisioning status of the product.",
				Computed:    true,
			},
			"create_date": schema.StringAttribute{
				Description: "The date the product was created.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_by": schema.StringAttribute{
				Description: "The user who created the product.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Create a new resource.
func (r *vxcResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan vxcResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	buyReq := &megaport.BuyVXCRequest{
		PortUID:   plan.PortUID.ValueString(),
		VXCName:   plan.Name.ValueString(),
		RateLimit: int(plan.RateLimit.ValueInt64()),

		WaitForProvision: true,
		WaitForTime:      10 * time.Minute,
	}

	aEnd := megaport.VXCOrderEndpointConfiguration{
		ProductUID: plan.AEndOrderConfiguration.ProductUID.ValueString(),
		VLAN:       int(plan.AEndOrderConfiguration.VLAN.ValueInt64()),
	}
	if plan.AEndOrderConfiguration.vxcOrderMVEConfigModel != nil {
		aEnd.VXCOrderMVEConfig = &megaport.VXCOrderMVEConfig{
			InnerVLAN:             int(plan.AEndOrderConfiguration.InnerVLAN.ValueInt64()),
			NetworkInterfaceIndex: int(plan.AEndOrderConfiguration.NetworkInterfaceIndex.ValueInt64()),
		}
	}
	aEnd.PartnerConfig = toAPIPartnerConfig(*plan.AEndOrderConfiguration.PartnerConfig)

	bEnd := megaport.VXCOrderEndpointConfiguration{
		ProductUID: plan.BEndOrderConfiguration.ProductUID.ValueString(),
		VLAN:       int(plan.BEndOrderConfiguration.VLAN.ValueInt64()),
	}
	if plan.BEndOrderConfiguration.vxcOrderMVEConfigModel != nil {
		bEnd.VXCOrderMVEConfig = &megaport.VXCOrderMVEConfig{
			InnerVLAN:             int(plan.BEndOrderConfiguration.InnerVLAN.ValueInt64()),
			NetworkInterfaceIndex: int(plan.BEndOrderConfiguration.NetworkInterfaceIndex.ValueInt64()),
		}
	}
	bEnd.PartnerConfig = toAPIPartnerConfig(*plan.BEndOrderConfiguration.PartnerConfig)

	createdVXC, err := r.client.VXCService.BuyVXC(ctx, buyReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating VXC",
			"Could not create VXC with name "+plan.Name.ValueString()+": "+err.Error(),
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

	// update the plan with the VXC info
	plan.fromAPIVXC(vxc)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read resource information.
func (r *vxcResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state vxcResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get refreshed vxc value from API
	vxc, err := r.client.VXCService.GetVXC(ctx, state.UID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading VXC",
			"Could not read VXC with ID "+state.UID.ValueString()+": "+err.Error(),
		)
		return
	}

	state.fromAPIVXC(vxc)

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *vxcResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state vxcResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// TODO - Update Logic for Req
	updateReq := &megaport.UpdateVXCRequest{}

	_, err := r.client.VXCService.UpdateVXC(ctx, plan.ID.String(), updateReq)

	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating VXC",
			"Could not update VXC with ID "+state.UID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Get refreshed vxc value from API
	vxc, err := r.client.VXCService.GetVXC(ctx, state.UID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading VXC",
			"Could not read VXC with ID "+state.UID.ValueString()+": "+err.Error(),
		)
		return
	}

	state.fromAPIVXC(vxc)

	// Set refreshed state
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *vxcResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state vxcResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete existing order
	err := r.client.VXCService.DeleteVXC(ctx, state.UID.String(), &megaport.DeleteVXCRequest{
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

// Configure adds the provider configured client to the resource.
func (r *vxcResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *vxcResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Retrieve import ID and save to id attribute
	resource.ImportStatePassthroughID(ctx, path.Root("uid"), req, resp)
}

func toAPIPartnerConfig(c vxcPartnerConfiguration) megaport.VXCPartnerConfiguration {
	switch p := c.(type) {
	case *vxcPartnerConfigAWSModel:
		return &megaport.VXCPartnerConfigAWS{
			ConnectType:       p.ConnectType.ValueString(),
			Type:              p.Type.ValueString(),
			OwnerAccount:      p.OwnerAccount.ValueString(),
			ASN:               int(p.ASN.ValueInt64()),
			AmazonASN:         int(p.AmazonASN.ValueInt64()),
			AuthKey:           p.AuthKey.ValueString(),
			Prefixes:          p.Prefixes.ValueString(),
			CustomerIPAddress: p.CustomerIPAddress.ValueString(),
			AmazonIPAddress:   p.AmazonIPAddress.ValueString(),
			ConnectionName:    p.ConnectionName.ValueString(),
		}
	case *vxcPartnerConfigAzureModel:
		return &megaport.VXCPartnerConfigAzure{
			ConnectType: p.ConnectType.ValueString(),
			ServiceKey:  p.ServiceKey.ValueString(),
		}
	case *vxcPartnerConfigGoogleModel:
		return &megaport.VXCPartnerConfigGoogle{
			ConnectType: p.ConnectType.ValueString(),
			PairingKey:  p.PairingKey.ValueString(),
		}
	case *vxcPartnerConfigOracleModel:
		return &megaport.VXCPartnerConfigOracle{
			ConnectType:      p.ConnectType.ValueString(),
			VirtualCircuitId: p.VirtualCircuitId.ValueString(),
		}
	}
	return nil
}
