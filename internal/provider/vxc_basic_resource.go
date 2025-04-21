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
		"ordered_vlan":          types.Int64Type,
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
	OrderedVLAN           types.Int64  `tfsdk:"ordered_vlan"`
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

	var aEndOrderedVLAN, bEndOrderedVLAN *int64
	var aEndRequestedProductUID, bEndRequestedProductUID string
	if !orm.AEndConfiguration.IsNull() {
		existingAEnd := &vxcBasicEndConfigurationModel{}
		aEndDiags := orm.AEndConfiguration.As(ctx, existingAEnd, basetypes.ObjectAsOptions{})
		apiDiags = append(apiDiags, aEndDiags...)
		aEndRequestedProductUID = existingAEnd.RequestedProductUID.ValueString()
		if !existingAEnd.OrderedVLAN.IsNull() && !existingAEnd.OrderedVLAN.IsUnknown() {
			vlan := existingAEnd.OrderedVLAN.ValueInt64()
			aEndOrderedVLAN = &vlan
		}
	}

	aEndModel := &vxcBasicEndConfigurationModel{
		RequestedProductUID:   types.StringValue(aEndRequestedProductUID),
		CurrentProductUID:     types.StringValue(v.AEndConfiguration.UID),
		NetworkInterfaceIndex: types.Int64Value(int64(v.AEndConfiguration.NetworkInterfaceIndex)),
	}
	if aEndOrderedVLAN != nil {
		aEndModel.OrderedVLAN = types.Int64Value(*aEndOrderedVLAN)
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
		if !existingBEnd.OrderedVLAN.IsNull() && !existingBEnd.OrderedVLAN.IsUnknown() {
			vlan := existingBEnd.OrderedVLAN.ValueInt64()
			bEndOrderedVLAN = &vlan
		}
		bEndRequestedProductUID = existingBEnd.RequestedProductUID.ValueString()
	}

	bEndModel := &vxcBasicEndConfigurationModel{
		RequestedProductUID:   types.StringValue(bEndRequestedProductUID),
		CurrentProductUID:     types.StringValue(v.BEndConfiguration.UID),
		NetworkInterfaceIndex: types.Int64Value(int64(v.BEndConfiguration.NetworkInterfaceIndex)),
	}
	if bEndOrderedVLAN != nil {
		bEndModel.OrderedVLAN = types.Int64Value(*bEndOrderedVLAN)
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
						// PlanModifiers: []planmodifier.String{
						// 	stringplanmodifier.RequiresReplaceIf(
						// 		func(ctx context.Context, sr planmodifier.StringRequest, rrifr *stringplanmodifier.RequiresReplaceIfFuncResponse) {
						// 			if sr.PlanValue.IsUnknown() {
						// 				rrifr.RequiresReplace = true
						// 			}
						// 		},
						// 		"This modifier will replace the VXC if the new `requested_product_uid` is unknown. This allows the provider to better handle situations when the connected product (Port, MVE, MCR) is being replaced. To avoid replacement, make sure the new `requested_product_uid` is a known value (i.e. an existing product in the state).",
						// 		"This modifier will replace the VXC if the new `requested_product_uid` is unknown. This allows the provider to better handle situations when the connected product (Port, MVE, MCR) is being replaced. To avoid replacement, make sure the new `requested_product_uid` is a known value (i.e. an existing product in the state).",
						// 	),
						// 	stringplanmodifier.UseStateForUnknown(),
						// },
					},
					"current_product_uid": schema.StringAttribute{
						Description: "The current product UID of the A-End configuration. The Megaport API may change a Partner Port from the Requested Port to a different Port in the same location and diversity zone.",
						Optional:    true,
						Computed:    true,
					},
					"ordered_vlan": schema.Int64Attribute{
						Description: "The customer-ordered unique VLAN ID of the A-End configuration. Values can range from 2 to 4093. If this value is set to 0, or not included, the Megaport system allocates a valid VLAN ID to the A-End configuration.  To set this VLAN to untagged, set the VLAN value to -1. Please note that if the A-End ordered_vlan is set to -1, the Megaport API will not allow for the A-End inner_vlan field to be set as the VLAN for this end configuration will be untagged.",
						Optional:    true,
						Computed:    true,
						Validators:  []validator.Int64{int64validator.Between(-1, 4093), int64validator.NoneOf(1)},
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"vlan": schema.Int64Attribute{
						Description: "The current VLAN of the A-End configuration. May be different from the A-End ordered VLAN if the system allocated a different VLAN. Values can range from 2 to 4093. If the A-End ordered_vlan was set to 0, the Megaport system allocated a valid VLAN. If the A-End ordered_vlan was set to -1, the Megaport system will automatically set this value to null.",
						Computed:    true,
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
						// PlanModifiers: []planmodifier.String{
						// 	stringplanmodifier.RequiresReplaceIf(
						// 		func(ctx context.Context, sr planmodifier.StringRequest, rrifr *stringplanmodifier.RequiresReplaceIfFuncResponse) {
						// 			if sr.PlanValue.IsUnknown() {
						// 				rrifr.RequiresReplace = true
						// 			}
						// 		},
						// 		"This modifier will replace the VXC if the new `requested_product_uid` is unknown. This allows the provider to better handle situations when the connected product (Port, MVE, MCR) is being replaced. To avoid replacement, make sure the new `requested_product_uid` is a known value (i.e. an existing product in the state).",
						// 		"This modifier will replace the VXC if the new `requested_product_uid` is unknown. This allows the provider to better handle situations when the connected product (Port, MVE, MCR) is being replaced. To avoid replacement, make sure the new `requested_product_uid` is a known value (i.e. an existing product in the state).",
						// 	),
						// 	stringplanmodifier.UseStateForUnknown(),
						// },
					},
					"current_product_uid": schema.StringAttribute{
						Description: "The current product UID of the B-End configuration. The Megaport API may change a Partner Port on the end configuration from the Requested Port UID to a different Port in the same location and diversity zone.",
						Optional:    true,
						Computed:    true,
					},
					"ordered_vlan": schema.Int64Attribute{
						Description: "The customer-ordered unique VLAN ID of the B-End configuration. Values can range from 2 to 4093. If this value is set to 0, or not included, the Megaport system allocates a valid VLAN ID to the B-End configuration.  To set this VLAN to untagged, set the VLAN value to -1. Please note that if the B-End ordered_vlan is set to -1, the Megaport API will not allow for the B-End inner_vlan field to be set as the VLAN for this end configuration will be untagged.",
						Optional:    true,
						Computed:    true,
						Validators:  []validator.Int64{int64validator.Between(-1, 4093), int64validator.NoneOf(1)},
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"vlan": schema.Int64Attribute{
						Description: "The current VLAN of the B-End configuration. May be different from the B-End ordered VLAN if the system allocated a different VLAN. Values can range from 2 to 4093. If the B-End ordered_vlan was set to 0, the Megaport system allocated a valid VLAN. If the B-End ordered_vlan was set to -1, the Megaport system will automatically set this value to null.",
						Computed:    true,
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
	aEndConfig := &megaport.VXCOrderEndpointConfiguration{
		ProductUID: a.RequestedProductUID.ValueString(),
		VLAN:       int(a.VLAN.ValueInt64()),
	}
	buyReq.PortUID = a.RequestedProductUID.ValueString()

	if !a.OrderedVLAN.IsNull() {
		aEndConfig.VLAN = int(a.OrderedVLAN.ValueInt64())
	} else {
		aEndConfig.VLAN = 0
	}

	if !a.InnerVLAN.IsNull() || !a.NetworkInterfaceIndex.IsNull() {
		vxcOrderMVEConfig := &megaport.VXCOrderMVEConfig{}
		if !a.InnerVLAN.IsNull() {
			vxcOrderMVEConfig.InnerVLAN = int(a.InnerVLAN.ValueInt64())
		}
		if !a.NetworkInterfaceIndex.IsNull() {
			vxcOrderMVEConfig.NetworkInterfaceIndex = int(a.NetworkInterfaceIndex.ValueInt64())
		}
		aEndConfig.VXCOrderMVEConfig = vxcOrderMVEConfig
	}

	if !plan.AEndPartnerConfig.IsNull() {
		var aPartnerConfig vxcPartnerConfigurationModel
		aPartnerDiags := plan.AEndPartnerConfig.As(ctx, &aPartnerConfig, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty:    true,
			UnhandledUnknownAsEmpty: true,
		})
		resp.Diagnostics.Append(aPartnerDiags...)
		switch aPartnerConfig.Partner.ValueString() {
		case "aws":
			if aPartnerConfig.AWSPartnerConfig.IsNull() {
				resp.Diagnostics.AddError(
					"Error creating VXC",
					"Could not create VXC with name "+plan.Name.ValueString()+": AWS Partner configuration is required",
				)
				return
			}
			var awsConfig vxcPartnerConfigAWSModel
			awsDiags := aPartnerConfig.AWSPartnerConfig.As(ctx, &awsConfig, basetypes.ObjectAsOptions{})
			if awsDiags.HasError() {
				resp.Diagnostics.Append(awsDiags...)
				return
			}
			aEndPartnerConfig := megaport.VXCPartnerConfigAWS{
				ConnectType:       awsConfig.ConnectType.ValueString(),
				Type:              awsConfig.Type.ValueString(),
				OwnerAccount:      awsConfig.OwnerAccount.ValueString(),
				ASN:               int(awsConfig.ASN.ValueInt64()),
				AmazonASN:         int(awsConfig.AmazonASN.ValueInt64()),
				AuthKey:           awsConfig.AuthKey.ValueString(),
				Prefixes:          awsConfig.Prefixes.ValueString(),
				CustomerIPAddress: awsConfig.CustomerIPAddress.ValueString(),
				AmazonIPAddress:   awsConfig.AmazonIPAddress.ValueString(),
				ConnectionName:    awsConfig.ConnectionName.ValueString(),
			}
			awsConfigObj, awsDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAWSAttrs, awsConfig)
			resp.Diagnostics.Append(awsDiags...)

			azure := types.ObjectNull(vxcPartnerConfigAzureAttrs)
			google := types.ObjectNull(vxcPartnerConfigGoogleAttrs)
			oracle := types.ObjectNull(vxcPartnerConfigOracleAttrs)
			vrouter := types.ObjectNull(vxcPartnerConfigVrouterAttrs)
			aEndPartner := types.ObjectNull(vxcPartnerConfigAEndAttrs)
			ibmPartner := types.ObjectNull(vxcPartnerConfigIbmAttrs)
			aEndPartnerConfigModel := &vxcPartnerConfigurationModel{
				Partner:              aPartnerConfig.Partner,
				AWSPartnerConfig:     awsConfigObj,
				AzurePartnerConfig:   azure,
				GooglePartnerConfig:  google,
				OraclePartnerConfig:  oracle,
				VrouterPartnerConfig: vrouter,
				PartnerAEndConfig:    aEndPartner,
				IBMPartnerConfig:     ibmPartner,
			}

			partnerConfigObj, partnerDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, aEndPartnerConfigModel)
			resp.Diagnostics.Append(partnerDiags...)
			plan.AEndPartnerConfig = partnerConfigObj
			aEndConfig.PartnerConfig = aEndPartnerConfig
		case "azure":
			if aPartnerConfig.AzurePartnerConfig.IsNull() {
				resp.Diagnostics.AddError(
					"Error creating VXC",
					"Could not create VXC with name "+plan.Name.ValueString()+": Azure Partner configuration is required",
				)
				return
			}
			var azureConfig vxcPartnerConfigAzureModel
			azureDiags := aPartnerConfig.AzurePartnerConfig.As(ctx, &azureConfig, basetypes.ObjectAsOptions{})
			if azureDiags.HasError() {
				resp.Diagnostics.Append(azureDiags...)
				return
			}
			partnerPortReq := &megaport.ListPartnerPortsRequest{
				Key:     azureConfig.ServiceKey.ValueString(),
				Partner: "AZURE",
			}
			partnerPortRes, err := r.client.VXCService.ListPartnerPorts(ctx, partnerPortReq)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error creating VXC",
					fmt.Sprintf("Could not create %s, there was an error looking up partner ports: %s", plan.Name.ValueString(), err.Error()),
				)
				return
			}
			// find primary or secondary port
			for _, port := range partnerPortRes.Data.Megaports {
				p := &port
				if p.Type == azureConfig.PortChoice.ValueString() {
					aEndConfig.ProductUID = p.ProductUID
				}
			}
			if aEndConfig.ProductUID == "" {
				resp.Diagnostics.AddError(
					"Error creating VXC",
					fmt.Sprintf("Could not find azure port with type: %s", azureConfig.PortChoice.ValueString()),
				)
				return
			}

			aEndPartnerConfig := megaport.VXCPartnerConfigAzure{
				ConnectType: "AZURE",
				ServiceKey:  azureConfig.ServiceKey.ValueString(),
			}

			azurePeerModels := []partnerOrderAzurePeeringConfigModel{}
			azurePeerDiags := azureConfig.Peers.ElementsAs(ctx, &azurePeerModels, false)
			resp.Diagnostics.Append(azurePeerDiags...)
			if len(azurePeerModels) > 0 {
				aEndPartnerConfig.Peers = []megaport.PartnerOrderAzurePeeringConfig{}
				for _, peer := range azurePeerModels {
					peeringConfig := megaport.PartnerOrderAzurePeeringConfig{
						Type:            peer.Type.ValueString(),
						PeerASN:         peer.PeerASN.ValueString(),
						PrimarySubnet:   peer.PrimarySubnet.ValueString(),
						SecondarySubnet: peer.SecondarySubnet.ValueString(),
						VLAN:            int(peer.VLAN.ValueInt64()),
					}
					if !peer.Prefixes.IsNull() {
						peeringConfig.Prefixes = peer.Prefixes.ValueString()
					}
					if !peer.SharedKey.IsNull() {
						peeringConfig.SharedKey = peer.SharedKey.ValueString()
					}
					aEndPartnerConfig.Peers = append(aEndPartnerConfig.Peers, peeringConfig)
				}
			}

			azureConfigObj, azureDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAzureAttrs, azureConfig)
			resp.Diagnostics.Append(azureDiags...)

			aws := types.ObjectNull(vxcPartnerConfigAWSAttrs)
			google := types.ObjectNull(vxcPartnerConfigGoogleAttrs)
			oracle := types.ObjectNull(vxcPartnerConfigOracleAttrs)
			vrouter := types.ObjectNull(vxcPartnerConfigVrouterAttrs)
			aEndPartner := types.ObjectNull(vxcPartnerConfigAEndAttrs)
			ibmPartner := types.ObjectNull(vxcPartnerConfigIbmAttrs)
			aEndPartnerConfigModel := &vxcPartnerConfigurationModel{
				Partner:              aPartnerConfig.Partner,
				AWSPartnerConfig:     aws,
				AzurePartnerConfig:   azureConfigObj,
				GooglePartnerConfig:  google,
				OraclePartnerConfig:  oracle,
				VrouterPartnerConfig: vrouter,
				PartnerAEndConfig:    aEndPartner,
				IBMPartnerConfig:     ibmPartner,
			}

			partnerConfigObj, partnerDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, aEndPartnerConfigModel)
			resp.Diagnostics.Append(partnerDiags...)
			plan.AEndPartnerConfig = partnerConfigObj
			aEndConfig.PartnerConfig = aEndPartnerConfig
		case "google":
			if aPartnerConfig.GooglePartnerConfig.IsNull() {
				resp.Diagnostics.AddError(
					"Error creating VXC",
					"Could not create VXC with name "+plan.Name.ValueString()+": Google Partner configuration is required",
				)
				return
			}
			var googleConfig vxcPartnerConfigGoogleModel
			googleDiags := aPartnerConfig.GooglePartnerConfig.As(ctx, &googleConfig, basetypes.ObjectAsOptions{})
			if googleDiags.HasError() {
				resp.Diagnostics.Append(googleDiags...)
				return
			}
			aEndPartnerConfig := megaport.VXCPartnerConfigGoogle{
				ConnectType: "GOOGLE",
				PairingKey:  googleConfig.PairingKey.ValueString(),
			}
			googleConfigObj, googleDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigGoogleAttrs, googleConfig)
			resp.Diagnostics.Append(googleDiags...)

			partnerPortReq := &megaport.LookupPartnerPortsRequest{
				Key:       googleConfig.PairingKey.ValueString(),
				PortSpeed: int(plan.RateLimit.ValueInt64()),
				Partner:   "GOOGLE",
			}
			partnerPortReq.ProductID = a.RequestedProductUID.ValueString()
			partnerPortRes, err := r.client.VXCService.LookupPartnerPorts(ctx, partnerPortReq)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error creating VXC",
					fmt.Sprintf("Could not create %s, there was an error looking up partner ports: %s", plan.Name.ValueString(), err.Error()),
				)
				return
			}
			aEndConfig.ProductUID = partnerPortRes.ProductUID

			aws := types.ObjectNull(vxcPartnerConfigAWSAttrs)
			azure := types.ObjectNull(vxcPartnerConfigAzureAttrs)
			oracle := types.ObjectNull(vxcPartnerConfigOracleAttrs)
			vrouter := types.ObjectNull(vxcPartnerConfigVrouterAttrs)
			aEndPartner := types.ObjectNull(vxcPartnerConfigAEndAttrs)
			ibmPartner := types.ObjectNull(vxcPartnerConfigIbmAttrs)
			aEndPartnerConfigModel := &vxcPartnerConfigurationModel{
				Partner:              aPartnerConfig.Partner,
				AWSPartnerConfig:     aws,
				AzurePartnerConfig:   azure,
				GooglePartnerConfig:  googleConfigObj,
				OraclePartnerConfig:  oracle,
				VrouterPartnerConfig: vrouter,
				IBMPartnerConfig:     ibmPartner,
				PartnerAEndConfig:    aEndPartner,
			}

			partnerConfigObj, partnerDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, aEndPartnerConfigModel)
			resp.Diagnostics.Append(partnerDiags...)
			plan.AEndPartnerConfig = partnerConfigObj

			aEndConfig.PartnerConfig = aEndPartnerConfig
		case "oracle":
			if aPartnerConfig.OraclePartnerConfig.IsNull() {
				resp.Diagnostics.AddError(
					"Error creating VXC",
					"Could not create VXC with name "+plan.Name.ValueString()+": Oracle Partner configuration is required",
				)
				return
			}
			var oracleConfig vxcPartnerConfigOracleModel
			oracleDiags := aPartnerConfig.OraclePartnerConfig.As(ctx, &oracleConfig, basetypes.ObjectAsOptions{})
			if oracleDiags.HasError() {
				resp.Diagnostics.Append(oracleDiags...)
				return
			}
			aEndPartnerConfig := &megaport.VXCPartnerConfigOracle{
				ConnectType:      "ORACLE",
				VirtualCircuitId: oracleConfig.VirtualCircuitId.ValueString(),
			}

			partnerPortReq := &megaport.LookupPartnerPortsRequest{
				Key:       oracleConfig.VirtualCircuitId.ValueString(),
				PortSpeed: int(plan.RateLimit.ValueInt64()),
				Partner:   "ORACLE",
			}
			partnerPortReq.ProductID = a.RequestedProductUID.ValueString()

			partnerPortRes, err := r.client.VXCService.LookupPartnerPorts(ctx, partnerPortReq)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error creating VXC",
					fmt.Sprintf("Could not create %s, there was an error looking up partner ports: %s", plan.Name.ValueString(), err.Error()),
				)
				return
			}
			aEndConfig.ProductUID = partnerPortRes.ProductUID

			oracleConfigObj, oracleDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigOracleAttrs, oracleConfig)
			resp.Diagnostics.Append(oracleDiags...)

			aws := types.ObjectNull(vxcPartnerConfigAWSAttrs)
			azure := types.ObjectNull(vxcPartnerConfigAzureAttrs)
			google := types.ObjectNull(vxcPartnerConfigGoogleAttrs)
			vrouter := types.ObjectNull(vxcPartnerConfigVrouterAttrs)
			aEndPartner := types.ObjectNull(vxcPartnerConfigAEndAttrs)
			ibmPartner := types.ObjectNull(vxcPartnerConfigIbmAttrs)
			aEndPartnerConfigModel := &vxcPartnerConfigurationModel{
				Partner:              aPartnerConfig.Partner,
				AWSPartnerConfig:     aws,
				AzurePartnerConfig:   azure,
				GooglePartnerConfig:  google,
				OraclePartnerConfig:  oracleConfigObj,
				VrouterPartnerConfig: vrouter,
				PartnerAEndConfig:    aEndPartner,
				IBMPartnerConfig:     ibmPartner,
			}

			partnerConfigObj, partnerDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, aEndPartnerConfigModel)
			resp.Diagnostics.Append(partnerDiags...)
			plan.AEndPartnerConfig = partnerConfigObj
			aEndConfig.PartnerConfig = aEndPartnerConfig
		case "ibm":
			if aPartnerConfig.IBMPartnerConfig.IsNull() {
				resp.Diagnostics.AddError(
					"Error creating VXC",
					"Could not create VXC with name "+plan.Name.ValueString()+": IBM Partner configuration is required",
				)
				return
			}
			var ibmConfig vxcPartnerConfigIbmModel
			ibmDiags := aPartnerConfig.IBMPartnerConfig.As(ctx, &ibmConfig, basetypes.ObjectAsOptions{})
			resp.Diagnostics.Append(ibmDiags...)
			if resp.Diagnostics.HasError() {
				return
			}
			aEndPartnerConfig := megaport.VXCPartnerConfigIBM{
				ConnectType:       "IBM",
				AccountID:         ibmConfig.AccountID.ValueString(),
				CustomerASN:       int(ibmConfig.CustomerASN.ValueInt64()),
				Name:              ibmConfig.Name.ValueString(),
				CustomerIPAddress: ibmConfig.CustomerIPAddress.ValueString(),
				ProviderIPAddress: ibmConfig.ProviderIPAddress.ValueString(),
			}
			aws := types.ObjectNull(vxcPartnerConfigAWSAttrs)
			azure := types.ObjectNull(vxcPartnerConfigAzureAttrs)
			google := types.ObjectNull(vxcPartnerConfigGoogleAttrs)
			oracle := types.ObjectNull(vxcPartnerConfigOracleAttrs)
			vrouter := types.ObjectNull(vxcPartnerConfigVrouterAttrs)
			aEndPartner := types.ObjectNull(vxcPartnerConfigAEndAttrs)
			aEndPartnerConfigModel := &vxcPartnerConfigurationModel{
				Partner:              aPartnerConfig.Partner,
				AWSPartnerConfig:     aws,
				AzurePartnerConfig:   azure,
				GooglePartnerConfig:  google,
				OraclePartnerConfig:  oracle,
				VrouterPartnerConfig: vrouter,
				PartnerAEndConfig:    aEndPartner,
				IBMPartnerConfig:     aPartnerConfig.IBMPartnerConfig,
			}
			partnerConfigObj, partnerDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, aEndPartnerConfigModel)
			resp.Diagnostics.Append(partnerDiags...)
			plan.AEndPartnerConfig = partnerConfigObj
			aEndConfig.PartnerConfig = aEndPartnerConfig
		case "vrouter":
			if aPartnerConfig.VrouterPartnerConfig.IsNull() {
				resp.Diagnostics.AddError(
					"Error creating VXC",
					"Could not create VXC with name "+plan.Name.ValueString()+": Virtual router configuration is required",
				)
				return
			}
			var partnerConfigAEnd vxcPartnerConfigVrouterModel
			aEndDiags := aPartnerConfig.VrouterPartnerConfig.As(ctx, &partnerConfigAEnd, basetypes.ObjectAsOptions{})
			if aEndDiags.HasError() {
				resp.Diagnostics.Append(aEndDiags...)
				return
			}
			prefixFilterListRes, err := r.client.MCRService.ListMCRPrefixFilterLists(ctx, a.RequestedProductUID.ValueString())
			if err != nil {
				resp.Diagnostics.AddError(
					"Error creating VXC",
					"Could not create VXC with name "+plan.Name.ValueString()+": "+err.Error(),
				)
				return
			}

			aEndMegaportConfig := megaport.VXCOrderVrouterPartnerConfig{}
			ifaceModels := []*vxcPartnerConfigInterfaceModel{}
			ifaceDiags := partnerConfigAEnd.Interfaces.ElementsAs(ctx, &ifaceModels, false)
			resp.Diagnostics = append(resp.Diagnostics, ifaceDiags...)
			for _, iface := range ifaceModels {
				toAppend := megaport.PartnerConfigInterface{}
				if !iface.IPAddresses.IsNull() {
					ipAddresses := []string{}
					ipDiags := iface.IPAddresses.ElementsAs(ctx, &ipAddresses, true)
					resp.Diagnostics = append(resp.Diagnostics, ipDiags...)
					toAppend.IpAddresses = ipAddresses
				}
				if !iface.IPRoutes.IsNull() {
					ipRoutes := []*ipRouteModel{}
					ipRouteDiags := iface.IPRoutes.ElementsAs(ctx, &ipRoutes, true)
					resp.Diagnostics = append(resp.Diagnostics, ipRouteDiags...)
					for _, ipRoute := range ipRoutes {
						toAppend.IpRoutes = append(toAppend.IpRoutes, megaport.IpRoute{
							Prefix:      ipRoute.Prefix.ValueString(),
							Description: ipRoute.Description.ValueString(),
							NextHop:     ipRoute.NextHop.ValueString(),
						})
					}
				}
				if !iface.NatIPAddresses.IsNull() {
					natIPAddresses := []string{}
					natDiags := iface.NatIPAddresses.ElementsAs(ctx, &natIPAddresses, true)
					resp.Diagnostics = append(resp.Diagnostics, natDiags...)
					toAppend.NatIpAddresses = natIPAddresses
				}
				if !iface.Bfd.IsNull() {
					bfd := &bfdConfigModel{}
					bfdDiags := iface.Bfd.As(ctx, bfd, basetypes.ObjectAsOptions{})
					resp.Diagnostics = append(resp.Diagnostics, bfdDiags...)
					toAppend.Bfd = megaport.BfdConfig{
						TxInterval: int(bfd.TxInterval.ValueInt64()),
						RxInterval: int(bfd.RxInterval.ValueInt64()),
						Multiplier: int(bfd.Multiplier.ValueInt64()),
					}
				}
				if !iface.VLAN.IsNull() {
					toAppend.VLAN = int(iface.VLAN.ValueInt64())
				}
				if !iface.BgpConnections.IsNull() {
					bgpConnections := []*bgpConnectionConfigModel{}
					bgpDiags := iface.BgpConnections.ElementsAs(ctx, &bgpConnections, false)
					resp.Diagnostics = append(resp.Diagnostics, bgpDiags...)
					for _, bgpConnection := range bgpConnections {
						bgpToAppend := megaport.BgpConnectionConfig{
							PeerAsn:            int(bgpConnection.PeerAsn.ValueInt64()),
							LocalIpAddress:     bgpConnection.LocalIPAddress.ValueString(),
							PeerIpAddress:      bgpConnection.PeerIPAddress.ValueString(),
							Password:           bgpConnection.Password.ValueString(),
							Shutdown:           bgpConnection.Shutdown.ValueBool(),
							Description:        bgpConnection.Description.ValueString(),
							MedIn:              int(bgpConnection.MedIn.ValueInt64()),
							MedOut:             int(bgpConnection.MedOut.ValueInt64()),
							BfdEnabled:         bgpConnection.BfdEnabled.ValueBool(),
							ExportPolicy:       bgpConnection.ExportPolicy.ValueString(),
							AsPathPrependCount: int(bgpConnection.AsPathPrependCount.ValueInt64()),
							PeerType:           bgpConnection.PeerType.ValueString(),
						}
						if !bgpConnection.LocalAsn.IsNull() {
							bgpToAppend.LocalAsn = megaport.PtrTo(int(bgpConnection.LocalAsn.ValueInt64()))
						}
						if !bgpConnection.ImportWhitelist.IsNull() {
							for _, prefixFilterList := range prefixFilterListRes {
								if prefixFilterList.Description == bgpConnection.ImportWhitelist.ValueString() {
									bgpToAppend.ImportWhitelist = prefixFilterList.Id
								}
							}
						}
						if !bgpConnection.ImportBlacklist.IsNull() {
							for _, prefixFilterList := range prefixFilterListRes {
								if prefixFilterList.Description == bgpConnection.ImportBlacklist.ValueString() {
									bgpToAppend.ImportBlacklist = prefixFilterList.Id
								}
							}
						}
						if !bgpConnection.ExportWhitelist.IsNull() {
							for _, prefixFilterList := range prefixFilterListRes {
								if prefixFilterList.Description == bgpConnection.ExportWhitelist.ValueString() {
									bgpToAppend.ExportWhitelist = prefixFilterList.Id
								}
							}
						}
						if !bgpConnection.ExportBlacklist.IsNull() {
							for _, prefixFilterList := range prefixFilterListRes {
								if prefixFilterList.Description == bgpConnection.ExportBlacklist.ValueString() {
									bgpToAppend.ExportBlacklist = prefixFilterList.Id
								}
							}
						}
						if !bgpConnection.PermitExportTo.IsNull() {
							permitExportTo := []string{}
							permitDiags := bgpConnection.PermitExportTo.ElementsAs(ctx, &permitExportTo, true)
							resp.Diagnostics = append(resp.Diagnostics, permitDiags...)
							bgpToAppend.PermitExportTo = permitExportTo
							bgpToAppend.PermitExportTo = permitExportTo
						}
						if !bgpConnection.DenyExportTo.IsNull() {
							denyExportTo := []string{}
							denyDiags := bgpConnection.DenyExportTo.ElementsAs(ctx, &denyExportTo, true)
							resp.Diagnostics = append(resp.Diagnostics, denyDiags...)
							bgpToAppend.DenyExportTo = denyExportTo
						}
						toAppend.BgpConnections = append(toAppend.BgpConnections, bgpToAppend)
					}
				}
				aEndMegaportConfig.Interfaces = append(aEndMegaportConfig.Interfaces, toAppend)
			}
			vRouterConfigObj, aEndDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigVrouterAttrs, partnerConfigAEnd)
			resp.Diagnostics.Append(aEndDiags...)
			aws := types.ObjectNull(vxcPartnerConfigAWSAttrs)
			azure := types.ObjectNull(vxcPartnerConfigAzureAttrs)
			google := types.ObjectNull(vxcPartnerConfigGoogleAttrs)
			oracle := types.ObjectNull(vxcPartnerConfigOracleAttrs)
			aEndPartner := types.ObjectNull(vxcPartnerConfigAEndAttrs)
			ibmPartner := types.ObjectNull(vxcPartnerConfigIbmAttrs)
			aEndPartnerConfigModel := &vxcPartnerConfigurationModel{
				Partner:              aPartnerConfig.Partner,
				AWSPartnerConfig:     aws,
				AzurePartnerConfig:   azure,
				GooglePartnerConfig:  google,
				OraclePartnerConfig:  oracle,
				VrouterPartnerConfig: vRouterConfigObj,
				PartnerAEndConfig:    aEndPartner,
				IBMPartnerConfig:     ibmPartner,
			}
			aEndPartnerConfigObj, partnerDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, aEndPartnerConfigModel)
			resp.Diagnostics.Append(partnerDiags...)
			plan.AEndPartnerConfig = aEndPartnerConfigObj
			aEndConfig.PartnerConfig = aEndMegaportConfig
		case "a-end":
			if aPartnerConfig.PartnerAEndConfig.IsNull() {
				resp.Diagnostics.AddError(
					"Error creating VXC",
					"Could not create VXC with name "+plan.Name.ValueString()+": A-End Partner configuration is required",
				)
				return
			}
			var partnerConfigAEnd vxcPartnerConfigAEndModel
			aEndDiags := aPartnerConfig.PartnerAEndConfig.As(ctx, &partnerConfigAEnd, basetypes.ObjectAsOptions{})
			if aEndDiags.HasError() {
				resp.Diagnostics.Append(aEndDiags...)
				return
			}
			prefixFilterListRes, err := r.client.MCRService.ListMCRPrefixFilterLists(ctx, a.RequestedProductUID.ValueString())
			if err != nil {
				resp.Diagnostics.AddError(
					"Error creating VXC",
					"Could not create VXC with name "+plan.Name.ValueString()+": "+err.Error(),
				)
				return
			}

			aEndMegaportConfig := megaport.VXCOrderVrouterPartnerConfig{}
			ifaceModels := []*vxcPartnerConfigInterfaceModel{}
			ifaceDiags := partnerConfigAEnd.Interfaces.ElementsAs(ctx, &ifaceModels, true)
			resp.Diagnostics = append(resp.Diagnostics, ifaceDiags...)
			for _, iface := range ifaceModels {
				toAppend := megaport.PartnerConfigInterface{}
				if !iface.IPAddresses.IsNull() {
					ipAddresses := []string{}
					ipDiags := iface.IPAddresses.ElementsAs(ctx, &ipAddresses, true)
					resp.Diagnostics = append(resp.Diagnostics, ipDiags...)
					toAppend.IpAddresses = ipAddresses
				}
				if !iface.IPRoutes.IsNull() {
					ipRoutes := []*ipRouteModel{}
					ipRouteDiags := iface.IPRoutes.ElementsAs(ctx, &ipRoutes, true)
					resp.Diagnostics = append(resp.Diagnostics, ipRouteDiags...)
					for _, ipRoute := range ipRoutes {
						toAppend.IpRoutes = append(toAppend.IpRoutes, megaport.IpRoute{
							Prefix:      ipRoute.Prefix.ValueString(),
							Description: ipRoute.Description.ValueString(),
							NextHop:     ipRoute.NextHop.ValueString(),
						})
					}
				}
				if !iface.NatIPAddresses.IsNull() {
					natIPAddresses := []string{}
					natDiags := iface.NatIPAddresses.ElementsAs(ctx, &natIPAddresses, true)
					resp.Diagnostics = append(resp.Diagnostics, natDiags...)
					toAppend.NatIpAddresses = natIPAddresses
				}
				if !iface.Bfd.IsNull() {
					bfd := &bfdConfigModel{}
					bfdDiags := iface.Bfd.As(ctx, bfd, basetypes.ObjectAsOptions{})
					resp.Diagnostics = append(resp.Diagnostics, bfdDiags...)
					toAppend.Bfd = megaport.BfdConfig{
						TxInterval: int(bfd.TxInterval.ValueInt64()),
						RxInterval: int(bfd.RxInterval.ValueInt64()),
						Multiplier: int(bfd.Multiplier.ValueInt64()),
					}
				}
				if !iface.BgpConnections.IsNull() {
					bgpConnections := []*bgpConnectionConfigModel{}
					bgpDiags := iface.BgpConnections.ElementsAs(ctx, &bgpConnections, false)
					resp.Diagnostics = append(resp.Diagnostics, bgpDiags...)
					for _, bgpConnection := range bgpConnections {
						bgpToAppend := megaport.BgpConnectionConfig{
							PeerAsn:            int(bgpConnection.PeerAsn.ValueInt64()),
							LocalIpAddress:     bgpConnection.LocalIPAddress.ValueString(),
							PeerIpAddress:      bgpConnection.PeerIPAddress.ValueString(),
							Password:           bgpConnection.Password.ValueString(),
							Shutdown:           bgpConnection.Shutdown.ValueBool(),
							Description:        bgpConnection.Description.ValueString(),
							MedIn:              int(bgpConnection.MedIn.ValueInt64()),
							MedOut:             int(bgpConnection.MedOut.ValueInt64()),
							BfdEnabled:         bgpConnection.BfdEnabled.ValueBool(),
							ExportPolicy:       bgpConnection.ExportPolicy.ValueString(),
							AsPathPrependCount: int(bgpConnection.AsPathPrependCount.ValueInt64()),
						}
						if !bgpConnection.LocalAsn.IsNull() {
							bgpToAppend.LocalAsn = megaport.PtrTo(int(bgpConnection.LocalAsn.ValueInt64()))
						}
						if !bgpConnection.ImportWhitelist.IsNull() {
							for _, prefixFilterList := range prefixFilterListRes {
								if prefixFilterList.Description == bgpConnection.ImportWhitelist.ValueString() {
									bgpToAppend.ImportWhitelist = prefixFilterList.Id
								}
							}
						}
						if !bgpConnection.ImportBlacklist.IsNull() {
							for _, prefixFilterList := range prefixFilterListRes {
								if prefixFilterList.Description == bgpConnection.ImportBlacklist.ValueString() {
									bgpToAppend.ImportBlacklist = prefixFilterList.Id
								}
							}
						}
						if !bgpConnection.ExportWhitelist.IsNull() {
							for _, prefixFilterList := range prefixFilterListRes {
								if prefixFilterList.Description == bgpConnection.ExportWhitelist.ValueString() {
									bgpToAppend.ExportWhitelist = prefixFilterList.Id
								}
							}
						}
						if !bgpConnection.ExportBlacklist.IsNull() {
							for _, prefixFilterList := range prefixFilterListRes {
								if prefixFilterList.Description == bgpConnection.ExportBlacklist.ValueString() {
									bgpToAppend.ExportBlacklist = prefixFilterList.Id
								}
							}
						}
						if !bgpConnection.PermitExportTo.IsNull() {
							permitExportTo := []string{}
							permitDiags := bgpConnection.PermitExportTo.ElementsAs(ctx, &permitExportTo, true)
							resp.Diagnostics = append(resp.Diagnostics, permitDiags...)
							bgpToAppend.PermitExportTo = permitExportTo
							bgpToAppend.PermitExportTo = permitExportTo
						}
						if !bgpConnection.DenyExportTo.IsNull() {
							denyExportTo := []string{}
							denyDiags := bgpConnection.DenyExportTo.ElementsAs(ctx, &denyExportTo, true)
							resp.Diagnostics = append(resp.Diagnostics, denyDiags...)
							bgpToAppend.DenyExportTo = denyExportTo
						}
						toAppend.BgpConnections = append(toAppend.BgpConnections, bgpToAppend)
					}
				}
				aEndMegaportConfig.Interfaces = append(aEndMegaportConfig.Interfaces, toAppend)
			}
			aEndConfigObj, aEndDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAEndAttrs, partnerConfigAEnd)
			resp.Diagnostics.Append(aEndDiags...)
			aws := types.ObjectNull(vxcPartnerConfigAWSAttrs)
			azure := types.ObjectNull(vxcPartnerConfigAzureAttrs)
			google := types.ObjectNull(vxcPartnerConfigGoogleAttrs)
			oracle := types.ObjectNull(vxcPartnerConfigOracleAttrs)
			vrouter := types.ObjectNull(vxcPartnerConfigVrouterAttrs)
			ibmPartner := types.ObjectNull(vxcPartnerConfigIbmAttrs)
			aEndPartnerConfigModel := &vxcPartnerConfigurationModel{
				Partner:              aPartnerConfig.Partner,
				AWSPartnerConfig:     aws,
				AzurePartnerConfig:   azure,
				GooglePartnerConfig:  google,
				OraclePartnerConfig:  oracle,
				PartnerAEndConfig:    aEndConfigObj,
				VrouterPartnerConfig: vrouter,
				IBMPartnerConfig:     ibmPartner,
			}
			aEndPartnerConfigObj, partnerDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, aEndPartnerConfigModel)
			resp.Diagnostics.Append(partnerDiags...)
			plan.AEndPartnerConfig = aEndPartnerConfigObj
			aEndConfig.PartnerConfig = aEndMegaportConfig
		case "transit":
			aEndPartnerConfig := &megaport.VXCPartnerConfigTransit{
				ConnectType: "TRANSIT",
			}
			aws := types.ObjectNull(vxcPartnerConfigAWSAttrs)
			azure := types.ObjectNull(vxcPartnerConfigAzureAttrs)
			google := types.ObjectNull(vxcPartnerConfigGoogleAttrs)
			oracle := types.ObjectNull(vxcPartnerConfigOracleAttrs)
			vrouter := types.ObjectNull(vxcPartnerConfigVrouterAttrs)
			aEndPartner := types.ObjectNull(vxcPartnerConfigAEndAttrs)
			aEndPartnerConfigModel := &vxcPartnerConfigurationModel{
				Partner:              aPartnerConfig.Partner,
				AWSPartnerConfig:     aws,
				AzurePartnerConfig:   azure,
				GooglePartnerConfig:  google,
				OraclePartnerConfig:  oracle,
				VrouterPartnerConfig: vrouter,
				PartnerAEndConfig:    aEndPartner,
			}

			partnerConfigObj, partnerDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, aEndPartnerConfigModel)
			resp.Diagnostics.Append(partnerDiags...)
			plan.AEndPartnerConfig = partnerConfigObj
			aEndConfig.PartnerConfig = aEndPartnerConfig
		default:
			resp.Diagnostics.AddError(
				"Error creating VXC",
				"Could not create VXC with name "+plan.Name.ValueString()+": Partner configuration not supported",
			)
			return
		}
	}

	buyReq.AEndConfiguration = *aEndConfig

	var b vxcBasicEndConfigurationModel
	bEndDiags := bEndObj.As(ctx, &b, basetypes.ObjectAsOptions{})
	if bEndDiags.HasError() {
		resp.Diagnostics.Append(bEndDiags...)
		return
	}
	bEndConfig := &megaport.VXCOrderEndpointConfiguration{
		ProductUID: b.RequestedProductUID.ValueString(),
		VLAN:       int(b.VLAN.ValueInt64()),
	}
	if !b.OrderedVLAN.IsNull() {
		bEndConfig.VLAN = int(b.OrderedVLAN.ValueInt64())
	} else {
		bEndConfig.VLAN = 0
	}
	if !b.InnerVLAN.IsNull() || !b.NetworkInterfaceIndex.IsNull() {
		vxcOrderMVEConfig := &megaport.VXCOrderMVEConfig{}
		if !b.InnerVLAN.IsNull() {
			vxcOrderMVEConfig.InnerVLAN = int(b.InnerVLAN.ValueInt64())
		}
		if !b.NetworkInterfaceIndex.IsNull() {
			vxcOrderMVEConfig.NetworkInterfaceIndex = int(b.NetworkInterfaceIndex.ValueInt64())
		}
		bEndConfig.VXCOrderMVEConfig = vxcOrderMVEConfig
	}
	if !plan.BEndPartnerConfig.IsNull() {
		var bPartnerConfig vxcPartnerConfigurationModel
		bPartnerDiags := plan.BEndPartnerConfig.As(ctx, &bPartnerConfig, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(bPartnerDiags...)
		switch bPartnerConfig.Partner.ValueString() {
		case "aws":
			if bPartnerConfig.AWSPartnerConfig.IsNull() {
				resp.Diagnostics.AddError(
					"Error creating VXC",
					"Could not create VXC with name "+plan.Name.ValueString()+": AWS Partner configuration is required",
				)
				return
			}
			var awsConfig vxcPartnerConfigAWSModel
			awsDiags := bPartnerConfig.AWSPartnerConfig.As(ctx, &awsConfig, basetypes.ObjectAsOptions{})
			if awsDiags.HasError() {
				resp.Diagnostics.Append(awsDiags...)
				return
			}
			bEndPartnerConfig := megaport.VXCPartnerConfigAWS{
				ConnectType:       awsConfig.ConnectType.ValueString(),
				Type:              awsConfig.Type.ValueString(),
				OwnerAccount:      awsConfig.OwnerAccount.ValueString(),
				ASN:               int(awsConfig.ASN.ValueInt64()),
				AmazonASN:         int(awsConfig.AmazonASN.ValueInt64()),
				AuthKey:           awsConfig.AuthKey.ValueString(),
				Prefixes:          awsConfig.Prefixes.ValueString(),
				CustomerIPAddress: awsConfig.CustomerIPAddress.ValueString(),
				AmazonIPAddress:   awsConfig.AmazonIPAddress.ValueString(),
				ConnectionName:    awsConfig.ConnectionName.ValueString(),
			}

			awsConfigObj, awsDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAWSAttrs, awsConfig)
			resp.Diagnostics.Append(awsDiags...)

			azure := types.ObjectNull(vxcPartnerConfigAzureAttrs)
			google := types.ObjectNull(vxcPartnerConfigGoogleAttrs)
			oracle := types.ObjectNull(vxcPartnerConfigOracleAttrs)
			vrouter := types.ObjectNull(vxcPartnerConfigVrouterAttrs)
			aEndPartner := types.ObjectNull(vxcPartnerConfigAEndAttrs)
			ibmPartner := types.ObjectNull(vxcPartnerConfigIbmAttrs)
			bEndPartnerConfigModel := &vxcPartnerConfigurationModel{
				Partner:              bPartnerConfig.Partner,
				AWSPartnerConfig:     awsConfigObj,
				AzurePartnerConfig:   azure,
				GooglePartnerConfig:  google,
				OraclePartnerConfig:  oracle,
				IBMPartnerConfig:     ibmPartner,
				VrouterPartnerConfig: vrouter,
				PartnerAEndConfig:    aEndPartner,
			}

			partnerConfigObj, partnerDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, bEndPartnerConfigModel)
			resp.Diagnostics.Append(partnerDiags...)
			plan.BEndPartnerConfig = partnerConfigObj
			bEndConfig.PartnerConfig = bEndPartnerConfig
		case "azure":
			if bPartnerConfig.AzurePartnerConfig.IsNull() {
				resp.Diagnostics.AddError(
					"Error creating VXC",
					"Could not create VXC with name "+plan.Name.ValueString()+": Azure Partner configuration is required",
				)
				return
			}
			var azureConfig vxcPartnerConfigAzureModel
			azureDiags := bPartnerConfig.AzurePartnerConfig.As(ctx, &azureConfig, basetypes.ObjectAsOptions{})
			if azureDiags.HasError() {
				resp.Diagnostics.Append(azureDiags...)
				return
			}
			bEndPartnerConfig := megaport.VXCPartnerConfigAzure{
				ConnectType: "AZURE",
				ServiceKey:  azureConfig.ServiceKey.ValueString(),
			}

			azurePeerModels := []partnerOrderAzurePeeringConfigModel{}
			azurePeerDiags := azureConfig.Peers.ElementsAs(ctx, &azurePeerModels, false)
			resp.Diagnostics.Append(azurePeerDiags...)
			if len(azurePeerModels) > 0 {
				bEndPartnerConfig.Peers = []megaport.PartnerOrderAzurePeeringConfig{}
				for _, peer := range azurePeerModels {
					peeringConfig := megaport.PartnerOrderAzurePeeringConfig{
						Type:            peer.Type.ValueString(),
						PeerASN:         peer.PeerASN.ValueString(),
						PrimarySubnet:   peer.PrimarySubnet.ValueString(),
						SecondarySubnet: peer.SecondarySubnet.ValueString(),
						VLAN:            int(peer.VLAN.ValueInt64()),
					}
					if !peer.Prefixes.IsNull() {
						peeringConfig.Prefixes = peer.Prefixes.ValueString()
					}
					if !peer.SharedKey.IsNull() {
						peeringConfig.SharedKey = peer.SharedKey.ValueString()
					}
					bEndPartnerConfig.Peers = append(bEndPartnerConfig.Peers, peeringConfig)
				}
			}

			azureConfigObj, azureDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAzureAttrs, azureConfig)
			resp.Diagnostics.Append(azureDiags...)

			partnerPortReq := &megaport.ListPartnerPortsRequest{
				Key:     azureConfig.ServiceKey.ValueString(),
				Partner: "AZURE",
			}
			partnerPortRes, err := r.client.VXCService.ListPartnerPorts(ctx, partnerPortReq)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error creating VXC",
					fmt.Sprintf("Could not create %s, there was an error looking up partner ports: %s", plan.Name.ValueString(), err.Error()),
				)
				return
			}
			// find primary or secondary port
			for _, port := range partnerPortRes.Data.Megaports {
				p := &port
				if p.Type == azureConfig.PortChoice.ValueString() {
					bEndConfig.ProductUID = p.ProductUID
				}
			}
			if bEndConfig.ProductUID == "" {
				resp.Diagnostics.AddError(
					"Error creating VXC",
					fmt.Sprintf("Could not find azure port with type: %s", azureConfig.PortChoice.ValueString()),
				)
				return
			}

			aws := types.ObjectNull(vxcPartnerConfigAWSAttrs)
			google := types.ObjectNull(vxcPartnerConfigGoogleAttrs)
			oracle := types.ObjectNull(vxcPartnerConfigOracleAttrs)
			vrouter := types.ObjectNull(vxcPartnerConfigVrouterAttrs)
			aEndPartner := types.ObjectNull(vxcPartnerConfigAEndAttrs)
			ibmPartner := types.ObjectNull(vxcPartnerConfigIbmAttrs)
			bEndPartnerConfigModel := &vxcPartnerConfigurationModel{
				Partner:              bPartnerConfig.Partner,
				AWSPartnerConfig:     aws,
				AzurePartnerConfig:   azureConfigObj,
				GooglePartnerConfig:  google,
				OraclePartnerConfig:  oracle,
				IBMPartnerConfig:     ibmPartner,
				VrouterPartnerConfig: vrouter,
				PartnerAEndConfig:    aEndPartner,
			}

			partnerConfigObj, partnerDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, bEndPartnerConfigModel)
			resp.Diagnostics.Append(partnerDiags...)
			plan.BEndPartnerConfig = partnerConfigObj
			bEndConfig.PartnerConfig = bEndPartnerConfig
		case "google":
			if bPartnerConfig.GooglePartnerConfig.IsNull() {
				resp.Diagnostics.AddError(
					"Error creating VXC",
					"Could not create VXC with name "+plan.Name.ValueString()+": Google Partner configuration is required",
				)
				return
			}
			var googleConfig vxcPartnerConfigGoogleModel
			googleDiags := bPartnerConfig.GooglePartnerConfig.As(ctx, &googleConfig, basetypes.ObjectAsOptions{})
			if googleDiags.HasError() {
				resp.Diagnostics.Append(googleDiags...)
				return
			}
			bEndPartnerConfig := megaport.VXCPartnerConfigGoogle{
				ConnectType: "GOOGLE",
				PairingKey:  googleConfig.PairingKey.ValueString(),
			}
			googleConfigObj, googleDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigGoogleAttrs, googleConfig)
			resp.Diagnostics.Append(googleDiags...)

			partnerPortReq := &megaport.LookupPartnerPortsRequest{
				Key:       googleConfig.PairingKey.ValueString(),
				PortSpeed: int(plan.RateLimit.ValueInt64()),
				Partner:   "GOOGLE",
			}
			if !b.RequestedProductUID.IsNull() {
				partnerPortReq.ProductID = b.RequestedProductUID.ValueString()
			}
			partnerPortRes, err := r.client.VXCService.LookupPartnerPorts(ctx, partnerPortReq)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error creating VXC",
					fmt.Sprintf("Could not create %s, there was an error looking up partner ports: %s", plan.Name.ValueString(), err.Error()),
				)
				return
			}
			bEndConfig.ProductUID = partnerPortRes.ProductUID

			aws := types.ObjectNull(vxcPartnerConfigAWSAttrs)
			azure := types.ObjectNull(vxcPartnerConfigAzureAttrs)
			oracle := types.ObjectNull(vxcPartnerConfigOracleAttrs)
			vrouter := types.ObjectNull(vxcPartnerConfigVrouterAttrs)
			aEndPartner := types.ObjectNull(vxcPartnerConfigAEndAttrs)
			ibmPartner := types.ObjectNull(vxcPartnerConfigIbmAttrs)
			bEndPartnerConfigModel := &vxcPartnerConfigurationModel{
				Partner:              bPartnerConfig.Partner,
				AWSPartnerConfig:     aws,
				AzurePartnerConfig:   azure,
				GooglePartnerConfig:  googleConfigObj,
				OraclePartnerConfig:  oracle,
				IBMPartnerConfig:     ibmPartner,
				VrouterPartnerConfig: vrouter,
				PartnerAEndConfig:    aEndPartner,
			}

			partnerConfigObj, partnerDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, bEndPartnerConfigModel)
			resp.Diagnostics.Append(partnerDiags...)
			plan.BEndPartnerConfig = partnerConfigObj
			bEndConfig.PartnerConfig = bEndPartnerConfig
		case "oracle":
			if bPartnerConfig.OraclePartnerConfig.IsNull() {
				resp.Diagnostics.AddError(
					"Error creating VXC",
					"Could not create VXC with name "+plan.Name.ValueString()+": Oracle Partner configuration is required",
				)
				return
			}
			var oracleConfig vxcPartnerConfigOracleModel
			oracleDiags := bPartnerConfig.OraclePartnerConfig.As(ctx, &oracleConfig, basetypes.ObjectAsOptions{})
			if oracleDiags.HasError() {
				resp.Diagnostics.Append(oracleDiags...)
				return
			}
			bEndPartnerConfig := megaport.VXCPartnerConfigOracle{
				ConnectType:      "ORACLE",
				VirtualCircuitId: oracleConfig.VirtualCircuitId.ValueString(),
			}

			partnerPortReq := &megaport.LookupPartnerPortsRequest{
				Key:       oracleConfig.VirtualCircuitId.ValueString(),
				PortSpeed: int(plan.RateLimit.ValueInt64()),
				Partner:   "ORACLE",
			}
			if !b.RequestedProductUID.IsNull() {
				partnerPortReq.ProductID = b.RequestedProductUID.ValueString()
			}
			partnerPortRes, err := r.client.VXCService.LookupPartnerPorts(ctx, partnerPortReq)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error creating VXC",
					fmt.Sprintf("Could not create %s, there was an error looking up partner ports: %s", plan.Name.ValueString(), err.Error()),
				)
				return
			}
			bEndConfig.ProductUID = partnerPortRes.ProductUID

			oracleConfigObj, oracleDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigOracleAttrs, oracleConfig)
			resp.Diagnostics.Append(oracleDiags...)

			aws := types.ObjectNull(vxcPartnerConfigAWSAttrs)
			azure := types.ObjectNull(vxcPartnerConfigAzureAttrs)
			google := types.ObjectNull(vxcPartnerConfigGoogleAttrs)
			vrouter := types.ObjectNull(vxcPartnerConfigVrouterAttrs)
			aEndPartner := types.ObjectNull(vxcPartnerConfigAEndAttrs)
			ibmPartner := types.ObjectNull(vxcPartnerConfigIbmAttrs)
			bEndPartnerConfigModel := &vxcPartnerConfigurationModel{
				Partner:              bPartnerConfig.Partner,
				AWSPartnerConfig:     aws,
				AzurePartnerConfig:   azure,
				GooglePartnerConfig:  google,
				OraclePartnerConfig:  oracleConfigObj,
				IBMPartnerConfig:     ibmPartner,
				VrouterPartnerConfig: vrouter,
				PartnerAEndConfig:    aEndPartner,
			}

			partnerConfigObj, partnerDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, bEndPartnerConfigModel)
			resp.Diagnostics.Append(partnerDiags...)
			plan.BEndPartnerConfig = partnerConfigObj
			bEndConfig.PartnerConfig = bEndPartnerConfig
		case "ibm":
			if bPartnerConfig.IBMPartnerConfig.IsNull() {
				resp.Diagnostics.AddError(
					"Error creating VXC",
					"Could not create VXC with name "+plan.Name.ValueString()+": IBM Partner configuration is required",
				)
				return
			}
			var ibmConfig vxcPartnerConfigIbmModel
			ibmDiags := bPartnerConfig.IBMPartnerConfig.As(ctx, &ibmConfig, basetypes.ObjectAsOptions{})
			resp.Diagnostics.Append(ibmDiags...)
			if resp.Diagnostics.HasError() {
				return
			}
			bEndPartnerConfig := megaport.VXCPartnerConfigIBM{
				ConnectType:       "IBM",
				AccountID:         ibmConfig.AccountID.ValueString(),
				CustomerASN:       int(ibmConfig.CustomerASN.ValueInt64()),
				Name:              ibmConfig.Name.ValueString(),
				CustomerIPAddress: ibmConfig.CustomerIPAddress.ValueString(),
				ProviderIPAddress: ibmConfig.ProviderIPAddress.ValueString(),
			}
			aws := types.ObjectNull(vxcPartnerConfigAWSAttrs)
			azure := types.ObjectNull(vxcPartnerConfigAzureAttrs)
			google := types.ObjectNull(vxcPartnerConfigGoogleAttrs)
			oracle := types.ObjectNull(vxcPartnerConfigOracleAttrs)
			vrouter := types.ObjectNull(vxcPartnerConfigVrouterAttrs)
			aEndPartner := types.ObjectNull(vxcPartnerConfigAEndAttrs)
			bEndPartnerConfigModel := &vxcPartnerConfigurationModel{
				Partner:              bPartnerConfig.Partner,
				AWSPartnerConfig:     aws,
				AzurePartnerConfig:   azure,
				GooglePartnerConfig:  google,
				OraclePartnerConfig:  oracle,
				VrouterPartnerConfig: vrouter,
				PartnerAEndConfig:    aEndPartner,
				IBMPartnerConfig:     bPartnerConfig.IBMPartnerConfig,
			}
			partnerConfigObj, partnerDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, bEndPartnerConfigModel)
			resp.Diagnostics.Append(partnerDiags...)
			plan.BEndPartnerConfig = partnerConfigObj
			bEndConfig.PartnerConfig = bEndPartnerConfig
		case "transit":
			bEndPartnerConfig := &megaport.VXCPartnerConfigTransit{
				ConnectType: "TRANSIT",
			}
			aws := types.ObjectNull(vxcPartnerConfigAWSAttrs)
			azure := types.ObjectNull(vxcPartnerConfigAzureAttrs)
			google := types.ObjectNull(vxcPartnerConfigGoogleAttrs)
			oracle := types.ObjectNull(vxcPartnerConfigOracleAttrs)
			vrouter := types.ObjectNull(vxcPartnerConfigVrouterAttrs)
			aEndPartner := types.ObjectNull(vxcPartnerConfigAEndAttrs)
			ibmPartner := types.ObjectNull(vxcPartnerConfigIbmAttrs)
			bEndPartnerConfigModel := &vxcPartnerConfigurationModel{
				Partner:              bPartnerConfig.Partner,
				AWSPartnerConfig:     aws,
				AzurePartnerConfig:   azure,
				GooglePartnerConfig:  google,
				OraclePartnerConfig:  oracle,
				VrouterPartnerConfig: vrouter,
				IBMPartnerConfig:     ibmPartner,
				PartnerAEndConfig:    aEndPartner,
			}

			partnerConfigObj, partnerDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, bEndPartnerConfigModel)
			resp.Diagnostics.Append(partnerDiags...)
			plan.BEndPartnerConfig = partnerConfigObj
			bEndConfig.PartnerConfig = bEndPartnerConfig
		case "vrouter":
			if bPartnerConfig.VrouterPartnerConfig.IsNull() {
				resp.Diagnostics.AddError(
					"Error creating VXC",
					"Could not create VXC with name "+plan.Name.ValueString()+": Virtual router configuration is required",
				)
				return
			}
			var partnerConfigBEnd vxcPartnerConfigVrouterModel
			bEndDiags := bPartnerConfig.VrouterPartnerConfig.As(ctx, &partnerConfigBEnd, basetypes.ObjectAsOptions{})
			if aEndDiags.HasError() {
				resp.Diagnostics.Append(bEndDiags...)
				return
			}
			prefixFilterListRes, err := r.client.MCRService.ListMCRPrefixFilterLists(ctx, b.RequestedProductUID.ValueString())
			if err != nil {
				resp.Diagnostics.AddError(
					"Error creating VXC",
					"Could not create VXC with name "+plan.Name.ValueString()+": "+err.Error(),
				)
				return
			}

			bEndMegaportConfig := megaport.VXCOrderVrouterPartnerConfig{}
			ifaceModels := []*vxcPartnerConfigInterfaceModel{}
			ifaceDiags := partnerConfigBEnd.Interfaces.ElementsAs(ctx, &ifaceModels, false)
			resp.Diagnostics = append(resp.Diagnostics, ifaceDiags...)
			for _, iface := range ifaceModels {
				toAppend := megaport.PartnerConfigInterface{}
				if !iface.IPAddresses.IsNull() {
					ipAddresses := []string{}
					ipDiags := iface.IPAddresses.ElementsAs(ctx, &ipAddresses, true)
					resp.Diagnostics = append(resp.Diagnostics, ipDiags...)
					toAppend.IpAddresses = ipAddresses
				}
				if !iface.IPRoutes.IsNull() {
					ipRoutes := []*ipRouteModel{}
					ipRouteDiags := iface.IPRoutes.ElementsAs(ctx, &ipRoutes, true)
					resp.Diagnostics = append(resp.Diagnostics, ipRouteDiags...)
					for _, ipRoute := range ipRoutes {
						toAppend.IpRoutes = append(toAppend.IpRoutes, megaport.IpRoute{
							Prefix:      ipRoute.Prefix.ValueString(),
							Description: ipRoute.Description.ValueString(),
							NextHop:     ipRoute.NextHop.ValueString(),
						})
					}
				}
				if !iface.NatIPAddresses.IsNull() {
					natIPAddresses := []string{}
					natDiags := iface.NatIPAddresses.ElementsAs(ctx, &natIPAddresses, true)
					resp.Diagnostics = append(resp.Diagnostics, natDiags...)
					toAppend.NatIpAddresses = natIPAddresses
				}
				if !iface.Bfd.IsNull() {
					bfd := &bfdConfigModel{}
					bfdDiags := iface.Bfd.As(ctx, bfd, basetypes.ObjectAsOptions{})
					resp.Diagnostics = append(resp.Diagnostics, bfdDiags...)
					toAppend.Bfd = megaport.BfdConfig{
						TxInterval: int(bfd.TxInterval.ValueInt64()),
						RxInterval: int(bfd.RxInterval.ValueInt64()),
						Multiplier: int(bfd.Multiplier.ValueInt64()),
					}
				}
				if !iface.VLAN.IsNull() {
					toAppend.VLAN = int(iface.VLAN.ValueInt64())
				}
				if !iface.BgpConnections.IsNull() {
					bgpConnections := []*bgpConnectionConfigModel{}
					bgpDiags := iface.BgpConnections.ElementsAs(ctx, &bgpConnections, false)
					resp.Diagnostics = append(resp.Diagnostics, bgpDiags...)
					for _, bgpConnection := range bgpConnections {
						bgpToAppend := megaport.BgpConnectionConfig{
							PeerAsn:            int(bgpConnection.PeerAsn.ValueInt64()),
							LocalIpAddress:     bgpConnection.LocalIPAddress.ValueString(),
							PeerIpAddress:      bgpConnection.PeerIPAddress.ValueString(),
							Password:           bgpConnection.Password.ValueString(),
							Shutdown:           bgpConnection.Shutdown.ValueBool(),
							Description:        bgpConnection.Description.ValueString(),
							MedIn:              int(bgpConnection.MedIn.ValueInt64()),
							MedOut:             int(bgpConnection.MedOut.ValueInt64()),
							BfdEnabled:         bgpConnection.BfdEnabled.ValueBool(),
							ExportPolicy:       bgpConnection.ExportPolicy.ValueString(),
							AsPathPrependCount: int(bgpConnection.AsPathPrependCount.ValueInt64()),
							PeerType:           bgpConnection.PeerType.ValueString(),
						}
						if !bgpConnection.LocalAsn.IsNull() {
							bgpToAppend.LocalAsn = megaport.PtrTo(int(bgpConnection.LocalAsn.ValueInt64()))
						}
						if !bgpConnection.ImportWhitelist.IsNull() {
							for _, prefixFilterList := range prefixFilterListRes {
								if prefixFilterList.Description == bgpConnection.ImportWhitelist.ValueString() {
									bgpToAppend.ImportWhitelist = prefixFilterList.Id
								}
							}
						}
						if !bgpConnection.ImportBlacklist.IsNull() {
							for _, prefixFilterList := range prefixFilterListRes {
								if prefixFilterList.Description == bgpConnection.ImportBlacklist.ValueString() {
									bgpToAppend.ImportBlacklist = prefixFilterList.Id
								}
							}
						}
						if !bgpConnection.ExportWhitelist.IsNull() {
							for _, prefixFilterList := range prefixFilterListRes {
								if prefixFilterList.Description == bgpConnection.ExportWhitelist.ValueString() {
									bgpToAppend.ExportWhitelist = prefixFilterList.Id
								}
							}
						}
						if !bgpConnection.ExportBlacklist.IsNull() {
							for _, prefixFilterList := range prefixFilterListRes {
								if prefixFilterList.Description == bgpConnection.ExportBlacklist.ValueString() {
									bgpToAppend.ExportBlacklist = prefixFilterList.Id
								}
							}
						}
						if !bgpConnection.PermitExportTo.IsNull() {
							permitExportTo := []string{}
							permitDiags := bgpConnection.PermitExportTo.ElementsAs(ctx, &permitExportTo, true)
							resp.Diagnostics = append(resp.Diagnostics, permitDiags...)
							bgpToAppend.PermitExportTo = permitExportTo
							bgpToAppend.PermitExportTo = permitExportTo
						}
						if !bgpConnection.DenyExportTo.IsNull() {
							denyExportTo := []string{}
							denyDiags := bgpConnection.DenyExportTo.ElementsAs(ctx, &denyExportTo, true)
							resp.Diagnostics = append(resp.Diagnostics, denyDiags...)
							bgpToAppend.DenyExportTo = denyExportTo
						}
						toAppend.BgpConnections = append(toAppend.BgpConnections, bgpToAppend)
					}
				}
				bEndMegaportConfig.Interfaces = append(bEndMegaportConfig.Interfaces, toAppend)
			}
			vrouterConfigObj, bEndDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigVrouterAttrs, partnerConfigBEnd)
			resp.Diagnostics.Append(bEndDiags...)
			aws := types.ObjectNull(vxcPartnerConfigAWSAttrs)
			azure := types.ObjectNull(vxcPartnerConfigAzureAttrs)
			google := types.ObjectNull(vxcPartnerConfigGoogleAttrs)
			oracle := types.ObjectNull(vxcPartnerConfigOracleAttrs)
			aEndPartner := types.ObjectNull(vxcPartnerConfigAEndAttrs)
			ibmPartner := types.ObjectNull(vxcPartnerConfigIbmAttrs)
			bEndPartnerConfigModel := &vxcPartnerConfigurationModel{
				Partner:              bPartnerConfig.Partner,
				AWSPartnerConfig:     aws,
				AzurePartnerConfig:   azure,
				GooglePartnerConfig:  google,
				OraclePartnerConfig:  oracle,
				VrouterPartnerConfig: vrouterConfigObj,
				IBMPartnerConfig:     ibmPartner,
				PartnerAEndConfig:    aEndPartner,
			}
			bEndPartnerConfigObj, partnerDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, bEndPartnerConfigModel)
			resp.Diagnostics.Append(partnerDiags...)
			plan.BEndPartnerConfig = bEndPartnerConfigObj
			bEndConfig.PartnerConfig = bEndMegaportConfig
		default:
			resp.Diagnostics.AddError(
				"Error creating VXC",
				"Could not create VXC with name "+plan.Name.ValueString()+": Partner configuration not supported",
			)
			return
		}
	}

	buyReq.BEndConfiguration = *bEndConfig

	buyReq.BEndConfiguration = *bEndConfig

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

	var aEndPartnerChange, bEndPartnerChange bool

	// If Imported, AEndPartnerConfig will be null. Set the partner config to the existing one in the plan.
	if !plan.AEndPartnerConfig.Equal(state.AEndPartnerConfig) {
		aEndPartnerChange = true
	}
	if state.AEndPartnerConfig.IsNull() {
		state.AEndPartnerConfig = plan.AEndPartnerConfig
	}
	if !plan.BEndPartnerConfig.Equal(state.BEndPartnerConfig) {
		bEndPartnerChange = true
	}
	if state.BEndPartnerConfig.IsNull() {
		state.BEndPartnerConfig = plan.BEndPartnerConfig
	}

	var aEndPlan, bEndPlan, aEndState, bEndState *vxcBasicEndConfigurationModel
	var aEndPartnerPlan, bEndPartnerPlan, aEndPartnerState, bEndPartnerState *vxcPartnerConfigurationModel

	// Check if AEnd or BEnd is a CSP Partner Configuration
	var aEndCSP, bEndCSP bool

	aEndPlanDiags := plan.AEndConfiguration.As(ctx, &aEndPlan, basetypes.ObjectAsOptions{})
	resp.Diagnostics.Append(aEndPlanDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	bEndPlanDiags := plan.BEndConfiguration.As(ctx, &bEndPlan, basetypes.ObjectAsOptions{})
	resp.Diagnostics.Append(bEndPlanDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	aEndStateDiags := state.AEndConfiguration.As(ctx, &aEndState, basetypes.ObjectAsOptions{})
	resp.Diagnostics.Append(aEndStateDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	bEndStateDiags := state.BEndConfiguration.As(ctx, &bEndState, basetypes.ObjectAsOptions{})
	resp.Diagnostics.Append(bEndStateDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	aEndPartnerPlanDiags := plan.AEndPartnerConfig.As(ctx, &aEndPartnerPlan, basetypes.ObjectAsOptions{})
	resp.Diagnostics.Append(aEndPartnerPlanDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !plan.AEndPartnerConfig.IsNull() {
		if !aEndPartnerPlan.Partner.IsNull() {
			// Check if the partner is a CSP Partner
			if aEndPartnerPlan.Partner.ValueString() != "a-end" && aEndPartnerPlan.Partner.ValueString() != "vrouter" && aEndPartnerPlan.Partner.ValueString() != "transit" {
				aEndCSP = true
			}
		}
	}
	bEndPartnerPlanDiags := plan.BEndPartnerConfig.As(ctx, &bEndPartnerPlan, basetypes.ObjectAsOptions{})
	resp.Diagnostics.Append(bEndPartnerPlanDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !plan.BEndPartnerConfig.IsNull() {
		if !bEndPartnerPlan.Partner.IsNull() {
			// Check if the partner is a CSP Partner
			if bEndPartnerPlan.Partner.ValueString() != "a-end" && bEndPartnerPlan.Partner.ValueString() != "vrouter" && bEndPartnerPlan.Partner.ValueString() != "transit" {
				bEndCSP = true
			}
		}
	}

	aEndPartnerStateDiags := state.AEndPartnerConfig.As(ctx, &aEndPartnerState, basetypes.ObjectAsOptions{})
	resp.Diagnostics.Append(aEndPartnerStateDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	bEndPartnerStateDiags := state.BEndPartnerConfig.As(ctx, &bEndPartnerState, basetypes.ObjectAsOptions{})
	resp.Diagnostics.Append(bEndPartnerStateDiags...)
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

	// If Ordered VLAN is different from actual VLAN, attempt to change it to the ordered VLAN value.
	if !aEndPlan.OrderedVLAN.IsUnknown() && !aEndPlan.OrderedVLAN.IsNull() && !aEndPlan.OrderedVLAN.Equal(aEndState.VLAN) {
		updateReq.AEndVLAN = megaport.PtrTo(int(aEndPlan.OrderedVLAN.ValueInt64()))
	}
	aEndState.OrderedVLAN = aEndPlan.OrderedVLAN

	if !aEndPlan.InnerVLAN.IsUnknown() && !aEndPlan.InnerVLAN.IsNull() && !aEndPlan.InnerVLAN.Equal(aEndState.InnerVLAN) {
		updateReq.AEndInnerVLAN = megaport.PtrTo(int(aEndPlan.InnerVLAN.ValueInt64()))
	}
	aEndState.InnerVLAN = aEndPlan.InnerVLAN

	// If Ordered VLAN is different from actual VLAN, attempt to change it to the ordered VLAN value.
	if !bEndPlan.OrderedVLAN.IsUnknown() && !bEndPlan.OrderedVLAN.IsNull() && !bEndPlan.OrderedVLAN.Equal(bEndState.VLAN) {
		updateReq.BEndVLAN = megaport.PtrTo(int(bEndPlan.OrderedVLAN.ValueInt64()))
	}
	bEndState.OrderedVLAN = bEndPlan.OrderedVLAN

	if !bEndPlan.InnerVLAN.IsUnknown() && !bEndPlan.InnerVLAN.IsNull() && !bEndPlan.InnerVLAN.Equal(bEndState.InnerVLAN) {
		updateReq.BEndInnerVLAN = megaport.PtrTo(int(bEndPlan.InnerVLAN.ValueInt64()))
	}
	bEndState.InnerVLAN = bEndPlan.InnerVLAN

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

	if !aEndPlan.RequestedProductUID.IsNull() && !aEndPlan.RequestedProductUID.Equal(aEndState.RequestedProductUID) {
		// Do not update the product UID if the partner is a CSP
		if !aEndCSP {
			updateReq.AEndProductUID = megaport.PtrTo(aEndPlan.RequestedProductUID.ValueString())
			aEndState.RequestedProductUID = aEndPlan.RequestedProductUID
		}
	}
	if !bEndPlan.RequestedProductUID.IsNull() && !bEndPlan.RequestedProductUID.Equal(bEndState.RequestedProductUID) {
		// Do not update the product UID if the partner is a CSP
		if !bEndCSP {
			updateReq.BEndProductUID = megaport.PtrTo(bEndPlan.RequestedProductUID.ValueString())
			bEndState.RequestedProductUID = bEndPlan.RequestedProductUID
		}
	}

	if !plan.AEndPartnerConfig.IsNull() && aEndPartnerChange && !aEndCSP {
		aPartnerConfig := aEndPartnerPlan
		switch aEndPartnerPlan.Partner.ValueString() {
		case "transit":
			aEndPartnerConfig := &megaport.VXCPartnerConfigTransit{
				ConnectType: "TRANSIT",
			}
			aws := types.ObjectNull(vxcPartnerConfigAWSAttrs)
			azure := types.ObjectNull(vxcPartnerConfigAzureAttrs)
			google := types.ObjectNull(vxcPartnerConfigGoogleAttrs)
			oracle := types.ObjectNull(vxcPartnerConfigOracleAttrs)
			vrouter := types.ObjectNull(vxcPartnerConfigVrouterAttrs)
			aEndPartner := types.ObjectNull(vxcPartnerConfigAEndAttrs)
			ibmPartner := types.ObjectNull(vxcPartnerConfigIbmAttrs)
			aEndPartnerConfigModel := &vxcPartnerConfigurationModel{
				Partner:              aPartnerConfig.Partner,
				AWSPartnerConfig:     aws,
				AzurePartnerConfig:   azure,
				GooglePartnerConfig:  google,
				OraclePartnerConfig:  oracle,
				VrouterPartnerConfig: vrouter,
				IBMPartnerConfig:     ibmPartner,
				PartnerAEndConfig:    aEndPartner,
			}

			partnerConfigObj, partnerDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, aEndPartnerConfigModel)
			resp.Diagnostics.Append(partnerDiags...)
			state.AEndPartnerConfig = partnerConfigObj
			updateReq.AEndPartnerConfig = aEndPartnerConfig
		case "a-end":
			if aPartnerConfig.PartnerAEndConfig.IsNull() {
				resp.Diagnostics.AddError(
					"Error updating VXC",
					"Could not update VXC with name "+plan.Name.ValueString()+": A-End Partner configuration is required",
				)
				return
			}
			var partnerConfigAEnd vxcPartnerConfigAEndModel
			aEndDiags := aPartnerConfig.PartnerAEndConfig.As(ctx, &partnerConfigAEnd, basetypes.ObjectAsOptions{})
			resp.Diagnostics.Append(aEndDiags...)
			if resp.Diagnostics.HasError() {
				return
			}
			prefixFilterListRes, err := r.client.MCRService.ListMCRPrefixFilterLists(ctx, aEndPlan.RequestedProductUID.ValueString())
			if err != nil {
				resp.Diagnostics.AddError(
					"Error updating VXC",
					"Could not update VXC with name "+plan.Name.ValueString()+": "+err.Error(),
				)
				return
			}

			aEndPartnerConfig := megaport.VXCOrderVrouterPartnerConfig{}
			ifaceModels := []*vxcPartnerConfigInterfaceModel{}
			ifaceDiags := partnerConfigAEnd.Interfaces.ElementsAs(ctx, &ifaceModels, true)
			resp.Diagnostics = append(resp.Diagnostics, ifaceDiags...)
			for _, iface := range ifaceModels {
				toAppend := megaport.PartnerConfigInterface{}
				if !iface.IPAddresses.IsNull() {
					ipAddresses := []string{}
					ipDiags := iface.IPAddresses.ElementsAs(ctx, &ipAddresses, true)
					resp.Diagnostics = append(resp.Diagnostics, ipDiags...)
					toAppend.IpAddresses = ipAddresses
				}
				if !iface.IPRoutes.IsNull() {
					ipRoutes := []*ipRouteModel{}
					ipRouteDiags := iface.IPRoutes.ElementsAs(ctx, &ipRoutes, true)
					resp.Diagnostics = append(resp.Diagnostics, ipRouteDiags...)
					for _, ipRoute := range ipRoutes {
						toAppend.IpRoutes = append(toAppend.IpRoutes, megaport.IpRoute{
							Prefix:      ipRoute.Prefix.ValueString(),
							Description: ipRoute.Description.ValueString(),
							NextHop:     ipRoute.NextHop.ValueString(),
						})
					}
				}
				if !iface.NatIPAddresses.IsNull() {
					natIPAddresses := []string{}
					natDiags := iface.NatIPAddresses.ElementsAs(ctx, &natIPAddresses, true)
					resp.Diagnostics = append(resp.Diagnostics, natDiags...)
					toAppend.NatIpAddresses = natIPAddresses
				}
				if !iface.Bfd.IsNull() {
					bfd := &bfdConfigModel{}
					bfdDiags := iface.Bfd.As(ctx, bfd, basetypes.ObjectAsOptions{})
					resp.Diagnostics = append(resp.Diagnostics, bfdDiags...)
					toAppend.Bfd = megaport.BfdConfig{
						TxInterval: int(bfd.TxInterval.ValueInt64()),
						RxInterval: int(bfd.RxInterval.ValueInt64()),
						Multiplier: int(bfd.Multiplier.ValueInt64()),
					}
				}
				if !iface.BgpConnections.IsNull() {
					bgpConnections := []*bgpConnectionConfigModel{}
					bgpDiags := iface.BgpConnections.ElementsAs(ctx, &bgpConnections, false)
					resp.Diagnostics = append(resp.Diagnostics, bgpDiags...)
					for _, bgpConnection := range bgpConnections {
						bgpToAppend := megaport.BgpConnectionConfig{
							PeerAsn:            int(bgpConnection.PeerAsn.ValueInt64()),
							LocalIpAddress:     bgpConnection.LocalIPAddress.ValueString(),
							PeerIpAddress:      bgpConnection.PeerIPAddress.ValueString(),
							Password:           bgpConnection.Password.ValueString(),
							Shutdown:           bgpConnection.Shutdown.ValueBool(),
							Description:        bgpConnection.Description.ValueString(),
							MedIn:              int(bgpConnection.MedIn.ValueInt64()),
							MedOut:             int(bgpConnection.MedOut.ValueInt64()),
							BfdEnabled:         bgpConnection.BfdEnabled.ValueBool(),
							ExportPolicy:       bgpConnection.ExportPolicy.ValueString(),
							AsPathPrependCount: int(bgpConnection.AsPathPrependCount.ValueInt64()),
						}
						if !bgpConnection.LocalAsn.IsNull() {
							bgpToAppend.LocalAsn = megaport.PtrTo(int(bgpConnection.LocalAsn.ValueInt64()))
						}
						if !bgpConnection.ImportWhitelist.IsNull() {
							for _, prefixFilterList := range prefixFilterListRes {
								if prefixFilterList.Description == bgpConnection.ImportWhitelist.ValueString() {
									bgpToAppend.ImportWhitelist = prefixFilterList.Id
								}
							}
						}
						if !bgpConnection.ImportBlacklist.IsNull() {
							for _, prefixFilterList := range prefixFilterListRes {
								if prefixFilterList.Description == bgpConnection.ImportBlacklist.ValueString() {
									bgpToAppend.ImportBlacklist = prefixFilterList.Id
								}
							}
						}
						if !bgpConnection.ExportWhitelist.IsNull() {
							for _, prefixFilterList := range prefixFilterListRes {
								if prefixFilterList.Description == bgpConnection.ExportWhitelist.ValueString() {
									bgpToAppend.ExportWhitelist = prefixFilterList.Id
								}
							}
						}
						if !bgpConnection.ExportBlacklist.IsNull() {
							for _, prefixFilterList := range prefixFilterListRes {
								if prefixFilterList.Description == bgpConnection.ExportBlacklist.ValueString() {
									bgpToAppend.ExportBlacklist = prefixFilterList.Id
								}
							}
						}
						if !bgpConnection.PermitExportTo.IsNull() {
							permitExportTo := []string{}
							permitDiags := bgpConnection.PermitExportTo.ElementsAs(ctx, &permitExportTo, true)
							resp.Diagnostics = append(resp.Diagnostics, permitDiags...)
							bgpToAppend.PermitExportTo = permitExportTo
							bgpToAppend.PermitExportTo = permitExportTo
						}
						if !bgpConnection.DenyExportTo.IsNull() {
							denyExportTo := []string{}
							denyDiags := bgpConnection.DenyExportTo.ElementsAs(ctx, &denyExportTo, true)
							resp.Diagnostics = append(resp.Diagnostics, denyDiags...)
							bgpToAppend.DenyExportTo = denyExportTo
						}
						toAppend.BgpConnections = append(toAppend.BgpConnections, bgpToAppend)
					}
				}
				aEndPartnerConfig.Interfaces = append(aEndPartnerConfig.Interfaces, toAppend)
			}
			aEndConfigObj, aEndDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAEndAttrs, partnerConfigAEnd)
			resp.Diagnostics.Append(aEndDiags...)
			aws := types.ObjectNull(vxcPartnerConfigAWSAttrs)
			azure := types.ObjectNull(vxcPartnerConfigAzureAttrs)
			google := types.ObjectNull(vxcPartnerConfigGoogleAttrs)
			oracle := types.ObjectNull(vxcPartnerConfigOracleAttrs)
			vrouter := types.ObjectNull(vxcPartnerConfigVrouterAttrs)
			ibmPartner := types.ObjectNull(vxcPartnerConfigIbmAttrs)
			aEndPartnerConfigModel := &vxcPartnerConfigurationModel{
				Partner:              aPartnerConfig.Partner,
				AWSPartnerConfig:     aws,
				AzurePartnerConfig:   azure,
				GooglePartnerConfig:  google,
				OraclePartnerConfig:  oracle,
				IBMPartnerConfig:     ibmPartner,
				PartnerAEndConfig:    aEndConfigObj,
				VrouterPartnerConfig: vrouter,
			}
			aEndPartnerConfigObj, partnerDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, aEndPartnerConfigModel)
			resp.Diagnostics.Append(partnerDiags...)
			state.AEndPartnerConfig = aEndPartnerConfigObj
			updateReq.AEndPartnerConfig = aEndPartnerConfig
		case "vrouter":
			if aEndPartnerPlan.VrouterPartnerConfig.IsNull() {
				resp.Diagnostics.AddError(
					"Error updating VXC",
					"Could not update VXC with name "+plan.Name.ValueString()+": Virtual router configuration is required",
				)
				return
			}
			var partnerConfigAEnd vxcPartnerConfigVrouterModel
			aEndDiags := aEndPartnerPlan.VrouterPartnerConfig.As(ctx, &partnerConfigAEnd, basetypes.ObjectAsOptions{})
			resp.Diagnostics.Append(aEndDiags...)
			if resp.Diagnostics.HasError() {
				return
			}
			prefixFilterListRes, err := r.client.MCRService.ListMCRPrefixFilterLists(ctx, aEndState.RequestedProductUID.ValueString())
			if err != nil {
				resp.Diagnostics.AddError(
					"Error updating VXC",
					"Could not update VXC with name "+plan.Name.ValueString()+": "+err.Error(),
				)
				return
			}

			aEndMegaportConfig := &megaport.VXCOrderVrouterPartnerConfig{}
			ifaceModels := []*vxcPartnerConfigInterfaceModel{}
			ifaceDiags := partnerConfigAEnd.Interfaces.ElementsAs(ctx, &ifaceModels, false)
			resp.Diagnostics = append(resp.Diagnostics, ifaceDiags...)
			for _, iface := range ifaceModels {
				toAppend := megaport.PartnerConfigInterface{}
				if !iface.IPAddresses.IsNull() {
					ipAddresses := []string{}
					ipDiags := iface.IPAddresses.ElementsAs(ctx, &ipAddresses, true)
					resp.Diagnostics = append(resp.Diagnostics, ipDiags...)
					toAppend.IpAddresses = ipAddresses
				}
				if !iface.IPRoutes.IsNull() {
					ipRoutes := []*ipRouteModel{}
					ipRouteDiags := iface.IPRoutes.ElementsAs(ctx, &ipRoutes, true)
					resp.Diagnostics = append(resp.Diagnostics, ipRouteDiags...)
					for _, ipRoute := range ipRoutes {
						toAppend.IpRoutes = append(toAppend.IpRoutes, megaport.IpRoute{
							Prefix:      ipRoute.Prefix.ValueString(),
							Description: ipRoute.Description.ValueString(),
							NextHop:     ipRoute.NextHop.ValueString(),
						})
					}
				}
				if !iface.NatIPAddresses.IsNull() {
					natIPAddresses := []string{}
					natDiags := iface.NatIPAddresses.ElementsAs(ctx, &natIPAddresses, true)
					resp.Diagnostics = append(resp.Diagnostics, natDiags...)
					toAppend.NatIpAddresses = natIPAddresses
				}
				if !iface.Bfd.IsNull() {
					bfd := &bfdConfigModel{}
					bfdDiags := iface.Bfd.As(ctx, bfd, basetypes.ObjectAsOptions{})
					resp.Diagnostics = append(resp.Diagnostics, bfdDiags...)
					toAppend.Bfd = megaport.BfdConfig{
						TxInterval: int(bfd.TxInterval.ValueInt64()),
						RxInterval: int(bfd.RxInterval.ValueInt64()),
						Multiplier: int(bfd.Multiplier.ValueInt64()),
					}
				}
				if !iface.VLAN.IsNull() {
					toAppend.VLAN = int(iface.VLAN.ValueInt64())
				}
				if !iface.BgpConnections.IsNull() {
					bgpConnections := []*bgpConnectionConfigModel{}
					bgpDiags := iface.BgpConnections.ElementsAs(ctx, &bgpConnections, false)
					resp.Diagnostics = append(resp.Diagnostics, bgpDiags...)
					for _, bgpConnection := range bgpConnections {
						bgpToAppend := megaport.BgpConnectionConfig{
							PeerAsn:            int(bgpConnection.PeerAsn.ValueInt64()),
							LocalIpAddress:     bgpConnection.LocalIPAddress.ValueString(),
							PeerIpAddress:      bgpConnection.PeerIPAddress.ValueString(),
							Password:           bgpConnection.Password.ValueString(),
							Shutdown:           bgpConnection.Shutdown.ValueBool(),
							Description:        bgpConnection.Description.ValueString(),
							MedIn:              int(bgpConnection.MedIn.ValueInt64()),
							MedOut:             int(bgpConnection.MedOut.ValueInt64()),
							BfdEnabled:         bgpConnection.BfdEnabled.ValueBool(),
							ExportPolicy:       bgpConnection.ExportPolicy.ValueString(),
							AsPathPrependCount: int(bgpConnection.AsPathPrependCount.ValueInt64()),
							PeerType:           bgpConnection.PeerType.ValueString(),
						}
						if !bgpConnection.LocalAsn.IsNull() {
							bgpToAppend.LocalAsn = megaport.PtrTo(int(bgpConnection.LocalAsn.ValueInt64()))
						}
						if !bgpConnection.ImportWhitelist.IsNull() {
							for _, prefixFilterList := range prefixFilterListRes {
								if prefixFilterList.Description == bgpConnection.ImportWhitelist.ValueString() {
									bgpToAppend.ImportWhitelist = prefixFilterList.Id
								}
							}
						}
						if !bgpConnection.ImportBlacklist.IsNull() {
							for _, prefixFilterList := range prefixFilterListRes {
								if prefixFilterList.Description == bgpConnection.ImportBlacklist.ValueString() {
									bgpToAppend.ImportBlacklist = prefixFilterList.Id
								}
							}
						}
						if !bgpConnection.ExportWhitelist.IsNull() {
							for _, prefixFilterList := range prefixFilterListRes {
								if prefixFilterList.Description == bgpConnection.ExportWhitelist.ValueString() {
									bgpToAppend.ExportWhitelist = prefixFilterList.Id
								}
							}
						}
						if !bgpConnection.ExportBlacklist.IsNull() {
							for _, prefixFilterList := range prefixFilterListRes {
								if prefixFilterList.Description == bgpConnection.ExportBlacklist.ValueString() {
									bgpToAppend.ExportBlacklist = prefixFilterList.Id
								}
							}
						}
						if !bgpConnection.PermitExportTo.IsNull() {
							permitExportTo := []string{}
							permitDiags := bgpConnection.PermitExportTo.ElementsAs(ctx, &permitExportTo, true)
							resp.Diagnostics = append(resp.Diagnostics, permitDiags...)
							bgpToAppend.PermitExportTo = permitExportTo
							bgpToAppend.PermitExportTo = permitExportTo
						}
						if !bgpConnection.DenyExportTo.IsNull() {
							denyExportTo := []string{}
							denyDiags := bgpConnection.DenyExportTo.ElementsAs(ctx, &denyExportTo, true)
							resp.Diagnostics = append(resp.Diagnostics, denyDiags...)
							bgpToAppend.DenyExportTo = denyExportTo
						}
						toAppend.BgpConnections = append(toAppend.BgpConnections, bgpToAppend)
					}
				}
				aEndMegaportConfig.Interfaces = append(aEndMegaportConfig.Interfaces, toAppend)
			}
			vRouterConfigObj, aEndDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigVrouterAttrs, partnerConfigAEnd)
			resp.Diagnostics.Append(aEndDiags...)
			aws := types.ObjectNull(vxcPartnerConfigAWSAttrs)
			azure := types.ObjectNull(vxcPartnerConfigAzureAttrs)
			google := types.ObjectNull(vxcPartnerConfigGoogleAttrs)
			oracle := types.ObjectNull(vxcPartnerConfigOracleAttrs)
			aEndPartner := types.ObjectNull(vxcPartnerConfigAEndAttrs)
			ibmPartner := types.ObjectNull(vxcPartnerConfigIbmAttrs)
			aEndPartnerConfigModel := &vxcPartnerConfigurationModel{
				Partner:              aEndPartnerPlan.Partner,
				AWSPartnerConfig:     aws,
				AzurePartnerConfig:   azure,
				GooglePartnerConfig:  google,
				OraclePartnerConfig:  oracle,
				IBMPartnerConfig:     ibmPartner,
				VrouterPartnerConfig: vRouterConfigObj,
				PartnerAEndConfig:    aEndPartner,
			}
			aEndPartnerConfigObj, partnerDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, aEndPartnerConfigModel)
			resp.Diagnostics.Append(partnerDiags...)
			state.AEndPartnerConfig = aEndPartnerConfigObj
			updateReq.AEndPartnerConfig = aEndMegaportConfig
		default:
			resp.Diagnostics.AddError(
				"Error Updating VXC",
				"Could not update VXC with ID "+state.UID.ValueString()+": Partner configuration not supported",
			)
			return
		}
	}

	if !plan.BEndPartnerConfig.IsNull() && bEndPartnerChange && !bEndCSP {
		bPartnerConfig := bEndPartnerPlan
		switch bEndPartnerPlan.Partner.ValueString() {
		case "transit":
			bEndPartnerConfig := &megaport.VXCPartnerConfigTransit{
				ConnectType: "TRANSIT",
			}
			aws := types.ObjectNull(vxcPartnerConfigAWSAttrs)
			azure := types.ObjectNull(vxcPartnerConfigAzureAttrs)
			google := types.ObjectNull(vxcPartnerConfigGoogleAttrs)
			oracle := types.ObjectNull(vxcPartnerConfigOracleAttrs)
			vrouter := types.ObjectNull(vxcPartnerConfigVrouterAttrs)
			aEndPartner := types.ObjectNull(vxcPartnerConfigAEndAttrs)
			ibmPartner := types.ObjectNull(vxcPartnerConfigIbmAttrs)
			bEndPartnerConfigModel := &vxcPartnerConfigurationModel{
				Partner:              bPartnerConfig.Partner,
				AWSPartnerConfig:     aws,
				AzurePartnerConfig:   azure,
				GooglePartnerConfig:  google,
				OraclePartnerConfig:  oracle,
				VrouterPartnerConfig: vrouter,
				IBMPartnerConfig:     ibmPartner,
				PartnerAEndConfig:    aEndPartner,
			}

			partnerConfigObj, partnerDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, bEndPartnerConfigModel)
			resp.Diagnostics.Append(partnerDiags...)
			state.AEndPartnerConfig = partnerConfigObj
			updateReq.AEndPartnerConfig = bEndPartnerConfig
		case "vrouter":
			if bEndPartnerPlan.VrouterPartnerConfig.IsNull() {
				resp.Diagnostics.AddError(
					"Error creating VXC",
					"Could not create VXC with name "+plan.Name.ValueString()+": Virtual router configuration is required",
				)
				return
			}
			var partnerConfigBEnd vxcPartnerConfigVrouterModel
			bEndDiags := bEndPartnerPlan.VrouterPartnerConfig.As(ctx, &partnerConfigBEnd, basetypes.ObjectAsOptions{})
			resp.Diagnostics.Append(bEndDiags...)
			if resp.Diagnostics.HasError() {
				return
			}
			prefixFilterListRes, err := r.client.MCRService.ListMCRPrefixFilterLists(ctx, bEndState.RequestedProductUID.ValueString())
			if err != nil {
				resp.Diagnostics.AddError(
					"Error updating VXC",
					"Could not update VXC with name "+plan.Name.ValueString()+": "+err.Error(),
				)
				return
			}

			bEndMegaportConfig := &megaport.VXCOrderVrouterPartnerConfig{}
			ifaceModels := []*vxcPartnerConfigInterfaceModel{}
			ifaceDiags := partnerConfigBEnd.Interfaces.ElementsAs(ctx, &ifaceModels, false)
			resp.Diagnostics = append(resp.Diagnostics, ifaceDiags...)
			for _, iface := range ifaceModels {
				toAppend := megaport.PartnerConfigInterface{}
				if !iface.IPAddresses.IsNull() {
					ipAddresses := []string{}
					ipDiags := iface.IPAddresses.ElementsAs(ctx, &ipAddresses, true)
					resp.Diagnostics = append(resp.Diagnostics, ipDiags...)
					toAppend.IpAddresses = ipAddresses
				}
				if !iface.IPRoutes.IsNull() {
					ipRoutes := []*ipRouteModel{}
					ipRouteDiags := iface.IPRoutes.ElementsAs(ctx, &ipRoutes, true)
					resp.Diagnostics = append(resp.Diagnostics, ipRouteDiags...)
					for _, ipRoute := range ipRoutes {
						toAppend.IpRoutes = append(toAppend.IpRoutes, megaport.IpRoute{
							Prefix:      ipRoute.Prefix.ValueString(),
							Description: ipRoute.Description.ValueString(),
							NextHop:     ipRoute.NextHop.ValueString(),
						})
					}
				}
				if !iface.NatIPAddresses.IsNull() {
					natIPAddresses := []string{}
					natDiags := iface.NatIPAddresses.ElementsAs(ctx, &natIPAddresses, true)
					resp.Diagnostics = append(resp.Diagnostics, natDiags...)
					toAppend.NatIpAddresses = natIPAddresses
				}
				if !iface.Bfd.IsNull() {
					bfd := &bfdConfigModel{}
					bfdDiags := iface.Bfd.As(ctx, bfd, basetypes.ObjectAsOptions{})
					resp.Diagnostics = append(resp.Diagnostics, bfdDiags...)
					toAppend.Bfd = megaport.BfdConfig{
						TxInterval: int(bfd.TxInterval.ValueInt64()),
						RxInterval: int(bfd.RxInterval.ValueInt64()),
						Multiplier: int(bfd.Multiplier.ValueInt64()),
					}
				}
				if !iface.VLAN.IsNull() {
					toAppend.VLAN = int(iface.VLAN.ValueInt64())
				}
				if !iface.BgpConnections.IsNull() {
					bgpConnections := []*bgpConnectionConfigModel{}
					bgpDiags := iface.BgpConnections.ElementsAs(ctx, &bgpConnections, false)
					resp.Diagnostics = append(resp.Diagnostics, bgpDiags...)
					for _, bgpConnection := range bgpConnections {
						bgpToAppend := megaport.BgpConnectionConfig{
							PeerAsn:            int(bgpConnection.PeerAsn.ValueInt64()),
							LocalIpAddress:     bgpConnection.LocalIPAddress.ValueString(),
							PeerIpAddress:      bgpConnection.PeerIPAddress.ValueString(),
							Password:           bgpConnection.Password.ValueString(),
							Shutdown:           bgpConnection.Shutdown.ValueBool(),
							Description:        bgpConnection.Description.ValueString(),
							MedIn:              int(bgpConnection.MedIn.ValueInt64()),
							MedOut:             int(bgpConnection.MedOut.ValueInt64()),
							BfdEnabled:         bgpConnection.BfdEnabled.ValueBool(),
							ExportPolicy:       bgpConnection.ExportPolicy.ValueString(),
							AsPathPrependCount: int(bgpConnection.AsPathPrependCount.ValueInt64()),
							PeerType:           bgpConnection.PeerType.ValueString(),
						}
						if !bgpConnection.LocalAsn.IsNull() {
							bgpToAppend.LocalAsn = megaport.PtrTo(int(bgpConnection.LocalAsn.ValueInt64()))
						}
						if !bgpConnection.ImportWhitelist.IsNull() {
							for _, prefixFilterList := range prefixFilterListRes {
								if prefixFilterList.Description == bgpConnection.ImportWhitelist.ValueString() {
									bgpToAppend.ImportWhitelist = prefixFilterList.Id
								}
							}
						}
						if !bgpConnection.ImportBlacklist.IsNull() {
							for _, prefixFilterList := range prefixFilterListRes {
								if prefixFilterList.Description == bgpConnection.ImportBlacklist.ValueString() {
									bgpToAppend.ImportBlacklist = prefixFilterList.Id
								}
							}
						}
						if !bgpConnection.ExportWhitelist.IsNull() {
							for _, prefixFilterList := range prefixFilterListRes {
								if prefixFilterList.Description == bgpConnection.ExportWhitelist.ValueString() {
									bgpToAppend.ExportWhitelist = prefixFilterList.Id
								}
							}
						}
						if !bgpConnection.ExportBlacklist.IsNull() {
							for _, prefixFilterList := range prefixFilterListRes {
								if prefixFilterList.Description == bgpConnection.ExportBlacklist.ValueString() {
									bgpToAppend.ExportBlacklist = prefixFilterList.Id
								}
							}
						}
						if !bgpConnection.PermitExportTo.IsNull() {
							permitExportTo := []string{}
							permitDiags := bgpConnection.PermitExportTo.ElementsAs(ctx, &permitExportTo, true)
							resp.Diagnostics = append(resp.Diagnostics, permitDiags...)
							bgpToAppend.PermitExportTo = permitExportTo
							bgpToAppend.PermitExportTo = permitExportTo
						}
						if !bgpConnection.DenyExportTo.IsNull() {
							denyExportTo := []string{}
							denyDiags := bgpConnection.DenyExportTo.ElementsAs(ctx, &denyExportTo, true)
							resp.Diagnostics = append(resp.Diagnostics, denyDiags...)
							bgpToAppend.DenyExportTo = denyExportTo
						}
						toAppend.BgpConnections = append(toAppend.BgpConnections, bgpToAppend)
					}
				}
				bEndMegaportConfig.Interfaces = append(bEndMegaportConfig.Interfaces, toAppend)
			}
			vrouterConfigObj, bEndDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigVrouterAttrs, partnerConfigBEnd)
			resp.Diagnostics.Append(bEndDiags...)
			aws := types.ObjectNull(vxcPartnerConfigAWSAttrs)
			azure := types.ObjectNull(vxcPartnerConfigAzureAttrs)
			google := types.ObjectNull(vxcPartnerConfigGoogleAttrs)
			oracle := types.ObjectNull(vxcPartnerConfigOracleAttrs)
			aEndPartner := types.ObjectNull(vxcPartnerConfigAEndAttrs)
			ibmPartner := types.ObjectNull(vxcPartnerConfigIbmAttrs)
			bEndPartnerConfigModel := &vxcPartnerConfigurationModel{
				Partner:              bEndPartnerPlan.Partner,
				AWSPartnerConfig:     aws,
				AzurePartnerConfig:   azure,
				GooglePartnerConfig:  google,
				OraclePartnerConfig:  oracle,
				IBMPartnerConfig:     ibmPartner,
				VrouterPartnerConfig: vrouterConfigObj,
				PartnerAEndConfig:    aEndPartner,
			}
			bEndPartnerConfigObj, partnerDiags := types.ObjectValueFrom(ctx, vxcPartnerConfigAttrs, bEndPartnerConfigModel)
			resp.Diagnostics.Append(partnerDiags...)
			state.BEndPartnerConfig = bEndPartnerConfigObj
			updateReq.BEndPartnerConfig = bEndMegaportConfig
		default:
			resp.Diagnostics.AddError(
				"Error Updating VXC",
				"Could not update VXC with ID "+state.UID.ValueString()+": Partner configuration not supported",
			)
			return
		}
	}

	_, err := r.client.VXCService.UpdateVXC(ctx, plan.UID.ValueString(), updateReq)

	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating VXC",
			"Could not update VXC with ID "+state.UID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Update state with any changes from plan configuration following successful update
	aEndStateObj, aEndStateDiags := types.ObjectValueFrom(ctx, vxcBasicEndConfigurationAttrs, aEndState)
	resp.Diagnostics.Append(aEndStateDiags...)
	state.AEndConfiguration = aEndStateObj
	bEndStateObj, bEndStateDiags := types.ObjectValueFrom(ctx, vxcBasicEndConfigurationAttrs, bEndState)
	resp.Diagnostics.Append(bEndStateDiags...)
	state.BEndConfiguration = bEndStateObj

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
			var aEndCSP, bEndCSP bool
			aEndStateObj := state.AEndConfiguration
			bEndStateObj := state.BEndConfiguration
			aEndStateConfig := &vxcBasicEndConfigurationModel{}
			bEndStateConfig := &vxcBasicEndConfigurationModel{}
			aEndDiags := aEndStateObj.As(ctx, aEndStateConfig, basetypes.ObjectAsOptions{})
			bEndDiags := bEndStateObj.As(ctx, bEndStateConfig, basetypes.ObjectAsOptions{})
			diags = append(diags, aEndDiags...)
			diags = append(diags, bEndDiags...)
			aEndPlanObj := plan.AEndConfiguration
			bEndPlanObj := plan.BEndConfiguration
			aEndPlanConfig := &vxcBasicEndConfigurationModel{}
			bEndPlanConfig := &vxcBasicEndConfigurationModel{}
			aEndPartnerConfigModel := &vxcPartnerConfigurationModel{}
			bEndPartnerConfigModel := &vxcPartnerConfigurationModel{}
			aEndDiags = aEndPlanObj.As(ctx, aEndPlanConfig, basetypes.ObjectAsOptions{})
			bEndDiags = bEndPlanObj.As(ctx, bEndPlanConfig, basetypes.ObjectAsOptions{})
			diags = append(diags, aEndDiags...)
			diags = append(diags, bEndDiags...)
			if aEndStateConfig.OrderedVLAN.IsUnknown() {
				aEndPlanConfig.OrderedVLAN = aEndStateConfig.VLAN
			}
			if bEndStateConfig.OrderedVLAN.IsUnknown() {
				bEndPlanConfig.OrderedVLAN = bEndStateConfig.VLAN
			}
			partnerConfigDiags := plan.AEndPartnerConfig.As(ctx, &aEndPartnerConfigModel, basetypes.ObjectAsOptions{})
			diags = append(diags, partnerConfigDiags...)
			if !plan.AEndPartnerConfig.IsNull() {
				if !aEndPartnerConfigModel.Partner.IsNull() {
					if aEndPartnerConfigModel.Partner.ValueString() != "transit" && aEndPartnerConfigModel.Partner.ValueString() != "vrouter" && aEndPartnerConfigModel.Partner.ValueString() != "a-end" {
						aEndCSP = true
					}
				}
			}
			if state.AEndPartnerConfig.IsNull() {
				if !plan.AEndPartnerConfig.IsNull() {
					state.AEndPartnerConfig = plan.AEndPartnerConfig
				} else {
					state.AEndPartnerConfig = types.ObjectNull(vxcPartnerConfigAttrs)
				}
			} else {
				if !plan.AEndPartnerConfig.Equal(state.AEndPartnerConfig) && aEndCSP {
					resp.RequiresReplace = append(resp.RequiresReplace, path.Root("a_end_partner_config"))
				}
			}

			if aEndStateConfig.RequestedProductUID.IsNull() {
				if aEndPlanConfig.RequestedProductUID.IsNull() {
					aEndStateConfig.RequestedProductUID = aEndStateConfig.CurrentProductUID
					aEndPlanConfig.RequestedProductUID = aEndStateConfig.CurrentProductUID
				} else {
					aEndStateConfig.RequestedProductUID = aEndPlanConfig.RequestedProductUID
				}
			} else if aEndCSP {
				if !aEndPlanConfig.RequestedProductUID.IsNull() && !aEndPlanConfig.RequestedProductUID.Equal(aEndStateConfig.RequestedProductUID) {
					diags.AddWarning(
						"Cloud provider port mapping detected",
						fmt.Sprintf("Different A-End Product UIDs detected for cloud provider endpoint: requested=%s, actual=%s. This is normal - Megaport automatically manages cloud connection port assignments. Your configuration remains unchanged while the connection uses the provider-assigned Product UID. No action needed.",
							aEndPlanConfig.RequestedProductUID.ValueString(),
							aEndStateConfig.CurrentProductUID.ValueString()),
					)
				}
				aEndPlanConfig.RequestedProductUID = aEndStateConfig.RequestedProductUID
			}

			partnerConfigDiags = plan.BEndPartnerConfig.As(ctx, &bEndPartnerConfigModel, basetypes.ObjectAsOptions{})
			diags = append(diags, partnerConfigDiags...)
			if !plan.BEndPartnerConfig.IsNull() {
				if !bEndPartnerConfigModel.Partner.IsNull() {
					if !bEndPartnerConfigModel.Partner.IsNull() {
						if bEndPartnerConfigModel.Partner.ValueString() != "transit" && bEndPartnerConfigModel.Partner.ValueString() != "vrouter" && bEndPartnerConfigModel.Partner.ValueString() != "a-end" {
							bEndCSP = true
						}
					}
				}
			}

			if state.BEndPartnerConfig.IsNull() {
				if !plan.BEndPartnerConfig.IsNull() {
					state.BEndPartnerConfig = plan.BEndPartnerConfig
				} else {
					state.BEndPartnerConfig = types.ObjectNull(vxcPartnerConfigAttrs)
				}
			} else {
				if !plan.BEndPartnerConfig.Equal(state.BEndPartnerConfig) && bEndCSP {
					resp.RequiresReplace = append(resp.RequiresReplace, path.Root("b_end_partner_config"))
				}
			}

			if bEndStateConfig.RequestedProductUID.IsNull() {
				if bEndPlanConfig.RequestedProductUID.IsNull() {
					bEndStateConfig.RequestedProductUID = bEndStateConfig.CurrentProductUID
					bEndPlanConfig.RequestedProductUID = bEndStateConfig.CurrentProductUID
				} else {
					bEndStateConfig.RequestedProductUID = bEndPlanConfig.RequestedProductUID
				}
			} else if bEndCSP {
				if !bEndPlanConfig.RequestedProductUID.IsNull() && !bEndPlanConfig.RequestedProductUID.Equal(bEndStateConfig.CurrentProductUID) {
					diags.AddWarning(
						"Cloud provider port mapping detected",
						fmt.Sprintf("Different B-End Product UIDs detected for cloud provider endpoint: requested=%s, actual=%s. This is normal - Megaport automatically manages cloud connection port assignments. Your configuration remains unchanged while the connection uses the provider-assigned Product UID. No action needed.",
							bEndPlanConfig.RequestedProductUID.ValueString(),
							bEndStateConfig.CurrentProductUID.ValueString()),
					)
				}
				bEndPlanConfig.RequestedProductUID = bEndStateConfig.RequestedProductUID
			}

			newPlanAEndObj, aEndDiags := types.ObjectValueFrom(ctx, vxcBasicEndConfigurationAttrs, aEndPlanConfig)
			newPlanBEndObj, bEndDiags := types.ObjectValueFrom(ctx, vxcBasicEndConfigurationAttrs, bEndPlanConfig)
			diags = append(diags, aEndDiags...)
			diags = append(diags, bEndDiags...)
			plan.AEndConfiguration = newPlanAEndObj
			plan.BEndConfiguration = newPlanBEndObj
			newStateAEndObj, aEndDiags := types.ObjectValueFrom(ctx, vxcBasicEndConfigurationAttrs, aEndStateConfig)
			newStateBEndObj, bEndDiags := types.ObjectValueFrom(ctx, vxcBasicEndConfigurationAttrs, bEndStateConfig)
			diags = append(diags, aEndDiags...)
			diags = append(diags, bEndDiags...)
			state.AEndConfiguration = newStateAEndObj
			state.BEndConfiguration = newStateBEndObj
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
