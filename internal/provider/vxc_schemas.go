package provider

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

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
	}
	azurePartnerConfigSchema = schema.SingleNestedAttribute{
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
	}
	googlePartnerConfigSchema = schema.SingleNestedAttribute{
		Description: "The Google partner configuration.",
		Optional:    true,
		Attributes: map[string]schema.Attribute{
			"pairing_key": schema.StringAttribute{
				Description: "The pairing key of the partner configuration. Required for Google partner configurations.",
				Required:    true,
			},
		},
	}
	ibmPartnerConfigSchema = schema.SingleNestedAttribute{
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
	}
	oraclePartnerConfigSchema = schema.SingleNestedAttribute{
		Description: "The Oracle partner configuration.",
		Optional:    true,
		Attributes: map[string]schema.Attribute{
			"virtual_circuit_id": schema.StringAttribute{
				Description: "The virtual circuit ID of the partner configuration. Required for Oracle partner configurations.",
				Required:    true,
			},
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
	aEndPartnerConfigSchema = schema.SingleNestedAttribute{
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
	}
)
