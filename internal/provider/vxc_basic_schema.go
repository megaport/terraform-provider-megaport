package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	mcrPartnerConfigSchema = schema.SingleNestedAttribute{
		Description: "The partner configuration of the MCR configuration.",
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
	}

	vxcBasicPartnerTypeSchema = schema.StringAttribute{
		Description: "The partner of the partner configuration.",
		Required:    true,
		Validators: []validator.String{
			stringvalidator.OneOf("aws", "azure", "google", "oracle", "ibm", "mcr", "transit"),
		},
	}
)

// Schema defines the schema for the resource.
func (r *vxcBasicResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Virtual Cross Connect (VXC) Resource for the Megaport Terraform Provider. This resource allows you to create, modify, and update VXCs. VXCs are Layer 2 Ethernet circuits providing private, flexible, and on-demand connections between any of the locations on the Megaport network with 1 Mbps to 100 Gbps of capacity. This is a basic resource for VXC management.",
		Attributes: map[string]schema.Attribute{
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
					"partner":       vxcBasicPartnerTypeSchema,
					"aws_config":    awsPartnerConfigSchema,
					"azure_config":  azurePartnerConfigSchema,
					"google_config": googlePartnerConfigSchema,
					"ibm_config":    ibmPartnerConfigSchema,
					"oracle_config": oraclePartnerConfigSchema,
					"mcr_config":    mcrPartnerConfigSchema,
				},
			},
			"b_end_partner_config": schema.SingleNestedAttribute{
				Description: `The partner configuration of the B-End order configuration. Contains CSP and/or BGP Configuration settings. For any partner configuration besides "vrouter", this configuration cannot be changed after the VXC is created and if it is modified, the VXC will be deleted and re-created. Imported VXCs do not have this field populated by the API, so the initially provided configuration will be ignored as it can't be verified to be correct. If the user wants to change the configuration after importing the resource, they can then do so by changing the field after importing the resource and running terraform apply.`,
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"partner":       vxcBasicPartnerTypeSchema,
					"aws_config":    awsPartnerConfigSchema,
					"azure_config":  azurePartnerConfigSchema,
					"google_config": googlePartnerConfigSchema,
					"ibm_config":    ibmPartnerConfigSchema,
					"oracle_config": oraclePartnerConfigSchema,
					"mcr_config":    mcrPartnerConfigSchema,
				},
			},
		},
	}
}

const (
	VXC_A_END = "A-End"
	VXC_B_END = "B-End"
)

func getEndConfigSchema(end string) schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Description: fmt.Sprintf("The current %s configuration of the VXC.", end),
		Required:    true,
		Attributes: map[string]schema.Attribute{
			"requested_product_uid": schema.StringAttribute{
				Description: fmt.Sprintf("The Product UID requested by the user for the %s configuration. Note: For cloud provider connections, the actual Product UID may differ from the requested UID due to Megaport's automatic port assignment for partner ports. This is expected behavior and ensures proper connectivity.", end),
				Required:    isAEndConfig(end),
				Optional:    isBEndConfig(end),
				Computed:    isBEndConfig(end),
			},
			"current_product_uid": schema.StringAttribute{
				Description: fmt.Sprintf("The current product UID of the %s configuration. The Megaport API may change a Partner Port on the end configuration from the Requested Port UID to a different Port in the same location and diversity zone.", end),
				Optional:    true,
				Computed:    true,
			},
			"vlan": schema.Int64Attribute{
				Description: fmt.Sprintf("The VLAN of the %s configuration. Values can range from 2 to 4093. If this value is set to 0 or not included, the Megaport system allocates a valid VLAN ID to the %s configuration. To set this VLAN to untagged, set the VLAN value to -1. For MCR endpoints, setting this to null will result in the API auto-assigning a VLAN ID. For MVE endpoints, setting this to null will use the VLAN associated with the VNIC specified in vnic_index.", end, end),
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.Between(2, 4093),
				},
				PlanModifiers: []planmodifier.Int64{
					detectNullModifier(),
				},
			},
			"inner_vlan": schema.Int64Attribute{
				Description: fmt.Sprintf("The inner VLAN of the %s configuration. Values can range from 2 to 4093. This field cannot be set if the %s VLAN is untagged. Setting to 0 for auto-assignment is not supported in Basic VXC. For MCR and MVE endpoints, inner_vlan is not supported.", end, end),
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.Between(2, 4093),
				},
				PlanModifiers: []planmodifier.Int64{
					detectNullModifier(),
				},
			},
			"vnic_index": schema.Int64Attribute{
				Description: fmt.Sprintf("The network interface index of the %s configuration. Required for MVE connections.", end),
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func isAEndConfig(end string) bool {
	return end == VXC_A_END
}

func isBEndConfig(end string) bool {
	return end == VXC_B_END
}
