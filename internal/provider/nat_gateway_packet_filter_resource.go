package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

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
	_ resource.Resource                = &natGatewayPacketFilterResource{}
	_ resource.ResourceWithConfigure   = &natGatewayPacketFilterResource{}
	_ resource.ResourceWithImportState = &natGatewayPacketFilterResource{}
)

// NewNATGatewayPacketFilterResource returns a new packet filter resource.
func NewNATGatewayPacketFilterResource() resource.Resource {
	return &natGatewayPacketFilterResource{}
}

type natGatewayPacketFilterResource struct {
	client *megaport.Client
}

// natGatewayPacketFilterResourceModel maps the resource schema data.
type natGatewayPacketFilterResourceModel struct {
	ID                  types.Int64  `tfsdk:"id"`
	NATGatewayProductID types.String `tfsdk:"nat_gateway_product_uid"`
	Description         types.String `tfsdk:"description"`
	Entries             types.List   `tfsdk:"entries"`
}

// natGatewayPacketFilterEntryModel maps a single entry.
type natGatewayPacketFilterEntryModel struct {
	Action             types.String `tfsdk:"action"`
	Description        types.String `tfsdk:"description"`
	SourceAddress      types.String `tfsdk:"source_address"`
	DestinationAddress types.String `tfsdk:"destination_address"`
	SourcePorts        types.String `tfsdk:"source_ports"`
	DestinationPorts   types.String `tfsdk:"destination_ports"`
	IPProtocol         types.Int64  `tfsdk:"ip_protocol"`
}

var natGatewayPacketFilterEntryAttrs = map[string]attr.Type{
	"action":              types.StringType,
	"description":         types.StringType,
	"source_address":      types.StringType,
	"destination_address": types.StringType,
	"source_ports":        types.StringType,
	"destination_ports":   types.StringType,
	"ip_protocol":         types.Int64Type,
}

func (r *natGatewayPacketFilterResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_nat_gateway_packet_filter"
}

func (r *natGatewayPacketFilterResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "NAT Gateway Packet Filter Resource for the Megaport Terraform Provider. " +
			"Manages an ACL-style packet filter on a NAT Gateway. The filter ID can be referenced from a VXC's vrouter interface via packet_filter_in / packet_filter_out.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "Numeric ID of the packet filter, assigned by the API.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"nat_gateway_product_uid": schema.StringAttribute{
				Description: "Product UID of the NAT Gateway that owns this packet filter.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Description: "Description of the packet filter.",
				Required:    true,
			},
			"entries": schema.ListNestedAttribute{
				Description: "Ordered list of filter entries. Entries are evaluated top-to-bottom; the first matching entry determines the action taken on the packet. At least one entry is required.",
				Required:    true,
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"action": schema.StringAttribute{
							Description: "Action to take on a matching packet. One of `permit` or `deny`.",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.OneOf(megaport.PacketFilterActionPermit, megaport.PacketFilterActionDeny),
							},
						},
						"description": schema.StringAttribute{
							Description: "Optional description for this entry.",
							Optional:    true,
						},
						"source_address": schema.StringAttribute{
							Description: "Source IP address or CIDR (e.g. `10.0.0.0/24` or `0.0.0.0/0`).",
							Required:    true,
						},
						"destination_address": schema.StringAttribute{
							Description: "Destination IP address or CIDR.",
							Required:    true,
						},
						"source_ports": schema.StringAttribute{
							Description: "Source ports — single port or comma-separated list (e.g. `80,443`). Optional.",
							Optional:    true,
						},
						"destination_ports": schema.StringAttribute{
							Description: "Destination ports — single port or comma-separated list. Optional.",
							Optional:    true,
						},
						"ip_protocol": schema.Int64Attribute{
							Description: "IANA IP protocol number (e.g. 6 = TCP, 17 = UDP). Omit or set to 0 for any protocol.",
							Optional:    true,
							Validators: []validator.Int64{
								int64validator.Between(0, 255),
							},
						},
					},
				},
			},
		},
	}
}

