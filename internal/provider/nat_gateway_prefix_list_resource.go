package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	megaport "github.com/megaport/megaportgo"
)

var (
	_ resource.Resource                = &natGatewayPrefixListResource{}
	_ resource.ResourceWithConfigure   = &natGatewayPrefixListResource{}
	_ resource.ResourceWithImportState = &natGatewayPrefixListResource{}
)

// NewNATGatewayPrefixListResource returns a new prefix list resource.
func NewNATGatewayPrefixListResource() resource.Resource {
	return &natGatewayPrefixListResource{}
}

type natGatewayPrefixListResource struct {
	client *megaport.Client
}

type natGatewayPrefixListResourceModel struct {
	ID                  types.Int64  `tfsdk:"id"`
	NATGatewayProductID types.String `tfsdk:"nat_gateway_product_uid"`
	Description         types.String `tfsdk:"description"`
	AddressFamily       types.String `tfsdk:"address_family"`
	Entries             types.List   `tfsdk:"entries"`
}

type natGatewayPrefixListEntryModel struct {
	Action types.String `tfsdk:"action"`
	Prefix types.String `tfsdk:"prefix"`
	Ge     types.Int64  `tfsdk:"ge"`
	Le     types.Int64  `tfsdk:"le"`
}

var natGatewayPrefixListEntryAttrs = map[string]attr.Type{
	"action": types.StringType,
	"prefix": types.StringType,
	"ge":     types.Int64Type,
	"le":     types.Int64Type,
}

func (r *natGatewayPrefixListResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_nat_gateway_prefix_list"
}

func (r *natGatewayPrefixListResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "NAT Gateway Prefix List Resource for the Megaport Terraform Provider. Manages a prefix list (route filter) on a NAT Gateway, scoped to either IPv4 or IPv6.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "Numeric ID of the prefix list, assigned by the API.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"nat_gateway_product_uid": schema.StringAttribute{
				Description: "Product UID of the NAT Gateway that owns this prefix list.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Description: "Description of the prefix list.",
				Required:    true,
			},
			"address_family": schema.StringAttribute{
				Description: "Address family of the prefix list. One of `IPv4` or `IPv6`. Changing this forces replacement.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(megaport.AddressFamilyIPv4, megaport.AddressFamilyIPv6),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"entries": schema.ListNestedAttribute{
				Description: "Entries in the prefix list. At least one entry is required. Each entry's prefix must match the address_family.",
				Required:    true,
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"action": schema.StringAttribute{
							Description: "Action for the entry. One of `permit` or `deny`.",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.OneOf(megaport.PrefixListActionPermit, megaport.PrefixListActionDeny),
							},
						},
						"prefix": schema.StringAttribute{
							Description: "CIDR network address of the entry. Must be a valid network address with no host bits set.",
							Required:    true,
							Validators: []validator.String{
								canonicalCIDRValidator{},
							},
						},
						"ge": schema.Int64Attribute{
							Description: "Minimum prefix length to be matched. 0–32 for IPv4, 0–128 for IPv6.",
							Optional:    true,
							Validators: []validator.Int64{
								int64validator.Between(0, 128),
							},
						},
						"le": schema.Int64Attribute{
							Description: "Maximum prefix length to be matched. Must be greater than or equal to `ge`.",
							Optional:    true,
							Validators: []validator.Int64{
								int64validator.Between(0, 128),
							},
						},
					},
				},
			},
		},
	}
}

