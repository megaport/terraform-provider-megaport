package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	usersEmailAttrs = map[string]attr.Type{
		"email_address_id": types.Int64Type,
		"email":            types.StringType,
		"primary":          types.BoolType,
		"bad_email":        types.BoolType,
		"bad_email_type":   types.StringType,
		"bad_email_reason": types.StringType,
	}

	usersUserAttrs = map[string]attr.Type{
		"employee_id":                  types.Int64Type,
		"party_id":                     types.Int64Type,
		"first_name":                   types.StringType,
		"last_name":                    types.StringType,
		"email":                        types.StringType,
		"phone":                        types.StringType,
		"mobile":                       types.StringType,
		"position":                     types.StringType,
		"salutation":                   types.StringType,
		"username":                     types.StringType,
		"description":                  types.StringType,
		"active":                       types.BoolType,
		"uid":                          types.StringType,
		"salesforce_id":                types.StringType,
		"channel_manager":              types.BoolType,
		"require_totp":                 types.BoolType,
		"notification_enabled":         types.BoolType,
		"newsletter":                   types.BoolType,
		"promotions":                   types.BoolType,
		"mfa_enabled":                  types.BoolType,
		"confirmation_pending":         types.BoolType,
		"invitation_pending":           types.BoolType,
		"name":                         types.StringType,
		"receives_child_notifications": types.BoolType,
		"security_roles":               types.ListType{}.WithElementType(types.StringType),
		"feature_flags":                types.ListType{}.WithElementType(types.StringType),
		"emails":                       types.ListType{}.WithElementType(types.ObjectType{}.WithAttributeTypes(usersEmailAttrs)),
		"company_id":                   types.Int64Type,
		"employment_id":                types.Int64Type,
		"position_id":                  types.Int64Type,
		"person_alt_id":                types.StringType,
		"employment_type":              types.StringType,
		"company_name":                 types.StringType,
	}
)

