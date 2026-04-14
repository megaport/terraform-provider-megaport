package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
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
	megaport "github.com/megaport/megaportgo"
)

var (
	// Ensure the implementation satisfies the expected interfaces.
	_ resource.Resource                = &mcrResource{}
	_ resource.ResourceWithConfigure   = &mcrResource{}
	_ resource.ResourceWithImportState = &mcrResource{}
	_ resource.ResourceWithMoveState   = &mcrResource{}
)

// mcrResourceModel maps the resource schema data.
type mcrResourceModel struct {
	UID                   types.String `tfsdk:"product_uid"`
	Name                  types.String `tfsdk:"product_name"`
	CostCentre            types.String `tfsdk:"cost_centre"`
	PortSpeed             types.Int64  `tfsdk:"port_speed"`
	LocationID            types.Int64  `tfsdk:"location_id"`
	MarketplaceVisibility types.Bool   `tfsdk:"marketplace_visibility"`
	CompanyUID            types.String `tfsdk:"company_uid"`
	ContractTermMonths    types.Int64  `tfsdk:"contract_term_months"`
	ASN                   types.Int64  `tfsdk:"asn"`
	DiversityZone         types.String `tfsdk:"diversity_zone"`
	PromoCode             types.String `tfsdk:"promo_code"`

	AttributeTags types.Map `tfsdk:"attribute_tags"`
	ResourceTags  types.Map `tfsdk:"resource_tags"`
}

// fromAPIMCR maps the API MCR response to the resource schema.
func (orm *mcrResourceModel) fromAPIMCR(ctx context.Context, m *megaport.MCR, tags map[string]string) diag.Diagnostics {
	apiDiags := diag.Diagnostics{}

	if m.Resources.VirtualRouter.ASN != 0 {
		orm.ASN = types.Int64Value(int64(m.Resources.VirtualRouter.ASN))
	} else {
		orm.ASN = types.Int64Null()
	}

	orm.UID = types.StringValue(m.UID)
	orm.Name = types.StringValue(m.Name)
	orm.CostCentre = types.StringValue(m.CostCentre)
	orm.PortSpeed = types.Int64Value(int64(m.PortSpeed))
	orm.LocationID = types.Int64Value(int64(m.LocationID))
	orm.MarketplaceVisibility = types.BoolValue(m.MarketplaceVisibility)
	orm.CompanyUID = types.StringValue(m.CompanyUID)
	orm.ContractTermMonths = types.Int64Value(int64(m.ContractTermMonths))
	if m.DiversityZone != "" {
		orm.DiversityZone = types.StringValue(m.DiversityZone)
	} else {
		orm.DiversityZone = types.StringNull()
	}

	if m.AttributeTags != nil {
		attributeTags := make(map[string]attr.Value)
		for k, v := range m.AttributeTags {
			attributeTags[k] = types.StringValue(v)
		}
		attributeTagsValue, tagDiags := types.MapValue(types.StringType, attributeTags)
		apiDiags = append(apiDiags, tagDiags...)
		orm.AttributeTags = attributeTagsValue
	} else {
		orm.AttributeTags = types.MapNull(types.StringType)
	}

	if len(tags) > 0 {
		resourceTags, tagDiags := types.MapValueFrom(ctx, types.StringType, tags)
		apiDiags = append(apiDiags, tagDiags...)
		orm.ResourceTags = resourceTags
	} else {
		orm.ResourceTags = types.MapNull(types.StringType)
	}

	return apiDiags
}

// NewMCRResource is a helper function to simplify the provider implementation.
func NewMCRResource() resource.Resource {
	return &mcrResource{}
}

// mcrResource is the resource implementation.
type mcrResource struct {
	client *megaport.Client
}

// Metadata returns the resource type name.
func (r *mcrResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mcr"
}