func (r *natGatewayPacketFilterResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *natGatewayPacketFilterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan natGatewayPacketFilterResourceModel
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
	created, err := r.client.NATGatewayService.CreateNATGatewayPacketFilter(ctx, productUID, apiReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating NAT Gateway packet filter",
			fmt.Sprintf("Could not create packet filter on NAT Gateway %s: %s", productUID, err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(plan.fromAPI(ctx, created)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *natGatewayPacketFilterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state natGatewayPacketFilterResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	productUID := state.NATGatewayProductID.ValueString()
	id := int(state.ID.ValueInt64())

	pf, err := r.client.NATGatewayService.GetNATGatewayPacketFilter(ctx, productUID, id)
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
			"Error reading NAT Gateway packet filter",
			fmt.Sprintf("Could not read packet filter %d on NAT Gateway %s: %s", id, productUID, err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(state.fromAPI(ctx, pf)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *natGatewayPacketFilterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state natGatewayPacketFilterResourceModel
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
	updated, err := r.client.NATGatewayService.UpdateNATGatewayPacketFilter(ctx, productUID, id, apiReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating NAT Gateway packet filter",
			fmt.Sprintf("Could not update packet filter %d on NAT Gateway %s: %s", id, productUID, err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(plan.fromAPI(ctx, updated)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *natGatewayPacketFilterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state natGatewayPacketFilterResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	productUID := state.NATGatewayProductID.ValueString()
	id := int(state.ID.ValueInt64())

	// A VXC delete upstream in the same apply only detaches packet-filter
	// references asynchronously. Between Terraform destroying the VXC and
	// destroying this packet filter, the server can still report the filter
	// as "associated with an interface" and return 409. Retry briefly so the
	// teardown is robust; the VXC removal eventually clears the binding.
	const (
		retries = 12
		delay   = 5 * time.Second
	)
	var lastErr error
	for attempt := 0; attempt < retries; attempt++ {
		err := r.client.NATGatewayService.DeleteNATGatewayPacketFilter(ctx, productUID, id)
		if err == nil {
			return
		}
		lastErr = err
		var apiErr *megaport.ErrorResponse
		if !errors.As(err, &apiErr) || apiErr.Response == nil {
			break
		}
		if apiErr.Response.StatusCode == http.StatusNotFound {
			return
		}
		if apiErr.Response.StatusCode != http.StatusConflict {
			break
		}
		select {
		case <-ctx.Done():
			lastErr = ctx.Err()
			attempt = retries
		case <-time.After(delay):
		}
	}
	resp.Diagnostics.AddError(
		"Error deleting NAT Gateway packet filter",
		fmt.Sprintf("Could not delete packet filter %d on NAT Gateway %s: %s", id, productUID, lastErr.Error()),
	)
}

// ImportState parses an import ID of the form `<nat_gateway_uid>:<packet_filter_id>`.
func (r *natGatewayPacketFilterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	natUID, pfID, err := parsePackedImportID(req.ID, "nat_gateway_uid", "packet_filter_id")
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("nat_gateway_product_uid"), natUID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), pfID)...)
}

// toAPIRequest builds the SDK create/update payload from the plan.
func (m *natGatewayPacketFilterResourceModel) toAPIRequest(ctx context.Context) (*megaport.NATGatewayPacketFilterRequest, diag.Diagnostics) {
	var diags diag.Diagnostics
	out := &megaport.NATGatewayPacketFilterRequest{
		Description: m.Description.ValueString(),
	}
	if m.Entries.IsNull() || m.Entries.IsUnknown() {
		return out, diags
	}
	entries := []*natGatewayPacketFilterEntryModel{}
	diags.Append(m.Entries.ElementsAs(ctx, &entries, false)...)
	if diags.HasError() {
		return nil, diags
	}
	for _, e := range entries {
		out.Entries = append(out.Entries, megaport.NATGatewayPacketFilterEntry{
			Action:             e.Action.ValueString(),
			Description:        e.Description.ValueString(),
			SourceAddress:      e.SourceAddress.ValueString(),
			DestinationAddress: e.DestinationAddress.ValueString(),
			SourcePorts:        e.SourcePorts.ValueString(),
			DestinationPorts:   e.DestinationPorts.ValueString(),
			IPProtocol:         int(e.IPProtocol.ValueInt64()),
		})
	}
	return out, diags
}

// fromAPI maps the SDK response back into the model.
func (m *natGatewayPacketFilterResourceModel) fromAPI(ctx context.Context, pf *megaport.NATGatewayPacketFilter) diag.Diagnostics {
	var diags diag.Diagnostics
	m.ID = types.Int64Value(int64(pf.ID))
	m.Description = types.StringValue(pf.Description)

	entryObjs := make([]types.Object, 0, len(pf.Entries))
	for _, e := range pf.Entries {
		em := &natGatewayPacketFilterEntryModel{
			Action:             types.StringValue(e.Action),
			Description:        stringOrNull(e.Description),
			SourceAddress:      types.StringValue(e.SourceAddress),
			DestinationAddress: types.StringValue(e.DestinationAddress),
			SourcePorts:        stringOrNull(e.SourcePorts),
			DestinationPorts:   stringOrNull(e.DestinationPorts),
			IPProtocol:         int64OrNull(e.IPProtocol),
		}
		obj, d := types.ObjectValueFrom(ctx, natGatewayPacketFilterEntryAttrs, em)
		diags.Append(d...)
		if !diags.HasError() {
			entryObjs = append(entryObjs, obj)
		}
	}
	list, d := types.ListValueFrom(ctx, types.ObjectType{}.WithAttributeTypes(natGatewayPacketFilterEntryAttrs), entryObjs)
	diags.Append(d...)
	m.Entries = list
	return diags
}

// stringOrNull returns a null types.String for the empty string, mirroring
// Terraform's "Optional with no default" semantics so refresh doesn't drift.
func stringOrNull(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}

// int64OrNull returns a null types.Int64 for the zero value.
func int64OrNull(i int) types.Int64 {
	if i == 0 {
		return types.Int64Null()
	}
	return types.Int64Value(int64(i))
}

// parsePackedImportID parses a colon-separated import ID of the form
// "<uid>:<numeric_id>". The labelUID/labelID arguments are used only in error
// messages so each resource can describe its own ID format.
func parsePackedImportID(importID, labelUID, labelID string) (string, int64, error) {
	colon := strings.IndexByte(importID, ':')
	if colon == -1 {
		return "", 0, fmt.Errorf("invalid import ID format, expected '%s:%s', got %q", labelUID, labelID, importID)
	}
	uidPart := importID[:colon]
	idPart := importID[colon+1:]
	if uidPart == "" || idPart == "" {
		return "", 0, fmt.Errorf("invalid import ID format, %s and %s cannot be empty", labelUID, labelID)
	}
	id, err := strconv.ParseInt(idPart, 10, 64)
	if err != nil {
		return "", 0, fmt.Errorf("invalid %s %q: %w", labelID, idPart, err)
	}
	if id <= 0 {
		return "", 0, fmt.Errorf("invalid %s %q: must be a positive integer", labelID, idPart)
	}
	return uidPart, id, nil
}
