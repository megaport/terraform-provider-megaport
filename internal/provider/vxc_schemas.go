package provider

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Schema attribute builder helpers — reduce boilerplate in partner config schema definitions.

func optionalString(desc string) schema.StringAttribute {
	return schema.StringAttribute{Description: desc, Optional: true}
}

func optionalInt64(desc string) schema.Int64Attribute {
	return schema.Int64Attribute{Description: desc, Optional: true}
}

func requiredString(desc string) schema.StringAttribute {
	return schema.StringAttribute{Description: desc, Required: true}
}

func optionalBool(desc string) schema.BoolAttribute {
	return schema.BoolAttribute{Description: desc, Optional: true}
}

var (
	awsPartnerConfigSchema = schema.SingleNestedAttribute{
		Description: "The AWS partner configuration.",
		Optional:    true,
		Attributes: map[string]schema.Attribute{
			"connect_type": schema.StringAttribute{
				Description: `The connection type of the partner configuration. Required for AWS partner configurations - valid values are "AWS" for Virtual Interface or AWSHC for AWS Hosted Connections.`,
				Validators: []validator.String{
					stringvalidator.OneOf("AWS", "AWSHC"),
				},
				Required: true,
			},
			"type": schema.StringAttribute{
				Description: `The type of the AWS Virtual Interface. Required for AWS Virtual Interface Partner Configurations (e.g. if the connect_type is "AWS"). Valid values are "private", "public", or "transit".`,
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("private", "public", "transit"),
				},
			},
			"owner_account": requiredString("The owner AWS account of the partner configuration. Required for AWS partner configurations."),
			"asn":           optionalInt64("The ASN of the partner configuration."),
			"amazon_asn":    optionalInt64("The Amazon ASN of the partner configuration."),
			"auth_key": schema.StringAttribute{
				Description: "The authentication key of the partner configuration.",
				Optional:    true,
				WriteOnly:   true,
			},
			"prefixes":            optionalString("The prefixes of the partner configuration."),
			"customer_ip_address": optionalString("The customer IP address of the partner configuration."),
			"amazon_ip_address":   optionalString("The Amazon IP address of the partner configuration."),
			"name":                requiredString("The name of the partner configuration."),
		},
	}
	azurePartnerConfigSchema = schema.SingleNestedAttribute{
		Description: "The Azure partner configuration.",
		Optional:    true,
		Attributes: map[string]schema.Attribute{
			"service_key": schema.StringAttribute{
				Description: "The service key of the partner configuration. Required for Azure partner configurations.",
				Required:    true,
				WriteOnly:   true,
			},
			"peers": schema.ListNestedAttribute{
				Description: "The peers of the partner configuration. If this is set, the user must delete any Azure resources associated with the VXC on Azure before deleting the VXC.",
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type":             requiredString("The type of the peer."),
						"peer_asn":         optionalString("The peer ASN of the peer."),
						"primary_subnet":   optionalString("The primary subnet of the peer."),
						"secondary_subnet": optionalString("The secondary subnet of the peer."),
						"prefixes":         optionalString("The prefixes of the peer."),
						"shared_key": schema.StringAttribute{
							Description: "The shared key of the peer.",
							Optional:    true,
							Sensitive:   true,
						},
						"vlan": optionalInt64("The VLAN of the peer."),
					},
				},
			},
		},
	}
	googlePartnerConfigSchema = schema.SingleNestedAttribute{
		Description: "The Google partner configuration.",
		Optional:    true,
		Attributes: map[string]schema.Attribute{
			"pairing_key": requiredString("The pairing key of the partner configuration. Required for Google partner configurations."),
		},
	}
	ibmPartnerConfigSchema = schema.SingleNestedAttribute{
		Description: "The IBM partner configuration.",
		Optional:    true,
		Attributes: map[string]schema.Attribute{
			"account_id":   requiredString("Customer's IBM Acount ID. Required for all IBM partner configurations."),
			"customer_asn": optionalInt64("Customer's ASN. Valid ranges: 1-64495, 64999, 131072-4199999999, 4201000000-4201064511. Required unless the connection at the other end of the VXC is an MCR."),
			"name": schema.StringAttribute{
				Description: `Description of this connection for identification purposes. Max 100 characters from 0-9 a-z A-Z / - _ , Defaults to "MEGAPORT"`,
				Optional:    true,
				Validators:  []validator.String{stringvalidator.LengthAtMost(100)},
			},
			"customer_ip_address": optionalString("Customer IPv4 network address including subnet mask. Default is /30 assigned from 169.254.0.0/16."),
			"provider_ip_address": optionalString("Provider IPv4 network address including subnet mask."),
		},
	}
	oraclePartnerConfigSchema = schema.SingleNestedAttribute{
		Description: "The Oracle partner configuration.",
		Optional:    true,
		Attributes: map[string]schema.Attribute{
			"virtual_circuit_id": requiredString("The virtual circuit ID of the partner configuration. Required for Oracle partner configurations."),
		},
	}
	vrouterPartnerConfigSchema = schema.SingleNestedAttribute{
		Description: "The partner configuration of the virtual router configuration.",
		Optional:    true,
		Attributes: map[string]schema.Attribute{
			"interfaces": schema.ListNestedAttribute{
				Description: "The interfaces of the partner configuration.",
				Required:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"ip_mtu": schema.Int64Attribute{
							Description: "The IP MTU of the partner configuration interface. Defaults to 1500.",
							Optional:    true,
							Validators: []validator.Int64{
								int64validator.Between(68, 9074),
							},
						},
						"ip_addresses": schema.ListAttribute{
							Description: "The IP addresses of the partner configuration. Each entry must be in CIDR notation (e.g., \"169.254.100.6/29\").",
							Optional:    true,
							ElementType: types.StringType,
						},
						"ip_routes": schema.ListNestedAttribute{
							Description: "The IP routes of the partner configuration.",
							Optional:    true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"prefix":      optionalString("The prefix of the IP route."),
									"description": optionalString("The description of the IP route."),
									"next_hop":    optionalString("The next hop of the IP route."),
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
								"tx_interval": optionalInt64("The transmit interval of the BFD."),
								"rx_interval": optionalInt64("The receive interval of the BFD."),
								"multiplier":  optionalInt64("The multiplier of the BFD."),
							},
						},
						"vlan": optionalInt64("Inner-VLAN for implicit Q-inQ VXCs. Typically used only for Azure VXCs. The default is no inner-vlan."),
						"bgp_connections": schema.ListNestedAttribute{
							Description: "The BGP connections of the partner configuration interface.",
							Optional:    true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"peer_asn":  optionalInt64("The peer ASN of the BGP connection."),
									"local_asn": optionalInt64("The local ASN of the BGP connection."),
									"peer_type": schema.StringAttribute{
										Description: "Defines the default BGP routing policy for this BGP connection. The default depends on the CSP type of the far end of this VXC.",
										Optional:    true,
										Validators: []validator.String{
											stringvalidator.OneOf("NON_CLOUD", "PRIV_CLOUD", "PUB_CLOUD"),
										},
									},
									"local_ip_address": optionalString("The local IP address of the BGP connection. Must be an IP address without a CIDR mask (e.g., \"169.254.100.6\")."),
									"peer_ip_address":  optionalString("The peer IP address of the BGP connection. Must be an IP address without a CIDR mask (e.g., \"169.254.100.1\")."),
									"password": schema.StringAttribute{
										Description: "The password of the BGP connection.",
										Optional:    true,
										WriteOnly:   true,
									},
									"shutdown":      optionalBool("Whether the BGP connection is shut down."),
									"description":   optionalString("The description of the BGP connection."),
									"med_in":        optionalInt64("The MED in of the BGP connection."),
									"med_out":       optionalInt64("The MED out of the BGP connection."),
									"bfd_enabled":   optionalBool("Whether BFD is enabled for the BGP connection."),
									"export_policy": optionalString("The export policy of the BGP connection."),
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
									"import_whitelist": optionalString("The import whitelist of the BGP connection."),
									"import_blacklist": optionalString("The import blacklist of the BGP connection."),
									"export_whitelist": optionalString("The export whitelist of the BGP connection."),
									"export_blacklist": optionalString("The export blacklist of the BGP connection."),
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
)