func usersDataSourceSchema() schema.Schema {
	return schema.Schema{
		Description: "Users data source for Megaport. Returns a list of all users in the company. Use this data source to retrieve information about all users associated with your company account. You can filter the results by various user attributes.",
		Attributes: map[string]schema.Attribute{
			"employee_id": &schema.Int64Attribute{
				Description: "Filter users by employee ID. If specified, only the user with this employee ID will be returned.",
				Optional:    true,
				Computed:    true,
			},
			"first_name": &schema.StringAttribute{
				Description: "Filter users by first name (exact match, case-sensitive).",
				Optional:    true,
				Computed:    true,
			},
			"last_name": &schema.StringAttribute{
				Description: "Filter users by last name (exact match, case-sensitive).",
				Optional:    true,
				Computed:    true,
			},
			"email": &schema.StringAttribute{
				Description: "Filter users by email address (exact match, case-insensitive).",
				Optional:    true,
				Computed:    true,
			},
			"phone": &schema.StringAttribute{
				Description: "Filter users by phone number (exact match).",
				Optional:    true,
				Computed:    true,
			},
			"position": &schema.StringAttribute{
				Description: "Filter users by position/role (exact match, case-sensitive).",
				Optional:    true,
				Computed:    true,
			},
			"uid": &schema.StringAttribute{
				Description: "Filter users by UID (exact match).",
				Optional:    true,
				Computed:    true,
			},
			"name": &schema.StringAttribute{
				Description: "Filter users by full name (exact match, case-sensitive).",
				Optional:    true,
				Computed:    true,
			},
			"users": &schema.ListNestedAttribute{
				Description: "List of all users in the company.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"employee_id": &schema.Int64Attribute{
							Description: "The employee ID of the user (employment ID from the API).",
							Computed:    true,
						},
						"party_id": &schema.Int64Attribute{
							Description: "The party ID of the user (person ID from the API).",
							Computed:    true,
						},
						"first_name": &schema.StringAttribute{
							Description: "The first name of the user.",
							Computed:    true,
						},
						"last_name": &schema.StringAttribute{
							Description: "The last name of the user.",
							Computed:    true,
						},
						"email": &schema.StringAttribute{
							Description: "The primary email address of the user.",
							Computed:    true,
						},
						"phone": &schema.StringAttribute{
							Description: "The phone number of the user.",
							Computed:    true,
						},
						"mobile": &schema.StringAttribute{
							Description: "The mobile number of the user.",
							Computed:    true,
						},
						"position": &schema.StringAttribute{
							Description: "The position/role of the user within the organization.",
							Computed:    true,
						},
						"salutation": &schema.StringAttribute{
							Description: "The salutation of the user (e.g., Mr., Ms., Dr.).",
							Computed:    true,
						},
						"username": &schema.StringAttribute{
							Description: "The username of the user.",
							Computed:    true,
						},
						"description": &schema.StringAttribute{
							Description: "A description or additional information about the user.",
							Computed:    true,
						},
						"active": &schema.BoolAttribute{
							Description: "Whether the user account is active.",
							Computed:    true,
						},
						"uid": &schema.StringAttribute{
							Description: "The unique identifier (UID) of the user.",
							Computed:    true,
						},
						"salesforce_id": &schema.StringAttribute{
							Description: "The Salesforce ID associated with the user.",
							Computed:    true,
						},
						"channel_manager": &schema.BoolAttribute{
							Description: "Whether the user is a channel manager.",
							Computed:    true,
						},
						"require_totp": &schema.BoolAttribute{
							Description: "Whether the user is required to use TOTP (Time-based One-Time Password) for authentication.",
							Computed:    true,
						},
						"notification_enabled": &schema.BoolAttribute{
							Description: "Whether notifications are enabled for the user.",
							Computed:    true,
						},
						"newsletter": &schema.BoolAttribute{
							Description: "Whether the user has opted in to receive newsletters.",
							Computed:    true,
						},
						"promotions": &schema.BoolAttribute{
							Description: "Whether the user has opted in to receive promotional communications.",
							Computed:    true,
						},
						"mfa_enabled": &schema.BoolAttribute{
							Description: "Whether multi-factor authentication (MFA) is enabled for the user.",
							Computed:    true,
						},
						"confirmation_pending": &schema.BoolAttribute{
							Description: "Whether the user has a pending confirmation (e.g., email verification).",
							Computed:    true,
						},
						"invitation_pending": &schema.BoolAttribute{
							Description: "Whether the user has a pending invitation.",
							Computed:    true,
						},
						"name": &schema.StringAttribute{
							Description: "The full name of the user.",
							Computed:    true,
						},
						"receives_child_notifications": &schema.BoolAttribute{
							Description: "Whether the user receives notifications for child entities.",
							Computed:    true,
						},
						"security_roles": &schema.ListAttribute{
							Description: "List of security roles assigned to the user.",
							Computed:    true,
							ElementType: types.StringType,
						},
						"feature_flags": &schema.ListAttribute{
							Description: "List of feature flags enabled for the user.",
							Computed:    true,
							ElementType: types.StringType,
						},
						"emails": &schema.ListNestedAttribute{
							Description: "List of email addresses associated with the user.",
							Computed:    true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"email_address_id": &schema.Int64Attribute{
										Description: "The unique identifier for the email address.",
										Computed:    true,
									},
									"email": &schema.StringAttribute{
										Description: "The email address.",
										Computed:    true,
									},
									"primary": &schema.BoolAttribute{
										Description: "Whether this is the primary email address.",
										Computed:    true,
									},
									"bad_email": &schema.BoolAttribute{
										Description: "Whether the email address is marked as bad (bounced, invalid, etc.).",
										Computed:    true,
									},
									"bad_email_type": &schema.StringAttribute{
										Description: "The type of bad email issue if applicable.",
										Computed:    true,
									},
									"bad_email_reason": &schema.StringAttribute{
										Description: "The reason why the email is marked as bad if applicable.",
										Computed:    true,
									},
								},
							},
						},
						"company_id": &schema.Int64Attribute{
							Description: "The company ID associated with the user.",
							Computed:    true,
						},
						"employment_id": &schema.Int64Attribute{
							Description: "The employment ID of the user.",
							Computed:    true,
						},
						"position_id": &schema.Int64Attribute{
							Description: "The position ID associated with the user's role.",
							Computed:    true,
						},
						"person_alt_id": &schema.StringAttribute{
							Description: "An alternative person identifier for the user.",
							Computed:    true,
						},
						"employment_type": &schema.StringAttribute{
							Description: "The employment type of the user.",
							Computed:    true,
						},
						"company_name": &schema.StringAttribute{
							Description: "The name of the company associated with the user.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}
