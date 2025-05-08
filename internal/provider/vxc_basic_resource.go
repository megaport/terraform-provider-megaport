package provider

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
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
			"a_end": schema.SingleNestedAttribute{
				Description: "The current A-End configuration of the VXC.",
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"requested_product_uid": schema.StringAttribute{
						Description: "The Product UID requested by the user for the A-End configuration. Note: For cloud provider connections, the actual Product UID may differ from the requested UID due to Megaport's automatic port assignment for partner ports. This is expected behavior and ensures proper connectivity.",
						Required:    true,
					},
					"current_product_uid": schema.StringAttribute{
						Description: "The current product UID of the A-End configuration. The Megaport API may change a Partner Port from the Requested Port to a different Port in the same location and diversity zone.",
						Optional:    true,
						Computed:    true,
					},
					"vlan": schema.Int64Attribute{
						Description: "The VLAN of the A-End configuration. Values can range from 2 to 4093. If this value is set to 0 or not included, the Megaport system allocates a valid VLAN ID to the A-End configuration. To set this VLAN to untagged, set the VLAN value to null. For MCR endpoints, setting this to null will result in the API auto-assigning a VLAN ID. For MVE endpoints, setting this to null will use the VLAN associated with the VNIC specified in vnic_index.",
						Optional:    true,
						Computed:    true,
						Validators: []validator.Int64{
							int64validator.Between(2, 4093),
						},
					},
					"inner_vlan": schema.Int64Attribute{
						Description: "The inner VLAN of the A-End configuration. If the A-End ordered_vlan is untagged and set as -1, this field cannot be set by the API, as the VLAN of the A-End is designated as untagged.",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"vnic_index": schema.Int64Attribute{
						Description: "The network interface index of the A-End configuration.",
						Computed:    true,
						Optional:    true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
				},
			},
			"b_end": schema.SingleNestedAttribute{
				Description: "The current B-End configuration of the VXC.",
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"requested_product_uid": schema.StringAttribute{
						Description: "The Product UID requested by the user for the B-End configuration. Note: For cloud provider connections, the actual Product UID may differ from the requested UID due to Megaport's automatic port assignment for partner ports. This is expected behavior and ensures proper connectivity.",
						Optional:    true,
						Computed:    true,
					},
					"current_product_uid": schema.StringAttribute{
						Description: "The current product UID of the B-End configuration. The Megaport API may change a Partner Port on the end configuration from the Requested Port UID to a different Port in the same location and diversity zone.",
						Optional:    true,
						Computed:    true,
					},
					"vlan": schema.Int64Attribute{
						Description: "The VLAN of the B-End configuration. Values can range from 2 to 4093. If this value is set to 0 or not included, the Megaport system allocates a valid VLAN ID to the B-End configuration. To set this VLAN to untagged, set the VLAN value to -1. For MCR endpoints, setting this to null will result in the API auto-assigning a VLAN ID. For MVE endpoints, setting this to null will use the VLAN associated with the VNIC specified in vnic_index.",
						Optional:    true,
						Computed:    true,
						Validators: []validator.Int64{
							int64validator.Between(-1, 4093),
							int64validator.NoneOf(1),
							int64validator.NoneOf(0),
						},
					},
					"inner_vlan": schema.Int64Attribute{
						Description: "The inner VLAN of the B-End configuration. If the B-End ordered_vlan is untagged and set as -1, this field cannot be set by the API, as the VLAN of the B-End is designated as untagged.",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"vnic_index": schema.Int64Attribute{
						Description: "The network interface index of the B-End configuration.",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
				},
			},
			"a_end_partner_config": schema.SingleNestedAttribute{
				Description: `The partner configuration of the A-End order configuration. Contains CSP and/or BGP Configuration settings. For any partner configuration besides "vrouter", this configuration cannot be changed after the VXC is created and if it is modified, the VXC will be deleted and re-created. Imported VXCs do not have this field populated by the API, so the initially provided configuration will be ignored as it can't be verified to be correct. If the user wants to change the configuration after importing the resource, they can then do so by changing the field after importing the resource and running terraform apply.`,
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"partner": schema.StringAttribute{
						Description: "The partner of the partner configuration.",
						Required:    true,
						Validators: []validator.String{
							stringvalidator.OneOf("aws", "azure", "google", "oracle", "ibm", "vrouter", "transit", "a-end"),
						},
					},
					"aws_config": schema.SingleNestedAttribute{
						Description: "The AWS partner configuration.",
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"connect_type": schema.StringAttribute{
								Description: "The connection type of the partner configuration. Required for AWS partner configurations.",
								Required:    true,
							},
							"type": schema.StringAttribute{
								Description: "The type of the partner configuration. Required for AWS partner configurations.",
								Required:    true,
							},
							"owner_account": schema.StringAttribute{
								Description: "The owner AWS account of the partner configuration. Required for AWS partner configurations.",
								Required:    true,
							},
							"asn": schema.Int64Attribute{
								Description: "The ASN of the partner configuration.",
								Optional:    true,
							},
							"amazon_asn": schema.Int64Attribute{
								Description: "The Amazon ASN of the partner configuration.",
								Optional:    true,
							},
							"auth_key": schema.StringAttribute{
								Description: "The authentication key of the partner configuration.",
								Optional:    true,
							},
							"prefixes": schema.StringAttribute{
								Description: "The prefixes of the partner configuration.",
								Optional:    true,
							},
							"customer_ip_address": schema.StringAttribute{
								Description: "The customer IP address of the partner configuration.",
								Optional:    true,
							},
							"amazon_ip_address": schema.StringAttribute{
								Description: "The Amazon IP address of the partner configuration.",
								Optional:    true,
							},
							"name": schema.StringAttribute{
								Description: "The name of the partner configuration.",
								Required:    true,
							},
						},
					},
					"azure_config": schema.SingleNestedAttribute{
						Description: "The Azure partner configuration.",
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"service_key": schema.StringAttribute{
								Description: "The service key of the partner configuration. Required for Azure partner configurations.",
								Required:    true,
								Sensitive:   true,
							},
							"port_choice": schema.StringAttribute{
								Description: "Which port to choose when building the VXC. Can either be 'primary' or 'secondary'.",
								Required:    true,
								Validators: []validator.String{
									stringvalidator.OneOf("primary", "secondary"),
								},
							},
							"peers": schema.ListNestedAttribute{
								Description: "The peers of the partner configuration. If this is set, the user must delete any Azure resources associated with the VXC on Azure before deleting the VXC.",
								Optional:    true,
								NestedObject: schema.NestedAttributeObject{
									Attributes: map[string]schema.Attribute{
										"type": schema.StringAttribute{
											Description: "The type of the peer.",
											Required:    true,
										},
										"peer_asn": schema.StringAttribute{
											Description: "The peer ASN of the peer.",
											Optional:    true,
										},
										"primary_subnet": schema.StringAttribute{
											Description: "The primary subnet of the peer.",
											Optional:    true,
										},
										"secondary_subnet": schema.StringAttribute{
											Description: "The secondary subnet of the peer.",
											Optional:    true,
										},
										"prefixes": schema.StringAttribute{
											Description: "The prefixes of the peer.",
											Optional:    true,
										},
										"shared_key": schema.StringAttribute{
											Description: "The shared key of the peer.",
											Optional:    true,
										},
										"vlan": schema.Int64Attribute{
											Description: "The VLAN of the peer.",
											Optional:    true,
										},
									},
								},
							},
						},
					},
					"google_config": schema.SingleNestedAttribute{
						Description: "The Google partner configuration.",
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"pairing_key": schema.StringAttribute{
								Description: "The pairing key of the partner configuration. Required for Google partner configurations.",
								Required:    true,
							},
						},
					},
					"ibm_config": schema.SingleNestedAttribute{
						Description: "The IBM partner configuration.",
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"account_id": schema.StringAttribute{
								Description: "Customer's IBM Acount ID. Required for all IBM partner configurations.",
								Required:    true,
							},
							"customer_asn": schema.Int64Attribute{
								Description: "Customer's ASN. Valid ranges: 1-64495, 64999, 131072-4199999999, 4201000000-4201064511. Required unless the connection at the other end of the VXC is an MCR.",
								Optional:    true,
							},
							"name": schema.StringAttribute{
								Description: `Description of this connection for identification purposes. Max 100 characters from 0-9 a-z A-Z / - _ , Defaults to "MEGAPORT"`,
								Optional:    true,
								Validators:  []validator.String{stringvalidator.LengthAtMost(100)},
							},
							"customer_ip_address": schema.StringAttribute{
								Description: "Customer IPv4 network address including subnet mask. Default is /30 assigned from 169.254.0.0/16.",
								Optional:    true,
							},
							"provider_ip_address": schema.StringAttribute{
								Description: "Provider IPv4 network address including subnet mask.",
								Optional:    true,
							},
						},
					},
					"oracle_config": schema.SingleNestedAttribute{
						Description: "The Oracle partner configuration.",
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"virtual_circuit_id": schema.StringAttribute{
								Description: "The virtual circuit ID of the partner configuration. Required for Oracle partner configurations.",
								Required:    true,
							},
						},
					},
					"vrouter_config": schema.SingleNestedAttribute{
						Description: "The partner configuration of the virtual router configuration.",
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"interfaces": schema.ListNestedAttribute{
								Description: "The interfaces of the partner configuration.",
								Required:    true,
								NestedObject: schema.NestedAttributeObject{
									Attributes: map[string]schema.Attribute{
										"ip_addresses": schema.ListAttribute{
											Description: "The IP addresses of the partner configuration.",
											Optional:    true,
											ElementType: types.StringType,
										},
										"ip_routes": schema.ListNestedAttribute{
											Description: "The IP routes of the partner configuration.",
											Optional:    true,
											NestedObject: schema.NestedAttributeObject{
												Attributes: map[string]schema.Attribute{
													"prefix": schema.StringAttribute{
														Description: "The prefix of the IP route.",
														Optional:    true,
													},
													"description": schema.StringAttribute{
														Description: "The description of the IP route.",
														Optional:    true,
													},
													"next_hop": schema.StringAttribute{
														Description: "The next hop of the IP route.",
														Optional:    true,
													},
												},
											},
										},
										"nat_ip_addresses": schema.ListAttribute{
											Description: "The NAT IP addresses of the partner configuration.",
											Optional:    true,
											ElementType: types.StringType,
										},
										"bfd": schema.SingleNestedAttribute{
											Description: "The BFD of the partner configuration interface.",
											Optional:    true,
											Attributes: map[string]schema.Attribute{
												"tx_interval": schema.Int64Attribute{
													Description: "The transmit interval of the BFD.",
													Optional:    true,
												},
												"rx_interval": schema.Int64Attribute{
													Description: "The receive interval of the BFD.",
													Optional:    true,
												},
												"multiplier": schema.Int64Attribute{
													Description: "The multiplier of the BFD.",
													Optional:    true,
												},
											},
										},
										"vlan": schema.Int64Attribute{
											Description: "Inner-VLAN for implicit Q-inQ VXCs. Typically used only for Azure VXCs. The default is no inner-vlan.",
											Optional:    true,
										},
										"bgp_connections": schema.ListNestedAttribute{
											Description: "The BGP connections of the partner configuration interface.",
											Optional:    true,
											NestedObject: schema.NestedAttributeObject{
												Attributes: map[string]schema.Attribute{
													"peer_type": schema.StringAttribute{
														Description: "Defines the default BGP routing policy for this BGP connection. The default depends on the CSP type of the far end of this VXC.",
														Optional:    true,
														Validators: []validator.String{
															stringvalidator.OneOf("NON_CLOUD", "PRIV_CLOUD", "PUB_CLOUD"),
														},
													},
													"peer_asn": schema.Int64Attribute{
														Description: "The peer ASN of the BGP connection.",
														Optional:    true,
													},
													"local_asn": schema.Int64Attribute{
														Description: "The local ASN of the BGP connection.",
														Optional:    true,
													},
													"local_ip_address": schema.StringAttribute{
														Description: "The local IP address of the BGP connection.",
														Optional:    true,
													},
													"peer_ip_address": schema.StringAttribute{
														Description: "The peer IP address of the BGP connection.",
														Optional:    true,
													},
													"password": schema.StringAttribute{
														Description: "The password of the BGP connection.",
														Optional:    true,
													},
													"shutdown": schema.BoolAttribute{
														Description: "Whether the BGP connection is shut down.",
														Optional:    true,
													},
													"description": schema.StringAttribute{
														Description: "The description of the BGP connection.",
														Optional:    true,
													},
													"med_in": schema.Int64Attribute{
														Description: "The MED in of the BGP connection.",
														Optional:    true,
													},
													"med_out": schema.Int64Attribute{
														Description: "The MED out of the BGP connection.",
														Optional:    true,
													},
													"bfd_enabled": schema.BoolAttribute{
														Description: "Whether BFD is enabled for the BGP connection.",
														Optional:    true,
													},
													"export_policy": schema.StringAttribute{
														Description: "The export policy of the BGP connection.",
														Optional:    true,
													},
													"permit_export_to": schema.ListAttribute{
														Description: "The permitted export to of the BGP connection.",
														Optional:    true,
														ElementType: types.StringType,
													},
													"deny_export_to": schema.ListAttribute{
														Description: "The denied export to of the BGP connection.",
														Optional:    true,
														ElementType: types.StringType,
													},
													"import_whitelist": schema.StringAttribute{
														Description: "The import whitelist of the BGP connection.",
														Optional:    true,
													},
													"import_blacklist": schema.StringAttribute{
														Description: "The import blacklist of the BGP connection.",
														Optional:    true,
													},
													"export_whitelist": schema.StringAttribute{
														Description: "The export whitelist of the BGP connection.",
														Optional:    true,
													},
													"export_blacklist": schema.StringAttribute{
														Description: "The export blacklist of the BGP connection.",
														Optional:    true,
													},
													"as_path_prepend_count": schema.Int64Attribute{
														Description: "The AS path prepend count of the BGP connection. Minimum value of 0 and maximum value of 10.",
														Optional:    true,
														Validators:  []validator.Int64{int64validator.Between(0, 10)},
													},
												},
											},
										},
									},
								},
							},
						},
					},
					"partner_a_end_config": schema.SingleNestedAttribute{
						Description:        "The partner configuration of the A-End order configuration. Only exists for A-End Configurations. DEPRECATED: Use vrouter_config instead.",
						Optional:           true,
						DeprecationMessage: "Deprecated: Use `vrouter_config` instead.",
						Attributes: map[string]schema.Attribute{
							"interfaces": schema.ListNestedAttribute{
								Description: "The interfaces of the partner configuration.",
								Required:    true,
								NestedObject: schema.NestedAttributeObject{
									Attributes: map[string]schema.Attribute{
										"ip_addresses": schema.ListAttribute{
											Description: "The IP addresses of the partner configuration.",
											Optional:    true,
											ElementType: types.StringType,
										},
										"ip_routes": schema.ListNestedAttribute{
											Description: "The IP routes of the partner configuration.",
											Optional:    true,
											NestedObject: schema.NestedAttributeObject{
												Attributes: map[string]schema.Attribute{
													"prefix": schema.StringAttribute{
														Description: "The prefix of the IP route.",
														Optional:    true,
													},
													"description": schema.StringAttribute{
														Description: "The description of the IP route.",
														Optional:    true,
													},
													"next_hop": schema.StringAttribute{
														Description: "The next hop of the IP route.",
														Optional:    true,
													},
												},
											},
										},
										"nat_ip_addresses": schema.ListAttribute{
											Description: "The NAT IP addresses of the partner configuration.",
											Optional:    true,
											ElementType: types.StringType,
										},
										"bfd": schema.SingleNestedAttribute{
											Description: "The BFD of the partner configuration interface.",
											Optional:    true,
											Attributes: map[string]schema.Attribute{
												"tx_interval": schema.Int64Attribute{
													Description: "The transmit interval of the BFD.",
													Optional:    true,
												},
												"rx_interval": schema.Int64Attribute{
													Description: "The receive interval of the BFD.",
													Optional:    true,
												},
												"multiplier": schema.Int64Attribute{
													Description: "The multiplier of the BFD.",
													Optional:    true,
												},
											},
										},
										"bgp_connections": schema.ListNestedAttribute{
											Description: "The BGP connections of the partner configuration interface.",
											Optional:    true,
											NestedObject: schema.NestedAttributeObject{
												Attributes: map[string]schema.Attribute{
													"peer_asn": schema.Int64Attribute{
														Description: "The peer ASN of the BGP connection.",
														Optional:    true,
													},
													"local_asn": schema.Int64Attribute{
														Description: "The local ASN of the BGP connection.",
														Optional:    true,
													},
													"local_ip_address": schema.StringAttribute{
														Description: "The local IP address of the BGP connection.",
														Optional:    true,
													},
													"peer_ip_address": schema.StringAttribute{
														Description: "The peer IP address of the BGP connection.",
														Optional:    true,
													},
													"password": schema.StringAttribute{
														Description: "The password of the BGP connection.",
														Optional:    true,
													},
													"shutdown": schema.BoolAttribute{
														Description: "Whether the BGP connection is shut down.",
														Optional:    true,
													},
													"description": schema.StringAttribute{
														Description: "The description of the BGP connection.",
														Optional:    true,
													},
													"med_in": schema.Int64Attribute{
														Description: "The MED in of the BGP connection.",
														Optional:    true,
													},
													"med_out": schema.Int64Attribute{
														Description: "The MED out of the BGP connection.",
														Optional:    true,
													},
													"bfd_enabled": schema.BoolAttribute{
														Description: "Whether BFD is enabled for the BGP connection.",
														Optional:    true,
													},
													"export_policy": schema.StringAttribute{
														Description: "The export policy of the BGP connection.",
														Optional:    true,
													},
													"permit_export_to": schema.ListAttribute{
														Description: "The permitted export to of the BGP connection.",
														Optional:    true,
														ElementType: types.StringType,
													},
													"deny_export_to": schema.ListAttribute{
														Description: "The denied export to of the BGP connection.",
														Optional:    true,
														ElementType: types.StringType,
													},
													"import_whitelist": schema.StringAttribute{
														Description: "The import whitelist of the BGP connection.",
														Optional:    true,
													},
													"import_blacklist": schema.StringAttribute{
														Description: "The import blacklist of the BGP connection.",
														Optional:    true,
													},
													"export_whitelist": schema.StringAttribute{
														Description: "The export whitelist of the BGP connection.",
														Optional:    true,
													},
													"export_blacklist": schema.StringAttribute{
														Description: "The export blacklist of the BGP connection.",
														Optional:    true,
													},
													"as_path_prepend_count": schema.Int64Attribute{
														Description: "The AS path prepend count of the BGP connection. Minimum value of 0 and maximum value of 10.",
														Optional:    true,
														Validators:  []validator.Int64{int64validator.Between(0, 10)},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			"b_end_partner_config": schema.SingleNestedAttribute{
				Description: `The partner configuration of the B-End order configuration. Contains CSP and/or BGP Configuration settings. For any partner configuration besides "vrouter", this configuration cannot be changed after the VXC is created and if it is modified, the VXC will be deleted and re-created. Imported VXCs do not have this field populated by the API, so the initially provided configuration will be ignored as it can't be verified to be correct. If the user wants to change the configuration after importing the resource, they can then do so by changing the field after importing the resource and running terraform apply.`,
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"partner": schema.StringAttribute{
						Description: "The partner of the partner configuration.",
						Required:    true,
						Validators: []validator.String{
							stringvalidator.OneOf("aws", "azure", "google", "oracle", "ibm", "transit", "vrouter"),
						},
					},
					"aws_config": schema.SingleNestedAttribute{
						Description: "The AWS partner configuration.",
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"connect_type": schema.StringAttribute{
								Description: "The connection type of the partner configuration. Required for AWS partner configurations.",
								Required:    true,
							},
							"type": schema.StringAttribute{
								Description: "The type of the partner configuration. Required for AWS partner configurations.",
								Required:    true,
							},
							"owner_account": schema.StringAttribute{
								Description: "The owner AWS account of the partner configuration. Required for AWS partner configurations.",
								Required:    true,
							},
							"asn": schema.Int64Attribute{
								Description: "The ASN of the partner configuration.",
								Optional:    true,
							},
							"amazon_asn": schema.Int64Attribute{
								Description: "The Amazon ASN of the partner configuration.",
								Optional:    true,
							},
							"auth_key": schema.StringAttribute{
								Description: "The authentication key of the partner configuration.",
								Optional:    true,
							},
							"prefixes": schema.StringAttribute{
								Description: "The prefixes of the partner configuration.",
								Optional:    true,
							},
							"customer_ip_address": schema.StringAttribute{
								Description: "The customer IP address of the partner configuration.",
								Optional:    true,
							},
							"amazon_ip_address": schema.StringAttribute{
								Description: "The Amazon IP address of the partner configuration.",
								Optional:    true,
							},
							"name": schema.StringAttribute{
								Description: "The name of the partner configuration.",
								Required:    true,
							},
						},
					},
					"azure_config": schema.SingleNestedAttribute{
						Description: "The Azure partner configuration.",
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"service_key": schema.StringAttribute{
								Description: "The service key of the partner configuration. Required for Azure partner configurations.",
								Required:    true,
								Sensitive:   true,
							},
							"port_choice": schema.StringAttribute{
								Description: "Which port to choose when building the VXC. Can either be 'primary' or 'secondary'.",
								Required:    true,
								Validators: []validator.String{
									stringvalidator.OneOf("primary", "secondary"),
								},
							},
							"peers": schema.ListNestedAttribute{
								Description: "The peers of the partner configuration. If this is set, the user must delete any Azure resources associated with the VXC on Azure before deleting the VXC.",
								Optional:    true,
								NestedObject: schema.NestedAttributeObject{
									Attributes: map[string]schema.Attribute{
										"type": schema.StringAttribute{
											Description: "The type of the peer.",
											Required:    true,
										},
										"peer_asn": schema.StringAttribute{
											Description: "The peer ASN of the peer.",
											Optional:    true,
										},
										"primary_subnet": schema.StringAttribute{
											Description: "The primary subnet of the peer.",
											Optional:    true,
										},
										"secondary_subnet": schema.StringAttribute{
											Description: "The secondary subnet of the peer.",
											Optional:    true,
										},
										"prefixes": schema.StringAttribute{
											Description: "The prefixes of the peer.",
											Optional:    true,
										},
										"shared_key": schema.StringAttribute{
											Description: "The shared key of the peer.",
											Optional:    true,
										},
										"vlan": schema.Int64Attribute{
											Description: "The VLAN of the peer.",
											Optional:    true,
										},
									},
								},
							},
						},
					},
					"google_config": schema.SingleNestedAttribute{
						Description: "The Google partner configuration.",
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"pairing_key": schema.StringAttribute{
								Description: "The pairing key of the partner configuration. Required for Google partner configurations.",
								Required:    true,
							},
						},
					},
					"ibm_config": schema.SingleNestedAttribute{
						Description: "The IBM partner configuration.",
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"account_id": schema.StringAttribute{
								Description: "Customer's IBM Acount ID. Required for all IBM partner configurations.",
								Required:    true,
							},
							"customer_asn": schema.Int64Attribute{
								Description: "Customer's ASN. Valid ranges: 1-64495, 64999, 131072-4199999999, 4201000000-4201064511. Required unless the connection at the other end of the VXC is an MCR.",
								Optional:    true,
							},
							"name": schema.StringAttribute{
								Description: `Description of this connection for identification purposes. Max 100 characters from 0-9 a-z A-Z / - _ , Defaults to "MEGAPORT"`,
								Optional:    true,
								Validators:  []validator.String{stringvalidator.LengthAtMost(100)},
							},
							"customer_ip_address": schema.StringAttribute{
								Description: "Customer IPv4 network address including subnet mask. Default is /30 assigned from 169.254.0.0/16.",
								Optional:    true,
							},
							"provider_ip_address": schema.StringAttribute{
								Description: "Provider IPv4 network address including subnet mask.",
								Optional:    true,
							},
						},
					},
					"oracle_config": schema.SingleNestedAttribute{
						Description: "The Oracle partner configuration.",
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"virtual_circuit_id": schema.StringAttribute{
								Description: "The virtual circuit ID of the partner configuration. Required for Oracle partner configurations.",
								Required:    true,
							},
						},
					},
					"vrouter_config": schema.SingleNestedAttribute{
						Description: "The partner configuration of the virtual router configuration.",
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"interfaces": schema.ListNestedAttribute{
								Description: "The interfaces of the partner configuration.",
								Required:    true,
								NestedObject: schema.NestedAttributeObject{
									Attributes: map[string]schema.Attribute{
										"ip_addresses": schema.ListAttribute{
											Description: "The IP addresses of the partner configuration.",
											Optional:    true,
											ElementType: types.StringType,
										},
										"ip_routes": schema.ListNestedAttribute{
											Description: "The IP routes of the partner configuration.",
											Optional:    true,
											NestedObject: schema.NestedAttributeObject{
												Attributes: map[string]schema.Attribute{
													"prefix": schema.StringAttribute{
														Description: "The prefix of the IP route.",
														Optional:    true,
													},
													"description": schema.StringAttribute{
														Description: "The description of the IP route.",
														Optional:    true,
													},
													"next_hop": schema.StringAttribute{
														Description: "The next hop of the IP route.",
														Optional:    true,
													},
												},
											},
										},
										"nat_ip_addresses": schema.ListAttribute{
											Description: "The NAT IP addresses of the partner configuration.",
											Optional:    true,
											ElementType: types.StringType,
										},
										"bfd": schema.SingleNestedAttribute{
											Description: "The BFD of the partner configuration interface.",
											Optional:    true,
											Attributes: map[string]schema.Attribute{
												"tx_interval": schema.Int64Attribute{
													Description: "The transmit interval of the BFD.",
													Optional:    true,
												},
												"rx_interval": schema.Int64Attribute{
													Description: "The receive interval of the BFD.",
													Optional:    true,
												},
												"multiplier": schema.Int64Attribute{
													Description: "The multiplier of the BFD.",
													Optional:    true,
												},
											},
										},
										"vlan": schema.Int64Attribute{
											Description: "Inner-VLAN for implicit Q-inQ VXCs. Typically used only for Azure VXCs. The default is no inner-vlan.",
											Optional:    true,
										},
										"bgp_connections": schema.ListNestedAttribute{
											Description: "The BGP connections of the partner configuration interface.",
											Optional:    true,
											NestedObject: schema.NestedAttributeObject{
												Attributes: map[string]schema.Attribute{
													"peer_asn": schema.Int64Attribute{
														Description: "The peer ASN of the BGP connection.",
														Optional:    true,
													},
													"local_asn": schema.Int64Attribute{
														Description: "The local ASN of the BGP connection.",
														Optional:    true,
													},
													"peer_type": schema.StringAttribute{
														Description: "Defines the default BGP routing policy for this BGP connection. The default depends on the CSP type of the far end of this VXC.",
														Optional:    true,
														Validators: []validator.String{
															stringvalidator.OneOf("NON_CLOUD", "PRIV_CLOUD", "PUB_CLOUD"),
														},
													},
													"local_ip_address": schema.StringAttribute{
														Description: "The local IP address of the BGP connection.",
														Optional:    true,
													},
													"peer_ip_address": schema.StringAttribute{
														Description: "The peer IP address of the BGP connection.",
														Optional:    true,
													},
													"password": schema.StringAttribute{
														Description: "The password of the BGP connection.",
														Optional:    true,
													},
													"shutdown": schema.BoolAttribute{
														Description: "Whether the BGP connection is shut down.",
														Optional:    true,
													},
													"description": schema.StringAttribute{
														Description: "The description of the BGP connection.",
														Optional:    true,
													},
													"med_in": schema.Int64Attribute{
														Description: "The MED in of the BGP connection.",
														Optional:    true,
													},
													"med_out": schema.Int64Attribute{
														Description: "The MED out of the BGP connection.",
														Optional:    true,
													},
													"bfd_enabled": schema.BoolAttribute{
														Description: "Whether BFD is enabled for the BGP connection.",
														Optional:    true,
													},
													"export_policy": schema.StringAttribute{
														Description: "The export policy of the BGP connection.",
														Optional:    true,
													},
													"permit_export_to": schema.ListAttribute{
														Description: "The permitted export to of the BGP connection.",
														Optional:    true,
														ElementType: types.StringType,
													},
													"deny_export_to": schema.ListAttribute{
														Description: "The denied export to of the BGP connection.",
														Optional:    true,
														ElementType: types.StringType,
													},
													"import_whitelist": schema.StringAttribute{
														Description: "The import whitelist of the BGP connection.",
														Optional:    true,
													},
													"import_blacklist": schema.StringAttribute{
														Description: "The import blacklist of the BGP connection.",
														Optional:    true,
													},
													"export_whitelist": schema.StringAttribute{
														Description: "The export whitelist of the BGP connection.",
														Optional:    true,
													},
													"export_blacklist": schema.StringAttribute{
														Description: "The export blacklist of the BGP connection.",
														Optional:    true,
													},
													"as_path_prepend_count": schema.Int64Attribute{
														Description: "The AS path prepend count of the BGP connection. Minimum value of 0 and maximum value of 10.",
														Optional:    true,
														Validators:  []validator.Int64{int64validator.Between(0, 10)},
													},
												},
											},
										},
									},
								},
							},
						},
					},
					"partner_a_end_config": schema.SingleNestedAttribute{
						Description:        "The partner configuration of the A-End order configuration. Only exists for A-End Configurations, invalid on B-End Partner Config. DEPRECATED: Use vrouter_config instead.",
						Optional:           true,
						DeprecationMessage: "Deprecated: Use `vrouter_config` instead.",
						Attributes: map[string]schema.Attribute{
							"interfaces": schema.ListNestedAttribute{
								Description: "The interfaces of the partner configuration.",
								Required:    true,
								NestedObject: schema.NestedAttributeObject{
									Attributes: map[string]schema.Attribute{
										"ip_addresses": schema.ListAttribute{
											Description: "The IP addresses of the partner configuration.",
											Optional:    true,
											ElementType: types.StringType,
										},
										"ip_routes": schema.ListNestedAttribute{
											Description: "The IP routes of the partner configuration.",
											Optional:    true,
											NestedObject: schema.NestedAttributeObject{
												Attributes: map[string]schema.Attribute{
													"prefix": schema.StringAttribute{
														Description: "The prefix of the IP route.",
														Optional:    true,
													},
													"description": schema.StringAttribute{
														Description: "The description of the IP route.",
														Optional:    true,
													},
													"next_hop": schema.StringAttribute{
														Description: "The next hop of the IP route.",
														Optional:    true,
													},
												},
											},
										},
										"nat_ip_addresses": schema.ListAttribute{
											Description: "The NAT IP addresses of the partner configuration.",
											Optional:    true,
											ElementType: types.StringType,
										},
										"bfd": schema.SingleNestedAttribute{
											Description: "The BFD of the partner configuration interface.",
											Optional:    true,
											Attributes: map[string]schema.Attribute{
												"tx_interval": schema.Int64Attribute{
													Description: "The transmit interval of the BFD.",
													Optional:    true,
												},
												"rx_interval": schema.Int64Attribute{
													Description: "The receive interval of the BFD.",
													Optional:    true,
												},
												"multiplier": schema.Int64Attribute{
													Description: "The multiplier of the BFD.",
													Optional:    true,
												},
											},
										},
										"bgp_connections": schema.ListNestedAttribute{
											Description: "The BGP connections of the partner configuration interface.",
											Optional:    true,
											NestedObject: schema.NestedAttributeObject{
												Attributes: map[string]schema.Attribute{
													"peer_asn": schema.Int64Attribute{
														Description: "The peer ASN of the BGP connection.",
														Optional:    true,
													},
													"local_asn": schema.Int64Attribute{
														Description: "The local ASN of the BGP connection.",
														Optional:    true,
													},
													"local_ip_address": schema.StringAttribute{
														Description: "The local IP address of the BGP connection.",
														Optional:    true,
													},
													"peer_ip_address": schema.StringAttribute{
														Description: "The peer IP address of the BGP connection.",
														Optional:    true,
													},
													"password": schema.StringAttribute{
														Description: "The password of the BGP connection.",
														Optional:    true,
													},
													"shutdown": schema.BoolAttribute{
														Description: "Whether the BGP connection is shut down.",
														Optional:    true,
													},
													"description": schema.StringAttribute{
														Description: "The description of the BGP connection.",
														Optional:    true,
													},
													"med_in": schema.Int64Attribute{
														Description: "The MED in of the BGP connection.",
														Optional:    true,
													},
													"med_out": schema.Int64Attribute{
														Description: "The MED out of the BGP connection.",
														Optional:    true,
													},
													"bfd_enabled": schema.BoolAttribute{
														Description: "Whether BFD is enabled for the BGP connection.",
														Optional:    true,
													},
													"export_policy": schema.StringAttribute{
														Description: "The export policy of the BGP connection.",
														Optional:    true,
													},
													"permit_export_to": schema.ListAttribute{
														Description: "The permitted export to of the BGP connection.",
														Optional:    true,
														ElementType: types.StringType,
													},
													"deny_export_to": schema.ListAttribute{
														Description: "The denied export to of the BGP connection.",
														Optional:    true,
														ElementType: types.StringType,
													},
													"import_whitelist": schema.StringAttribute{
														Description: "The import whitelist of the BGP connection.",
														Optional:    true,
													},
													"import_blacklist": schema.StringAttribute{
														Description: "The import blacklist of the BGP connection.",
														Optional:    true,
													},
													"export_whitelist": schema.StringAttribute{
														Description: "The export whitelist of the BGP connection.",
														Optional:    true,
													},
													"export_blacklist": schema.StringAttribute{
														Description: "The export blacklist of the BGP connection.",
														Optional:    true,
													},
													"as_path_prepend_count": schema.Int64Attribute{
														Description: "The AS path prepend count of the BGP connection. Minimum value of 0 and maximum value of 10.",
														Optional:    true,
														Validators:  []validator.Int64{int64validator.Between(0, 10)},
													},
												},
											},
										},
									},
								},
							},
						},
					},
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

	aEndDiags, aEndState, aEndPartnerObj, aEndMegaportPartnerConfig, aEndRequestedProductUID, aEndVLAN, aEndInnerVLAN := r.makeUpdateEndConfig(ctx, plan.Name.ValueString(), plan.AEndConfiguration, state.AEndConfiguration, plan.AEndPartnerConfig, state.AEndPartnerConfig)
	resp.Diagnostics.Append(aEndDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.AEndPartnerConfig = aEndPartnerObj
	updateReq.AEndPartnerConfig = aEndMegaportPartnerConfig
	updateReq.AEndVLAN = aEndVLAN
	updateReq.AEndInnerVLAN = aEndInnerVLAN
	updateReq.AEndProductUID = aEndRequestedProductUID

	bEndDiags, bEndState, bEndPartnerObj, bEndMegaportPartnerConfig, bEndRequestedProductUID, bEndVLAN, bEndInnerVLAN := r.makeUpdateEndConfig(ctx, plan.Name.ValueString(), plan.BEndConfiguration, state.BEndConfiguration, plan.BEndPartnerConfig, state.BEndPartnerConfig)
	resp.Diagnostics.Append(bEndDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.BEndPartnerConfig = bEndPartnerObj
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

	// Get refreshed vxc value from API
	vxc, err := r.client.VXCService.GetVXC(ctx, state.UID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading VXC",
			"Could not read VXC with ID "+state.UID.ValueString()+": "+err.Error(),
		)
		return
	}

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
