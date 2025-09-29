package provider

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	megaport "github.com/megaport/megaportgo"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &cloudPortLookupDataSource{}
	_ datasource.DataSourceWithConfigure = &cloudPortLookupDataSource{}
)

// cloudPortLookupDataSource is the data source implementation.
type cloudPortLookupDataSource struct {
	client *megaport.Client
}

// cloudPortLookupModel maps the data source schema data.
type cloudPortLookupModel struct {
	ConnectType   types.String     `tfsdk:"connect_type"`
	LocationID    types.Int64      `tfsdk:"location_id"`
	DiversityZone types.String     `tfsdk:"diversity_zone"`
	CompanyName   types.String     `tfsdk:"company_name"`
	VXCPermitted  types.Bool       `tfsdk:"vxc_permitted"`
	IncludeSecure types.Bool       `tfsdk:"include_secure"`
	Key           types.String     `tfsdk:"key"`
	Ports         []cloudPortModel `tfsdk:"ports"`
}

// cloudPortModel represents a single cloud port
type cloudPortModel struct {
	ProductUID    types.String `tfsdk:"product_uid"`
	ProductName   types.String `tfsdk:"product_name"`
	ConnectType   types.String `tfsdk:"connect_type"`
	CompanyUID    types.String `tfsdk:"company_uid"`
	CompanyName   types.String `tfsdk:"company_name"`
	DiversityZone types.String `tfsdk:"diversity_zone"`
	LocationID    types.Int64  `tfsdk:"location_id"`
	Speed         types.Int64  `tfsdk:"speed"`
	Rank          types.Int64  `tfsdk:"rank"`
	VXCPermitted  types.Bool   `tfsdk:"vxc_permitted"`
	IsSecure      types.Bool   `tfsdk:"is_secure"`
	// Secure product specific fields
	SecureKey types.String `tfsdk:"secure_key"`
	VLAN      types.Int64  `tfsdk:"vlan"`
}

// NewCloudPortLookupDataSource is a helper function to simplify the provider implementation.
func NewCloudPortLookupDataSource() datasource.DataSource {
	return &cloudPortLookupDataSource{}
}

// Metadata returns the data source type name.
func (d *cloudPortLookupDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cloud_port_lookup"
}

