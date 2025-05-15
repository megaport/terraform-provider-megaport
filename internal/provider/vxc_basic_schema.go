package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

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
			},
			"inner_vlan": schema.Int64Attribute{
				Description: fmt.Sprintf("The inner VLAN of the %s configuration. If the %s ordered_vlan is untagged and set as -1, this field cannot be set by the API, as the VLAN of the %s is designated as untagged.", end, end, end),
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
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