// Schema defines the schema for the resource.
func (r *mcrResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Megaport Cloud Router (MCR) Resource for the Megaport Terraform Provider. This can be used to create, modify, and delete Megaport MCRs. The MCR is a managed virtual router service that establishes Layer 3 connectivity on the worldwide Megaport software-defined network (SDN). MCR instances are preconfigured in data centers in key global routing zones. An MCR enables data transfer between multi-cloud or hybrid cloud networks, network service providers, and cloud service providers.",
		Attributes: map[string]schema.Attribute{
			"product_uid": schema.StringAttribute{
				Description: "UID identifier of the product.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"product_name": schema.StringAttribute{
				Description: "Name of the product. Specify a name for the MCR that is easily identifiable as yours, particularly if you plan on provisioning more than one MCR.",
				Required:    true,
			},
			"diversity_zone": schema.StringAttribute{
				Description: "Diversity zone of the product. If the parameter is not provided, a diversity zone will be automatically allocated.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"promo_code": schema.StringAttribute{
				Description: "Promo code is an optional string that can be used to enter a promotional code for the service order. The code is not validated, so if the code doesn't exist or doesn't work for the service, the request will still be successful.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"port_speed": schema.Int64Attribute{
				Description: "Bandwidth speed of the product. The MCR can scale from 1 Gbps to 100 Gbps. The rate limit is an aggregate capacity that determines the speed for all connections through the MCR. MCR bandwidth is shared between all the Cloud Service Provider (CSP) connections added to it. The rate limit is fixed for the life of the service. MCR2 supports seven speeds: 1000, 2500, 5000, 10000, 25000, 50000, and 100000 MBPS",
				Required:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
				Validators: []validator.Int64{
					int64validator.OneOf(1000, 2500, 5000, 10000, 25000, 50000, 100000),
				},
			},
			"location_id": schema.Int64Attribute{
				Description: "The numeric location ID of the product. This value can be retrieved from the data source megaport_location.",
				Required:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"contract_term_months": schema.Int64Attribute{
				Description: "The term of the contract in months: valid values are 1, 12, 24, 36, 48, and 60. To set the product to a month-to-month contract with no minimum term, set the value to 1.",
				Required:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
				Validators: []validator.Int64{
					int64validator.OneOf(1, 12, 24, 36, 48, 60),
				},
			},
			"company_uid": schema.StringAttribute{
				Description: "Megaport Company UID of the product.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cost_centre": schema.StringAttribute{
				Description: "A customer reference number to be included in billing information and invoices. Also known as the service level reference (SLR) number. Specify a unique identifying number for the product to be used for billing purposes, such as a cost center number or a unique customer ID. The service level reference number appears for each service under the Product section of the invoice. You can also edit this field for an existing service. Please note that a VXC associated with the MCR is not automatically updated with the MCR service level reference number.",
				Computed:    true,
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"marketplace_visibility": schema.BoolAttribute{
				Description: "Whether the product is visible in the Marketplace.",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"asn": schema.Int64Attribute{
				Description: "Autonomous System Number (ASN) of the MCR in the MCR order configuration. Defaults to 133937 if not specified. For most configurations, the default ASN is appropriate. The ASN is used for BGP peering sessions on any VXCs connected to this MCR. See the documentation for your cloud providers before overriding the default value. For example, some public cloud services require the use of a public ASN and Microsoft blocks an ASN value of 65515 for Azure connections.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"attribute_tags": schema.MapAttribute{
				ElementType: types.StringType,
				Description: "Attribute tags of the product.",
				Computed:    true,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
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
		},
	}
}

// Create a new resource.
func (r *mcrResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan mcrResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	buyReq := &megaport.BuyMCRRequest{
		Name:             plan.Name.ValueString(),
		Term:             int(plan.ContractTermMonths.ValueInt64()),
		PortSpeed:        int(plan.PortSpeed.ValueInt64()),
		LocationID:       int(plan.LocationID.ValueInt64()),
		CostCentre:       plan.CostCentre.ValueString(),
		PromoCode:        plan.PromoCode.ValueString(),
		WaitForProvision: true,
		WaitForTime:      waitForTime,
	}

	if !plan.ASN.IsNull() {
		buyReq.MCRAsn = int(plan.ASN.ValueInt64())
	}

	if !plan.DiversityZone.IsNull() {
		buyReq.DiversityZone = plan.DiversityZone.ValueString()
	}

	if !plan.ResourceTags.IsNull() {
		tagMap, tagDiags := toResourceTagMap(ctx, plan.ResourceTags)
		resp.Diagnostics.Append(tagDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		buyReq.ResourceTags = tagMap
	}

	err := r.client.MCRService.ValidateMCROrder(ctx, buyReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Validation error while attempting to create MCR",
			fmt.Sprintf("Validation error while attempting to create MCR with name %s: %s", plan.Name.ValueString(), err.Error()),
		)
		return
	}

	createdMCR, err := r.client.MCRService.BuyMCR(ctx, buyReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating MCR",
			fmt.Sprintf("Could not create MCR with name %s: %s", plan.Name.ValueString(), err.Error()),
		)
		return
	}

	createdID := createdMCR.TechnicalServiceUID

	// get the created MCR
	mcr, err := r.client.MCRService.GetMCR(ctx, createdID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading newly created MCR",
			fmt.Sprintf("Could not read newly created MCR with ID %s: %s", createdID, err.Error()),
		)
		return
	}

	tags, err := r.fetchResourceTags(ctx, createdID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading resource tags",
			fmt.Sprintf("Could not read resource tags for MCR with ID %s: %s", createdID, err.Error()),
		)
		return
	}

	// update the plan with the MCR info
	apiDiags := plan.fromAPIMCR(ctx, mcr, tags)
	resp.Diagnostics.Append(apiDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read resource information.
func (r *mcrResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state mcrResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get refreshed mcr value from API
	mcr, err := r.client.MCRService.GetMCR(ctx, state.UID.ValueString())
	if err != nil {
		// MCR has been deleted or is not found
		if mpErr, ok := err.(*megaport.ErrorResponse); ok && mpErr.Response != nil {
			if mpErr.Response.StatusCode == http.StatusNotFound ||
				(mpErr.Response.StatusCode == http.StatusBadRequest && strings.Contains(mpErr.Message, "Could not find a service with UID")) {
				resp.State.RemoveResource(ctx)
				return
			}
		}

		resp.Diagnostics.AddError(
			"Error reading MCR",
			fmt.Sprintf("Could not read MCR with ID %s: %s", state.UID.ValueString(), err.Error()),
		)
		return
	}

	// If the MCR has been deleted
	if mcr.ProvisioningStatus == megaport.STATUS_DECOMMISSIONED {
		resp.State.RemoveResource(ctx)
		return
	}

	tags, err := r.fetchResourceTags(ctx, state.UID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading resource tags",
			fmt.Sprintf("Could not read resource tags for MCR with ID %s: %s", state.UID.ValueString(), err.Error()),
		)
		return
	}

	apiDiags := state.fromAPIMCR(ctx, mcr, tags)
	resp.Diagnostics.Append(apiDiags...)

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *mcrResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state mcrResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	name := plan.Name.ValueString()
	costCentre := plan.CostCentre.ValueString()
	marketplaceVisibility := plan.MarketplaceVisibility.ValueBool()

	_, err := r.client.MCRService.ModifyMCR(ctx, &megaport.ModifyMCRRequest{
		MCRID:                 plan.UID.ValueString(),
		Name:                  name,
		MarketplaceVisibility: &marketplaceVisibility,
		CostCentre:            costCentre,
		WaitForUpdate:         true,
		WaitForTime:           waitForTime,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating MCR",
			fmt.Sprintf("Could not update MCR, unexpected error: %s", err.Error()),
		)
		return
	}

	// Get refreshed mcr value from API
	mcr, err := r.client.MCRService.GetMCR(ctx, state.UID.ValueString())
	if err != nil {
		if mpErr, ok := err.(*megaport.ErrorResponse); ok && mpErr.Response != nil {
			if mpErr.Response.StatusCode == http.StatusNotFound ||
				(mpErr.Response.StatusCode == http.StatusBadRequest && strings.Contains(mpErr.Message, "Could not find a service with UID")) {
				resp.State.RemoveResource(ctx)
				return
			}
		}
		resp.Diagnostics.AddError(
			"Error reading MCR after update",
			fmt.Sprintf("Could not read MCR with ID %s: %s", state.UID.ValueString(), err.Error()),
		)
		return
	}

	// If change in resource tags from state
	if !plan.ResourceTags.Equal(state.ResourceTags) {
		tagMap, tagDiags := toResourceTagMap(ctx, plan.ResourceTags)
		resp.Diagnostics.Append(tagDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		err := r.client.MCRService.UpdateMCRResourceTags(ctx, plan.UID.ValueString(), tagMap)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating MCR resource tags",
				fmt.Sprintf("Could not update MCR resource tags with ID %s: %s", plan.UID.ValueString(), err.Error()),
			)
			return
		}
	}

	tags, err := r.fetchResourceTags(ctx, state.UID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading resource tags",
			fmt.Sprintf("Could not read resource tags for MCR with ID %s: %s", state.UID.ValueString(), err.Error()),
		)
		return
	}

	// Copy plan-only fields not returned by the API before overwriting with API response.
	state.PromoCode = plan.PromoCode
	apiDiags := state.fromAPIMCR(ctx, mcr, tags)
	resp.Diagnostics.Append(apiDiags...)

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *mcrResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state mcrResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete existing order
	_, err := r.client.MCRService.DeleteMCR(ctx, &megaport.DeleteMCRRequest{
		MCRID:      state.UID.ValueString(),
		DeleteNow:  true,
		SafeDelete: true,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting MCR",
			fmt.Sprintf("Could not delete MCR, unexpected error: %s", err.Error()),
		)
		return
	}
}