// Schema defines the schema for the data source.
func (d *cloudPortLookupDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Cloud Port Lookup Data Source. Returns an array of cloud service provider ports that match the specified criteria. 
This data source allows you to find and select the appropriate cloud ports for your VXC connections, including support for both public and secure partner ports.
Unlike the partner data source, this returns ALL matching ports, allowing you to choose the most suitable one for your requirements.`,
		Attributes: map[string]schema.Attribute{
			"connect_type": &schema.StringAttribute{
				Description: "The type of connection for the partner port. Filters by cloud provider connection types.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(
						"AWS", "AWSHC", "AZURE", "GOOGLE", "ORACLE", "IBM",
						"OUTSCALE", "TRANSIT", "FRANCEIX",
					),
				},
			},
			"location_id": &schema.Int64Attribute{
				Description: "Filter by the unique identifier of the location.",
				Optional:    true,
			},
			"diversity_zone": &schema.StringAttribute{
				Description: "Filter by diversity zone (red or blue).",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("red", "blue"),
				},
			},
			"company_name": &schema.StringAttribute{
				Description: "Filter by the name of the company that owns the partner port.",
				Optional:    true,
			},
			"vxc_permitted": &schema.BoolAttribute{
				Description: "Filter by whether VXCs are permitted on the port. Defaults to true if not specified.",
				Optional:    true,
			},
			"include_secure": &schema.BoolAttribute{
				Description: "Include secure partner ports (those requiring a key). Defaults to false. When true, you must also provide a key.",
				Optional:    true,
			},
			"key": &schema.StringAttribute{
				Description: "Key required for looking up secure partner ports (pairing key for GCP, service key for Azure/Oracle). Only used when include_secure is true.",
				Optional:    true,
				Sensitive:   true,
			},
			"ports": &schema.ListNestedAttribute{
				Description: "List of matching cloud ports.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"product_uid": &schema.StringAttribute{
							Description: "The unique identifier of the partner port.",
							Computed:    true,
						},
						"product_name": &schema.StringAttribute{
							Description: "The name of the partner port.",
							Computed:    true,
						},
						"connect_type": &schema.StringAttribute{
							Description: "The type of connection for the partner port.",
							Computed:    true,
						},
						"company_uid": &schema.StringAttribute{
							Description: "The unique identifier of the company that owns the partner port.",
							Computed:    true,
						},
						"company_name": &schema.StringAttribute{
							Description: "The name of the company that owns the partner port.",
							Computed:    true,
						},
						"diversity_zone": &schema.StringAttribute{
							Description: "The diversity zone of the partner port.",
							Computed:    true,
						},
						"location_id": &schema.Int64Attribute{
							Description: "The unique identifier of the location of the partner port.",
							Computed:    true,
						},
						"speed": &schema.Int64Attribute{
							Description: "The speed of the partner port in Mbps.",
							Computed:    true,
						},
						"rank": &schema.Int64Attribute{
							Description: "The rank of the partner port (lower is better).",
							Computed:    true,
						},
						"vxc_permitted": &schema.BoolAttribute{
							Description: "Whether VXCs are permitted on the partner port.",
							Computed:    true,
						},
						"is_secure": &schema.BoolAttribute{
							Description: "Whether this is a secure partner port requiring a key.",
							Computed:    true,
						},
						"secure_key": &schema.StringAttribute{
							Description: "Key for secure partner ports (pairing key for GCP, service key for Azure/Oracle).",
							Computed:    true,
							Sensitive:   true,
						},
						"vlan": &schema.Int64Attribute{
							Description: "VLAN ID for secure partner ports (if available from the API response).",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *cloudPortLookupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config cloudPortLookupModel

	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set defaults
	vxcPermitted := true
	if !config.VXCPermitted.IsNull() {
		vxcPermitted = config.VXCPermitted.ValueBool()
	}

	includeSecure := false
	if !config.IncludeSecure.IsNull() {
		includeSecure = config.IncludeSecure.ValueBool()
	}

	// Validate key is provided when include_secure is true
	if includeSecure && config.Key.IsNull() {
		resp.Diagnostics.AddError(
			"Missing key",
			"key is required when include_secure is true.",
		)
		return
	}

	var allPorts []cloudPortModel

	// Get public partner ports
	publicPorts, listErr := d.client.PartnerService.ListPartnerMegaports(ctx)
	if listErr != nil {
		resp.Diagnostics.AddError(
			"Error listing public partner ports",
			"Could not list public partner ports: "+listErr.Error(),
		)
		return
	}

	// Convert public ports
	for _, port := range publicPorts {
		cloudPort := cloudPortModel{}
		cloudPort.fromPublicPartnerPort(port)
		allPorts = append(allPorts, cloudPort)
	}

	// Get secure partner ports if requested
	if includeSecure && !config.Key.IsNull() {
		securePorts, secureErr := d.getSecurePorts(ctx, config.ConnectType.ValueString(), config.Key.ValueString())
		if secureErr != nil {
			resp.Diagnostics.AddWarning(
				"Could not retrieve secure partner ports",
				"Error listing secure partner ports: "+secureErr.Error()+". Continuing with public ports only.",
			)
		} else {
			allPorts = append(allPorts, securePorts...)
		}
	}

	// Apply filters
	filteredPorts := d.applyFilters(allPorts, config, vxcPermitted)

	// Sort by rank (lower is better), then by name
	slices.SortFunc(filteredPorts, func(a, b cloudPortModel) int {
		if a.Rank.ValueInt64() != b.Rank.ValueInt64() {
			return int(a.Rank.ValueInt64() - b.Rank.ValueInt64())
		}
		return strings.Compare(a.ProductName.ValueString(), b.ProductName.ValueString())
	})

	// Set the results
	config.Ports = filteredPorts

	// Set state
	diags = resp.State.Set(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Configure adds the provider configured client to the data source.
func (d *cloudPortLookupDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (cp *cloudPortModel) fromPublicPartnerPort(port *megaport.PartnerMegaport) {
	cp.ProductUID = types.StringValue(port.ProductUID)
	cp.ProductName = types.StringValue(port.ProductName)
	cp.ConnectType = types.StringValue(port.ConnectType)
	cp.CompanyUID = types.StringValue(port.CompanyUID)
	cp.CompanyName = types.StringValue(port.CompanyName)
	cp.DiversityZone = types.StringValue(port.DiversityZone)
	cp.LocationID = types.Int64Value(int64(port.LocationId))
	cp.Speed = types.Int64Value(int64(port.Speed))
	cp.Rank = types.Int64Value(int64(port.Rank))
	cp.VXCPermitted = types.BoolValue(port.VXCPermitted)
	cp.IsSecure = types.BoolValue(false)
	cp.SecureKey = types.StringNull()
	cp.VLAN = types.Int64Null()
}

func (cp *cloudPortModel) fromSecurePartnerPort(item *megaport.PartnerLookupItem, key string, vlan int) {
	cp.ProductUID = types.StringValue(item.ProductUID)
	cp.ProductName = types.StringValue(item.Name)
	cp.ConnectType = types.StringValue(item.Type)
	cp.CompanyUID = types.StringValue(fmt.Sprintf("%d", item.CompanyID)) // Convert int to string
	cp.CompanyName = types.StringValue(item.CompanyName)
	cp.DiversityZone = types.StringNull() // Not available in secure port response
	cp.LocationID = types.Int64Value(int64(item.LocationID))
	cp.Speed = types.Int64Value(int64(item.PortSpeed))
	cp.Rank = types.Int64Value(0)           // Not available in secure port response, default to 0
	cp.VXCPermitted = types.BoolValue(true) // Secure ports typically allow VXCs
	cp.IsSecure = types.BoolValue(true)
	cp.SecureKey = types.StringValue(key)
	cp.VLAN = types.Int64Value(int64(vlan))
}

func (d *cloudPortLookupDataSource) getSecurePorts(ctx context.Context, connectType string, key string) ([]cloudPortModel, error) {
	var securePorts []cloudPortModel

	// Map connect types to partner names for the secure API
	partners := d.getPartnersForConnectType(connectType)

	for _, partner := range partners {
		// Use ListPartnerPorts instead of LookupPartnerPorts to get all available ports
		partnerPorts, err := d.client.VXCService.ListPartnerPorts(ctx, &megaport.ListPartnerPortsRequest{
			Key:     key,
			Partner: partner,
		})
		if err != nil {
			continue // Skip this partner if there's an error
		}

		for _, port := range partnerPorts.Data.Megaports {
			cloudPort := cloudPortModel{}
			cloudPort.fromSecurePartnerPort(&port, partnerPorts.Data.ServiceKey, partnerPorts.Data.VLAN)
			securePorts = append(securePorts, cloudPort)
		}
	}

	return securePorts, nil
}

func (d *cloudPortLookupDataSource) getPartnersForConnectType(connectType string) []string {
	// Map connect types to partner names for the secure API
	switch strings.ToUpper(connectType) {
	case "GOOGLE":
		return []string{"GOOGLE"}
	case "ORACLE":
		return []string{"ORACLE"}
	case "AZURE":
		return []string{"AZURE"}
	case "":
		// If no connect type specified, try all secure partners
		return []string{"GOOGLE", "ORACLE", "AZURE"}
	default:
		return []string{}
	}
}

func (d *cloudPortLookupDataSource) applyFilters(ports []cloudPortModel, config cloudPortLookupModel, vxcPermitted bool) []cloudPortModel {
	var filtered []cloudPortModel

	for _, port := range ports {
		// Filter by VXC permitted
		if port.VXCPermitted.ValueBool() != vxcPermitted {
			continue
		}

		// Filter by connect type
		if !config.ConnectType.IsNull() &&
			!strings.EqualFold(port.ConnectType.ValueString(), config.ConnectType.ValueString()) {
			continue
		}

		// Filter by location ID
		if !config.LocationID.IsNull() &&
			port.LocationID.ValueInt64() != config.LocationID.ValueInt64() {
			continue
		}

		// Filter by diversity zone
		if !config.DiversityZone.IsNull() {
			// Skip ports that don't have diversity zone information
			if port.DiversityZone.IsNull() {
				continue
			}
			// Skip ports that don't match the requested diversity zone
			if !strings.EqualFold(port.DiversityZone.ValueString(), config.DiversityZone.ValueString()) {
				continue
			}
		}

		// Filter by company name
		if !config.CompanyName.IsNull() &&
			!strings.EqualFold(port.CompanyName.ValueString(), config.CompanyName.ValueString()) {
			continue
		}

		filtered = append(filtered, port)
	}

	return filtered
}
