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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	megaport "github.com/megaport/megaportgo"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &natGatewayResource{}
	_ resource.ResourceWithConfigure   = &natGatewayResource{}
	_ resource.ResourceWithImportState = &natGatewayResource{}
)

// natGatewayResourceModel maps the resource schema data.
type natGatewayResourceModel struct {
	ProductUID            types.String `tfsdk:"product_uid"`
	ProductName           types.String `tfsdk:"product_name"`
	CreateDate            types.String `tfsdk:"create_date"`
	CreatedBy             types.String `tfsdk:"created_by"`
	ContractEndDate       types.String `tfsdk:"contract_end_date"`
	LocationID            types.Int64  `tfsdk:"location_id"`
	Speed                 types.Int64  `tfsdk:"speed"`
	ContractTermMonths    types.Int64  `tfsdk:"contract_term_months"`
	AutoRenewTerm         types.Bool   `tfsdk:"auto_renew_term"`
	Locked                types.Bool   `tfsdk:"locked"`
	AdminLocked           types.Bool   `tfsdk:"admin_locked"`
	ServiceLevelReference types.String `tfsdk:"service_level_reference"`
	PromoCode             types.String `tfsdk:"promo_code"`
	ProvisioningStatus    types.String `tfsdk:"provisioning_status"`
	ResourceTags          types.Map    `tfsdk:"resource_tags"`

	// Config fields (flattened from NATGatewayNetworkConfig)
	DiversityZone      types.String `tfsdk:"diversity_zone"`
	ASN                types.Int64  `tfsdk:"asn"`
	BGPShutdownDefault types.Bool   `tfsdk:"bgp_shutdown_default"`
	SessionCount       types.Int64  `tfsdk:"session_count"`
}

// fromAPINATGateway maps the API NAT Gateway response to the resource schema.
func (m *natGatewayResourceModel) fromAPINATGateway(gw *megaport.NATGateway) diag.Diagnostics {
	m.ProductUID = types.StringValue(gw.ProductUID)
	m.ProductName = types.StringValue(gw.ProductName)
	m.CreateDate = types.StringValue(gw.CreateDate)
	m.CreatedBy = types.StringValue(gw.CreatedBy)
	m.ContractEndDate = types.StringValue(gw.ContractEndDate)
	m.LocationID = types.Int64Value(int64(gw.LocationID))
	m.Speed = types.Int64Value(int64(gw.Speed))
	m.ContractTermMonths = types.Int64Value(int64(gw.Term))
	m.AutoRenewTerm = types.BoolValue(gw.AutoRenewTerm)
	m.Locked = types.BoolValue(gw.Locked)
	m.AdminLocked = types.BoolValue(gw.AdminLocked)
	m.ServiceLevelReference = types.StringValue(gw.ServiceLevelReference)
	m.ProvisioningStatus = types.StringValue(gw.ProvisioningStatus)

	// Config fields
	m.DiversityZone = types.StringValue(gw.Config.DiversityZone)
	m.ASN = types.Int64Value(int64(gw.Config.ASN))
	m.BGPShutdownDefault = types.BoolValue(gw.Config.BGPShutdownDefault)
	m.SessionCount = types.Int64Value(int64(gw.Config.SessionCount))

	// Resource tags
	var diags diag.Diagnostics
	if len(gw.ResourceTags) > 0 {
		tagMap := make(map[string]attr.Value, len(gw.ResourceTags))
		for _, tag := range gw.ResourceTags {
			tagMap[tag.Key] = types.StringValue(tag.Value)
		}
		m.ResourceTags, diags = types.MapValue(types.StringType, tagMap)
	} else {
		m.ResourceTags = types.MapNull(types.StringType)
	}
	return diags
}

// toResourceTagSlice converts a Terraform map of tags to a slice of ResourceTag for the SDK.
func toResourceTagSlice(ctx context.Context, tags types.Map) ([]megaport.ResourceTag, error) {
	if tags.IsNull() || tags.IsUnknown() {
		return nil, nil
	}
	tagMap := map[string]string{}
	diags := tags.ElementsAs(ctx, &tagMap, false)
	if diags.HasError() {
		return nil, fmt.Errorf("error converting resource tags: %v", diags.Errors())
	}
	result := make([]megaport.ResourceTag, 0, len(tagMap))
	for k, v := range tagMap {
		result = append(result, megaport.ResourceTag{Key: k, Value: v})
	}
	return result, nil
}

