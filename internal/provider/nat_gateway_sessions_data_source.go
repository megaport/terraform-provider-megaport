package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	megaport "github.com/megaport/megaportgo"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &natGatewaySessionsDataSource{}
	_ datasource.DataSourceWithConfigure = &natGatewaySessionsDataSource{}
)

// natGatewaySessionsDataSource is the data source implementation.
type natGatewaySessionsDataSource struct {
	client *megaport.Client
}

// natGatewaySessionsDataSourceModel maps the data source schema data.
type natGatewaySessionsDataSourceModel struct {
	Sessions []natGatewaySessionEntryModel `tfsdk:"sessions"`
}

// natGatewaySessionEntryModel maps a single speed / session-count entry.
type natGatewaySessionEntryModel struct {
	SpeedMbps    types.Int64 `tfsdk:"speed_mbps"`
	SessionCount types.List  `tfsdk:"session_count"`
}

// NewNATGatewaySessionsDataSource is a helper function to simplify the provider implementation.
func NewNATGatewaySessionsDataSource() datasource.DataSource {
	return &natGatewaySessionsDataSource{}
}

// Metadata returns the data source type name.
func (d *natGatewaySessionsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_nat_gateway_sessions"
}

// Schema defines the schema for the data source.
func (d *natGatewaySessionsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Exposes the NAT Gateway speed and session-count availability matrix so HCL can reference valid combinations explicitly. Useful in for_each, validation, or locals to discover supported speed/session_count pairs at plan time instead of relying on apply-time API errors.",
		Attributes: map[string]schema.Attribute{
			"sessions": schema.ListNestedAttribute{
				Description: "All supported NAT Gateway speed / session-count pairings.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"speed_mbps": schema.Int64Attribute{
							Description: "Supported NAT Gateway speed in Mbps.",
							Computed:    true,
						},
						"session_count": schema.ListAttribute{
							Description: "Session counts permitted at this speed.",
							Computed:    true,
							ElementType: types.Int64Type,
						},
					},
				},
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *natGatewaySessionsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	sessions, err := d.client.NATGatewayService.ListNATGatewaySessions(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to list NAT Gateway sessions",
			err.Error(),
		)
		return
	}

	state := natGatewaySessionsDataSourceModel{
		Sessions: make([]natGatewaySessionEntryModel, 0, len(sessions)),
	}
	for _, s := range sessions {
		if s == nil || len(s.SessionCount) == 0 {
			continue
		}
		counts := make([]int64, 0, len(s.SessionCount))
		for _, c := range s.SessionCount {
			counts = append(counts, int64(c))
		}
		countList, listDiags := types.ListValueFrom(ctx, types.Int64Type, counts)
		resp.Diagnostics.Append(listDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.Sessions = append(state.Sessions, natGatewaySessionEntryModel{
			SpeedMbps:    types.Int64Value(int64(s.SpeedMbps)),
			SessionCount: countList,
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Configure adds the provider configured client to the data source.
func (d *natGatewaySessionsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	d.client = data.client
}
