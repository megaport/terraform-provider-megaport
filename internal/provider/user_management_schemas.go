package provider

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	userEmailAttrs = map[string]attr.Type{
		"email_address_id": types.Int64Type,
		"email":            types.StringType,
		"primary":          types.BoolType,
		"bad_email":        types.BoolType,
		"bad_email_type":   types.StringType,
		"bad_email_reason": types.StringType,
	}
)

func userSchema() schema.Schema {
	return schema.Schema{
		Description: "Megaport User resource for managing users in your company.",

		Attributes: map[string]schema.Attribute{
			"last_updated": schema.StringAttribute{
				Computed: true,
			},
			"employee_id": schema.Int64Attribute{
				Description: "The employee ID of the user.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"party_id": schema.Int64Attribute{
				Description: "The party ID of the user.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"first_name": schema.StringAttribute{
				Description: "The first name of the user.",
				Required:    true,
			},
			"last_name": schema.StringAttribute{
				Description: "The last name of the user.",
				Required:    true,
			},
			"email": schema.StringAttribute{
				Description: "The primary email address of the user.",
				Required:    true,
			},
			"phone": schema.StringAttribute{
				Description: "The phone number of the user.",
				Optional:    true,
				Computed:    true,
			},
			"mobile": schema.StringAttribute{
				Description: "The mobile phone number of the user.",
				Optional:    true,
				Computed:    true,
			},
			"position": schema.StringAttribute{
				Description: "The position/role of the user in the organization.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(
						"Company Admin",
						"Technical Admin",
						"Technical Contact",
						"Finance",
						"Financial Contact",
						"Read Only",
					),
				},
			},
			"salutation": schema.StringAttribute{
				Description: "The salutation for the user.",
				Optional:    true,
				Computed:    true,
			},
			"username": schema.StringAttribute{
				Description: "The username for the user.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"description": schema.StringAttribute{
				Description: "A description of the user.",
				Optional:    true,
				Computed:    true,
			},
			"active": schema.BoolAttribute{
				Description: "Whether the user account is active.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"uid": schema.StringAttribute{
				Description: "The unique identifier for the user.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"salesforce_id": schema.StringAttribute{
				Description: "The Salesforce ID associated with the user.",
				Optional:    true,
				Computed:    true,
			},
			"channel_manager": schema.BoolAttribute{
				Description: "Whether the user is a channel manager.",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"require_totp": schema.BoolAttribute{
				Description: "Whether the user requires TOTP (Time-based One-Time Password) for authentication.",
				Optional:    true,
				Computed:    true,
			},
			"notification_enabled": schema.BoolAttribute{
				Description: "Whether notifications are enabled for the user.",
				Optional:    true,
				Computed:    true,
			},
			"newsletter": schema.BoolAttribute{
				Description: "Whether the user has opted into the newsletter.",
				Optional:    true,
				Computed:    true,
			},
			"promotions": schema.BoolAttribute{
				Description: "Whether the user has opted into promotional communications.",
				Optional:    true,
				Computed:    true,
			},
			"mfa_enabled": schema.BoolAttribute{
				Description: "Whether multi-factor authentication is enabled for the user.",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"confirmation_pending": schema.BoolAttribute{
				Description: "Whether the user has a pending confirmation.",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The full name of the user.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"receives_child_notifications": schema.BoolAttribute{
				Description: "Whether the user receives notifications for child entities.",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"security_roles": schema.ListAttribute{
				Description: "List of security roles assigned to the user.",
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"feature_flags": schema.ListAttribute{
				Description: "List of feature flags enabled for the user.",
				ElementType: types.StringType,
				Computed:    true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"emails": schema.ListNestedAttribute{
				Description: "List of email addresses associated with the user.",
				Computed:    true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"email_address_id": schema.Int64Attribute{
							Description: "The ID of the email address.",
							Computed:    true,
						},
						"email": schema.StringAttribute{
							Description: "The email address.",
							Computed:    true,
						},
						"primary": schema.BoolAttribute{
							Description: "Whether this is the primary email address.",
							Computed:    true,
						},
						"bad_email": schema.BoolAttribute{
							Description: "Whether this email address is marked as bad.",
							Computed:    true,
						},
						"bad_email_type": schema.StringAttribute{
							Description: "The type of bad email if applicable.",
							Computed:    true,
						},
						"bad_email_reason": schema.StringAttribute{
							Description: "The reason the email is marked as bad if applicable.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}