func (r *natGatewayPrefixListResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *natGatewayPrefixListResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan natGatewayPrefixListResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq, convertDiags := plan.toAPIRequest(ctx)
	resp.Diagnostics.Append(convertDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	productUID := plan.NATGatewayProductID.ValueString()
	created, err := r.client.NATGatewayService.CreateNATGatewayPrefixList(ctx, productUID, apiReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating NAT Gateway prefix list",
			fmt.Sprintf("Could not create prefix list on NAT Gateway %s: %s", productUID, err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(plan.fromAPI(ctx, created)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *natGatewayPrefixListResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state natGatewayPrefixListResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	productUID := state.NATGatewayProductID.ValueString()
	id := int(state.ID.ValueInt64())

	pl, err := r.client.NATGatewayService.GetNATGatewayPrefixList(ctx, productUID, id)
	if err != nil {
		var apiErr *megaport.ErrorResponse
		if errors.As(err, &apiErr) && apiErr.Response != nil {
			if apiErr.Response.StatusCode == http.StatusNotFound ||
				(apiErr.Response.StatusCode == http.StatusBadRequest && strings.Contains(apiErr.Message, "Could not find a service with UID")) {
				resp.State.RemoveResource(ctx)
				return
			}
		}
		resp.Diagnostics.AddError(
			"Error reading NAT Gateway prefix list",
			fmt.Sprintf("Could not read prefix list %d on NAT Gateway %s: %s", id, productUID, err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(state.fromAPI(ctx, pl)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *natGatewayPrefixListResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state natGatewayPrefixListResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq, convertDiags := plan.toAPIRequest(ctx)
	resp.Diagnostics.Append(convertDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	productUID := state.NATGatewayProductID.ValueString()
	id := int(state.ID.ValueInt64())
	updated, err := r.client.NATGatewayService.UpdateNATGatewayPrefixList(ctx, productUID, id, apiReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating NAT Gateway prefix list",
			fmt.Sprintf("Could not update prefix list %d on NAT Gateway %s: %s", id, productUID, err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(plan.fromAPI(ctx, updated)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *natGatewayPrefixListResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state natGatewayPrefixListResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	productUID := state.NATGatewayProductID.ValueString()
	id := int(state.ID.ValueInt64())
	if err := r.client.NATGatewayService.DeleteNATGatewayPrefixList(ctx, productUID, id); err != nil {
		var apiErr *megaport.ErrorResponse
		if errors.As(err, &apiErr) && apiErr.Response != nil && apiErr.Response.StatusCode == http.StatusNotFound {
			return
		}
		resp.Diagnostics.AddError(
			"Error deleting NAT Gateway prefix list",
			fmt.Sprintf("Could not delete prefix list %d on NAT Gateway %s: %s", id, productUID, err.Error()),
		)
		return
	}
}

func (r *natGatewayPrefixListResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	natUID, plID, err := parsePackedImportID(req.ID, "nat_gateway_uid", "prefix_list_id")
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("nat_gateway_product_uid"), natUID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), plID)...)
}

// toAPIRequest builds the SDK request from the plan. The SDK handles the
// int↔string conversion for ge/le.
func (m *natGatewayPrefixListResourceModel) toAPIRequest(ctx context.Context) (*megaport.NATGatewayPrefixList, diag.Diagnostics) {
	var diags diag.Diagnostics
	out := &megaport.NATGatewayPrefixList{
		Description:   m.Description.ValueString(),
		AddressFamily: m.AddressFamily.ValueString(),
	}
	if m.Entries.IsNull() || m.Entries.IsUnknown() {
		return out, diags
	}
	entries := []*natGatewayPrefixListEntryModel{}
	diags.Append(m.Entries.ElementsAs(ctx, &entries, false)...)
	if diags.HasError() {
		return nil, diags
	}
	for _, e := range entries {
		out.Entries = append(out.Entries, megaport.NATGatewayPrefixListEntry{
			Action: e.Action.ValueString(),
			Prefix: e.Prefix.ValueString(),
			Ge:     int(e.Ge.ValueInt64()),
			Le:     int(e.Le.ValueInt64()),
		})
	}
	return out, diags
}

func (m *natGatewayPrefixListResourceModel) fromAPI(ctx context.Context, pl *megaport.NATGatewayPrefixList) diag.Diagnostics {
	var diags diag.Diagnostics
	m.ID = types.Int64Value(int64(pl.ID))
	m.Description = types.StringValue(pl.Description)
	m.AddressFamily = types.StringValue(pl.AddressFamily)

	entryObjs := make([]types.Object, 0, len(pl.Entries))
	for _, e := range pl.Entries {
		em := &natGatewayPrefixListEntryModel{
			Action: types.StringValue(e.Action),
			Prefix: types.StringValue(e.Prefix),
			Ge:     int64OrNull(e.Ge),
			Le:     int64OrNull(e.Le),
		}
		obj, d := types.ObjectValueFrom(ctx, natGatewayPrefixListEntryAttrs, em)
		diags.Append(d...)
		if !diags.HasError() {
			entryObjs = append(entryObjs, obj)
		}
	}
	list, d := types.ListValueFrom(ctx, types.ObjectType{}.WithAttributeTypes(natGatewayPrefixListEntryAttrs), entryObjs)
	diags.Append(d...)
	m.Entries = list
	return diags
}
