package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	serviceKeyValidForAttrs = map[string]attr.Type{
		"start_time": types.StringType,
		"end_time":   types.StringType,
	}
)

func serviceKeyResourceSchema() schema.Schema {
	return schema.Schema{
		Description: "Service Key Resource for the Megaport Terraform Provider. " +
			"This resource allows you to create and manage service keys that enable " +
			"other parties to create VXC connections to your ports.",
		Attributes: map[string]schema.Attribute{
			"product_uid": schema.StringAttribute{
				Description: "The UID of the port that this service key is associated with.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"max_speed": schema.Int64Attribute{
				Description: "The maximum speed in Mbps that the service key allows.",
				Required:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"single_use": schema.BoolAttribute{
				Description: "Whether the service key is single-use (true) or multi-use (false). " +
					"With a multi-use key, the other party can request multiple connections using the key.",
				Required: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"active": schema.BoolAttribute{
				Description: "Whether the service key is currently active and available for use.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "A description for the service key.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"vlan": schema.Int64Attribute{
				Description: "The VLAN ID for the service key. Required when single_use is true.",
				Optional:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"pre_approved": schema.BoolAttribute{
				Description: "Whether the service key is pre-approved for use.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"valid_for": schema.SingleNestedAttribute{
				Description: "The date range for which the service key is valid.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"start_time": schema.StringAttribute{
						Description: "The start time for the service key validity in RFC3339 format.",
						Optional:    true,
						Computed:    true,
					},
					"end_time": schema.StringAttribute{
						Description: "The end time for the service key validity in RFC3339 format.",
						Optional:    true,
						Computed:    true,
					},
				},
			},
			"key": schema.StringAttribute{
				Description: "The service key value. This is the secret key that is shared with the other party.",
				Computed:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"company_id": schema.Int64Attribute{
				Description: "The numeric company ID of the service key owner.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"company_uid": schema.StringAttribute{
				Description: "The UID of the company that owns the service key.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"create_date": schema.StringAttribute{
				Description: "The date and time when the service key was created.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"last_used": schema.StringAttribute{
				Description: "The date and time when the service key was last used.",
				Computed:    true,
			},
			"expired": schema.BoolAttribute{
				Description: "Whether the service key has expired.",
				Computed:    true,
			},
			"valid": schema.BoolAttribute{
				Description: "Whether the service key is currently valid.",
				Computed:    true,
			},
			"last_updated": schema.StringAttribute{
				Description: "The timestamp of the last Terraform update of the resource.",
				Computed:    true,
			},
		},
	}
}