// Configure adds the provider configured client to the resource.
func (r *mcrResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	providerData, ok := configureMegaportResource(req, resp)
	if !ok {
		return
	}
	r.client = providerData.client
}

func (r *mcrResource) fetchResourceTags(ctx context.Context, id string) (map[string]string, error) {
	return r.client.MCRService.ListMCRResourceTags(ctx, id)
}

func (r *mcrResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Retrieve import ID and save to id attribute
	resource.ImportStatePassthroughID(ctx, path.Root("product_uid"), req, resp)
}

// MoveState returns state movers that handle migration from V1 to V2 of the MCR resource.
func (r *mcrResource) MoveState(ctx context.Context) []resource.StateMover {
	return []resource.StateMover{
		{
			StateMover: func(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
				if req.SourceProviderAddress != "registry.terraform.io/megaport/megaport" || req.SourceTypeName != "megaport_mcr" {
					return
				}

				if req.SourceRawState == nil {
					resp.Diagnostics.AddError(
						"Missing source state",
						"Cannot migrate MCR state: source raw state is nil.",
					)
					return
				}
				var rawState map[string]json.RawMessage
				if err := json.Unmarshal(req.SourceRawState.JSON, &rawState); err != nil {
					resp.Diagnostics.AddError(
						"Error unmarshalling V1 MCR state",
						fmt.Sprintf("Could not unmarshal V1 MCR raw state: %s", err.Error()),
					)
					return
				}

				var model mcrResourceModel

				// UID (product_uid)
				if v, ok := rawState["product_uid"]; ok {
					var s *string
					if err := json.Unmarshal(v, &s); err == nil && s != nil {
						model.UID = types.StringValue(*s)
					} else {
						model.UID = types.StringNull()
					}
				}

				// Name (product_name)
				if v, ok := rawState["product_name"]; ok {
					var s *string
					if err := json.Unmarshal(v, &s); err == nil && s != nil {
						model.Name = types.StringValue(*s)
					} else {
						model.Name = types.StringNull()
					}
				}

				// CostCentre
				if v, ok := rawState["cost_centre"]; ok {
					var s *string
					if err := json.Unmarshal(v, &s); err == nil && s != nil {
						model.CostCentre = types.StringValue(*s)
					} else {
						model.CostCentre = types.StringNull()
					}
				}

				// PortSpeed
				if v, ok := rawState["port_speed"]; ok {
					var n *float64
					if err := json.Unmarshal(v, &n); err == nil && n != nil {
						model.PortSpeed = types.Int64Value(int64(*n))
					} else {
						model.PortSpeed = types.Int64Null()
					}
				}

				// LocationID
				if v, ok := rawState["location_id"]; ok {
					var n *float64
					if err := json.Unmarshal(v, &n); err == nil && n != nil {
						model.LocationID = types.Int64Value(int64(*n))
					} else {
						model.LocationID = types.Int64Null()
					}
				}

				// MarketplaceVisibility
				if v, ok := rawState["marketplace_visibility"]; ok {
					var b *bool
					if err := json.Unmarshal(v, &b); err == nil && b != nil {
						model.MarketplaceVisibility = types.BoolValue(*b)
					} else {
						model.MarketplaceVisibility = types.BoolNull()
					}
				}

				// CompanyUID
				if v, ok := rawState["company_uid"]; ok {
					var s *string
					if err := json.Unmarshal(v, &s); err == nil && s != nil {
						model.CompanyUID = types.StringValue(*s)
					} else {
						model.CompanyUID = types.StringNull()
					}
				}

				// ContractTermMonths
				if v, ok := rawState["contract_term_months"]; ok {
					var n *float64
					if err := json.Unmarshal(v, &n); err == nil && n != nil {
						model.ContractTermMonths = types.Int64Value(int64(*n))
					} else {
						model.ContractTermMonths = types.Int64Null()
					}
				}

				// ASN
				if v, ok := rawState["asn"]; ok {
					var n *float64
					if err := json.Unmarshal(v, &n); err == nil && n != nil {
						model.ASN = types.Int64Value(int64(*n))
					} else {
						model.ASN = types.Int64Null()
					}
				} else {
					model.ASN = types.Int64Null()
				}

				// DiversityZone
				if v, ok := rawState["diversity_zone"]; ok {
					var s *string
					if err := json.Unmarshal(v, &s); err == nil && s != nil {
						model.DiversityZone = types.StringValue(*s)
					} else {
						model.DiversityZone = types.StringNull()
					}
				}

				// PromoCode
				if v, ok := rawState["promo_code"]; ok {
					var s *string
					if err := json.Unmarshal(v, &s); err == nil && s != nil {
						model.PromoCode = types.StringValue(*s)
					} else {
						model.PromoCode = types.StringNull()
					}
				}

				// AttributeTags
				if v, ok := rawState["attribute_tags"]; ok {
					var tags *map[string]string
					if err := json.Unmarshal(v, &tags); err == nil && tags != nil {
						tagValues := make(map[string]attr.Value, len(*tags))
						for k, val := range *tags {
							tagValues[k] = types.StringValue(val)
						}
						mapVal, mapDiags := types.MapValue(types.StringType, tagValues)
						resp.Diagnostics.Append(mapDiags...)
						model.AttributeTags = mapVal
					} else {
						model.AttributeTags = types.MapNull(types.StringType)
					}
				} else {
					model.AttributeTags = types.MapNull(types.StringType)
				}

				// ResourceTags
				if v, ok := rawState["resource_tags"]; ok {
					var tags *map[string]string
					if err := json.Unmarshal(v, &tags); err == nil && tags != nil {
						tagValues := make(map[string]attr.Value, len(*tags))
						for k, val := range *tags {
							tagValues[k] = types.StringValue(val)
						}
						mapVal, mapDiags := types.MapValue(types.StringType, tagValues)
						resp.Diagnostics.Append(mapDiags...)
						model.ResourceTags = mapVal
					} else {
						model.ResourceTags = types.MapNull(types.StringType)
					}
				} else {
					model.ResourceTags = types.MapNull(types.StringType)
				}

				if resp.Diagnostics.HasError() {
					return
				}

				resp.Diagnostics.Append(resp.TargetState.Set(ctx, &model)...)
			},
		},
	}
}
