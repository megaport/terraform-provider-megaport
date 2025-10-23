package provider

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	// mcrPrefixFilterListEntryAttributes defines the attribute types for prefix list entries in the standalone resource
	mcrPrefixFilterListEntryAttributes = map[string]attr.Type{
		"action": types.StringType,
		"prefix": types.StringType,
		"ge":     types.Int64Type,
		"le":     types.Int64Type,
	}
)

// mcrPrefixFilterListResourceSchema returns the schema for the MCR prefix filter list resource
func mcrPrefixFilterListResourceSchema() schema.Schema {
	return schema.Schema{
		Description: "MCR Prefix Filter List Resource for the Megaport Terraform Provider. " +
			"This resource manages individual prefix filter lists for MCR instances, " +
			"providing better resource management compared to inline prefix_filter_lists.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "Numeric ID of the prefix filter list.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"mcr_id": schema.StringAttribute{
				Description: "The UID of the MCR instance this prefix filter list belongs to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Description: "Description of the prefix filter list.",
				Required:    true,
			},
			"address_family": schema.StringAttribute{
				Description: "The IP address standard of the IP network addresses in the prefix filter list. " +
					"Valid values are 'IPv4' and 'IPv6' (case-insensitive).",
				Required: true,
				Validators: []validator.String{
					stringvalidator.OneOfCaseInsensitive("IPv4", "IPv6"),
				},
			},
			"entries": schema.ListNestedAttribute{
				Description: "Entries in the prefix filter list. Must contain between 1 and 200 entries.",
				Required:    true,
				Validators: []validator.List{
					listvalidator.SizeBetween(1, 200),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"action": schema.StringAttribute{
							Description: "The action to take for the network address in the filter list. " +
								"Valid values are 'permit' and 'deny'.",
							Required: true,
							Validators: []validator.String{
								stringvalidator.OneOf("permit", "deny"),
							},
						},
						"prefix": schema.StringAttribute{
							Description: "The network address of the prefix filter list entry in CIDR notation " +
								"(e.g., '10.0.1.0/24').",
							Required: true,
						},
						"ge": schema.Int64Attribute{
							Description: "The minimum starting prefix length to be matched. " +
								"Valid values are from 0 to 32 (IPv4), or 0 to 128 (IPv6). " +
								"If not specified, defaults to the prefix length of the network address.",
							Optional: true,
							Validators: []validator.Int64{
								int64validator.Between(0, 128),
							},
						},
						"le": schema.Int64Attribute{
							Description: "The maximum ending prefix length to be matched. " +
								"Valid values are from 0 to 32 (IPv4), or 0 to 128 (IPv6). " +
								"Must be greater than or equal to 'ge'. " +
								"If not specified, defaults to 32 (IPv4) or 128 (IPv6).",
							Optional: true,
							Validators: []validator.Int64{
								int64validator.Between(0, 128),
							},
						},
					},
				},
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of when the resource was last updated.",
				Computed:    true,
			},
		},
	}
}