// NewNATGatewayResource is a helper function to simplify the provider implementation.
func NewNATGatewayResource() resource.Resource {
	return &natGatewayResource{}
}

// natGatewayResource is the resource implementation.
type natGatewayResource struct {
	client *megaport.Client
}

// Metadata returns the resource type name.
func (r *natGatewayResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_nat_gateway"
}

// Schema defines the schema for the resource.
func (r *natGatewayResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "NAT Gateway Resource for the Megaport Terraform Provider. This can be used to create, modify, and delete Megaport NAT Gateways. " +
			"Creating this resource places a NAT Gateway order: the design record is created, validated, and purchased, and the provider waits for the service to reach CONFIGURED/LIVE before returning.",
		Attributes: map[string]schema.Attribute{
			"product_uid": schema.StringAttribute{
				Description: "The unique identifier of the NAT Gateway.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"product_name": schema.StringAttribute{
				Description: "The name of the NAT Gateway.",
				Required:    true,
			},
			"create_date": schema.StringAttribute{
				Description: "The date the NAT Gateway was created.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_by": schema.StringAttribute{
				Description: "The user who created the NAT Gateway.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"contract_end_date": schema.StringAttribute{
				Description: "The end date of the contract for the NAT Gateway.",
				Computed:    true,
			},
			"location_id": schema.Int64Attribute{
				Description: "The numeric location ID of the NAT Gateway. This value can be retrieved from the data source megaport_location.",
				Required:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"speed": schema.Int64Attribute{
				Description: "The speed of the NAT Gateway in Mbps.",
				Required:    true,
			},
			"contract_term_months": schema.Int64Attribute{
				Description: "The contract term for the NAT Gateway in months. Valid values are 1, 12, 24, 36, 48, or 60.",
				Required:    true,
				Validators: []validator.Int64{
					int64validator.OneOf(1, 12, 24, 36, 48, 60),
				},
			},
			"auto_renew_term": schema.BoolAttribute{
				Description: "Whether the NAT Gateway contract will auto-renew.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"locked": schema.BoolAttribute{
				Description: "Whether the NAT Gateway is locked.",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"admin_locked": schema.BoolAttribute{
				Description: "Whether the NAT Gateway is admin locked.",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"service_level_reference": schema.StringAttribute{
				Description: "A service level reference for the NAT Gateway.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"promo_code": schema.StringAttribute{
				Description: "A promotional code for the NAT Gateway order.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"provisioning_status": schema.StringAttribute{
				Description: "The provisioning status of the NAT Gateway (e.g. CONFIGURED, LIVE).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"resource_tags": schema.MapAttribute{
				Description: "Resource tags for the NAT Gateway.",
				Optional:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
				},
			},
			"diversity_zone": schema.StringAttribute{
				Description: "The diversity zone of the NAT Gateway.",
				Required:    true,
			},
			"asn": schema.Int64Attribute{
				Description: "The Autonomous System Number (ASN) for the NAT Gateway.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"bgp_shutdown_default": schema.BoolAttribute{
				Description: "Whether BGP sessions are shut down by default on the NAT Gateway.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"session_count": schema.Int64Attribute{
				Description: "The NAT session count for the gateway. Must be a valid pairing with speed — see the nat-gateway session matrix for allowed combinations.",
				Required:    true,
			},
		},
	}
}

// Create a new resource.
func (r *natGatewayResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan natGatewayResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &megaport.CreateNATGatewayRequest{
		ProductName: plan.ProductName.ValueString(),
		LocationID:  int(plan.LocationID.ValueInt64()),
		Speed:       int(plan.Speed.ValueInt64()),
		Term:        int(plan.ContractTermMonths.ValueInt64()),
	}

	if !plan.AutoRenewTerm.IsNull() && !plan.AutoRenewTerm.IsUnknown() {
		createReq.AutoRenewTerm = plan.AutoRenewTerm.ValueBool()
	}

	if !plan.PromoCode.IsNull() && !plan.PromoCode.IsUnknown() {
		createReq.PromoCode = plan.PromoCode.ValueString()
	}

	// Config fields
	config := megaport.NATGatewayNetworkConfig{
		DiversityZone: plan.DiversityZone.ValueString(),
		SessionCount:  int(plan.SessionCount.ValueInt64()),
	}
	if !plan.ASN.IsNull() && !plan.ASN.IsUnknown() {
		config.ASN = int(plan.ASN.ValueInt64())
	}
	if !plan.BGPShutdownDefault.IsNull() && !plan.BGPShutdownDefault.IsUnknown() {
		config.BGPShutdownDefault = plan.BGPShutdownDefault.ValueBool()
	}
	createReq.Config = config

	// Resource tags
	if !plan.ResourceTags.IsNull() {
		tags, err := toResourceTagSlice(ctx, plan.ResourceTags)
		if err != nil {
			resp.Diagnostics.AddError("Error converting resource tags", err.Error())
			return
		}
		createReq.ResourceTags = tags
	}

	tflog.Debug(ctx, "Creating NAT Gateway", map[string]interface{}{
		"product_name": plan.ProductName.ValueString(),
		"location_id":  plan.LocationID.ValueInt64(),
		"speed":        plan.Speed.ValueInt64(),
	})

	createdGW, err := r.client.NATGatewayService.CreateNATGateway(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating NAT Gateway",
			"Could not create NAT Gateway with name "+plan.ProductName.ValueString()+": "+err.Error(),
		)
		return
	}

	createdUID := createdGW.ProductUID

	// The CreateNATGateway call leaves the gateway in DESIGN status. Validate
	// the order (surfaces pricing/config issues before charging the account),
	// then submit the purchase via /v3/networkdesign/buy, and wait for the
	// service to reach CONFIGURED/LIVE.
	if _, err := r.client.NATGatewayService.ValidateNATGatewayOrder(ctx, createdUID); err != nil {
		resp.Diagnostics.AddError(
			"Error validating NAT Gateway order",
			"Could not validate NAT Gateway order for "+createdUID+": "+err.Error(),
		)
		return
	}

	if _, err := r.client.NATGatewayService.BuyNATGateway(ctx, createdUID); err != nil {
		resp.Diagnostics.AddError(
			"Error purchasing NAT Gateway",
			"Could not purchase NAT Gateway order for "+createdUID+": "+err.Error(),
		)
		return
	}

	gw, err := r.waitForNATGatewayProvisioned(ctx, createdUID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error waiting for NAT Gateway provisioning",
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(plan.fromAPINATGateway(gw)...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// waitForNATGatewayProvisioned polls the NAT Gateway until it reaches
// CONFIGURED/LIVE, or returns an error on timeout, terminal state, or
// context cancellation.
func (r *natGatewayResource) waitForNATGatewayProvisioned(ctx context.Context, productUID string) (*megaport.NATGateway, error) {
	const pollInterval = 10 * time.Second

	pollCtx, cancel := context.WithTimeout(ctx, waitForTime)
	defer cancel()

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	var lastStatus string
	for {
		gw, err := r.client.NATGatewayService.GetNATGateway(pollCtx, productUID)
		if err != nil {
			return nil, fmt.Errorf("polling NAT Gateway %s: %w", productUID, err)
		}
		lastStatus = gw.ProvisioningStatus
		for _, ready := range megaport.SERVICE_STATE_READY {
			if lastStatus == ready {
				return gw, nil
			}
		}
		if lastStatus == megaport.STATUS_DECOMMISSIONED || lastStatus == megaport.STATUS_CANCELLED {
			return nil, fmt.Errorf("NAT Gateway %s reached terminal state %q before provisioning", productUID, lastStatus)
		}

		select {
		case <-pollCtx.Done():
			return nil, fmt.Errorf("NAT Gateway %s did not reach CONFIGURED/LIVE within %s (last status %q)", productUID, waitForTime, lastStatus)
		case <-ticker.C:
		}
	}
}

// Read resource information.
func (r *natGatewayResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state natGatewayResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	productUID := state.ProductUID.ValueString()

	gw, err := r.client.NATGatewayService.GetNATGateway(ctx, productUID)
	if err != nil {
		if mpErr, ok := err.(*megaport.ErrorResponse); ok {
			if mpErr.Response.StatusCode == http.StatusNotFound ||
				(mpErr.Response.StatusCode == http.StatusBadRequest && strings.Contains(mpErr.Message, "Could not find a service with UID")) {
				resp.State.RemoveResource(ctx)
				return
			}
		}
		resp.Diagnostics.AddError(
			"Error Reading NAT Gateway",
			"Could not read NAT Gateway with ID "+productUID+": "+err.Error(),
		)
		return
	}

	if gw.ProvisioningStatus == megaport.STATUS_DECOMMISSIONED || gw.ProvisioningStatus == megaport.STATUS_CANCELLED {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(state.fromAPINATGateway(gw)...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update the resource.
func (r *natGatewayResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state natGatewayResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	productUID := state.ProductUID.ValueString()

	// Build update request with all fields (full PUT)
	updateReq := &megaport.UpdateNATGatewayRequest{
		ProductUID:  productUID,
		ProductName: plan.ProductName.ValueString(),
		LocationID:  int(plan.LocationID.ValueInt64()),
		Speed:       int(plan.Speed.ValueInt64()),
		Term:        int(plan.ContractTermMonths.ValueInt64()),
	}

	if !plan.AutoRenewTerm.IsNull() && !plan.AutoRenewTerm.IsUnknown() {
		updateReq.AutoRenewTerm = plan.AutoRenewTerm.ValueBool()
	}

	if !plan.PromoCode.IsNull() && !plan.PromoCode.IsUnknown() {
		updateReq.PromoCode = plan.PromoCode.ValueString()
	}

	// Config fields
	config := megaport.NATGatewayNetworkConfig{
		DiversityZone: plan.DiversityZone.ValueString(),
		SessionCount:  int(plan.SessionCount.ValueInt64()),
	}
	if !plan.ASN.IsNull() && !plan.ASN.IsUnknown() {
		config.ASN = int(plan.ASN.ValueInt64())
	}
	if !plan.BGPShutdownDefault.IsNull() && !plan.BGPShutdownDefault.IsUnknown() {
		config.BGPShutdownDefault = plan.BGPShutdownDefault.ValueBool()
	}
	updateReq.Config = config

	// Resource tags — null means the user removed all tags, so send an empty
	// slice to clear them on the API side (omitting the field would leave stale tags).
	if plan.ResourceTags.IsNull() {
		updateReq.ResourceTags = []megaport.ResourceTag{}
	} else {
		tags, err := toResourceTagSlice(ctx, plan.ResourceTags)
		if err != nil {
			resp.Diagnostics.AddError("Error converting resource tags", err.Error())
			return
		}
		updateReq.ResourceTags = tags
	}

	_, err := r.client.NATGatewayService.UpdateNATGateway(ctx, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating NAT Gateway",
			"Could not update NAT Gateway with ID "+productUID+": "+err.Error(),
		)
		return
	}

	// Re-read from API
	gw, err := r.client.NATGatewayService.GetNATGateway(ctx, productUID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading NAT Gateway",
			"Could not read NAT Gateway with ID "+productUID+": "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(plan.fromAPINATGateway(gw)...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags := resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete the resource.
func (r *natGatewayResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state natGatewayResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	productUID := state.ProductUID.ValueString()

	// NATGatewayService.DeleteNATGateway inspects ProvisioningStatus and
	// routes DESIGN-state records to the design-only DELETE endpoint and
	// provisioned records through CANCEL_NOW. The Terraform resource
	// provisions the gateway on Create, so Delete reaches the CANCEL_NOW
	// branch — but relying on the SDK routing keeps the provider correct
	// if a create fails mid-way and leaves the record in DESIGN.
	if err := r.client.NATGatewayService.DeleteNATGateway(ctx, productUID); err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting NAT Gateway",
			"Could not delete NAT Gateway with ID "+productUID+": "+err.Error(),
		)
		return
	}
}

// Configure adds the provider configured client to the resource.
func (r *natGatewayResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	data, ok := req.ProviderData.(*megaportProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Provider Data Type",
			fmt.Sprintf("Expected *megaportProviderData, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = data.client
}

func (r *natGatewayResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("product_uid"), req, resp)
}
