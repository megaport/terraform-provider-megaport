package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	megaport "github.com/megaport/megaportgo"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &usersDataSource{}
	_ datasource.DataSourceWithConfigure = &usersDataSource{}
)

// usersDataSource is the data source implementation.
type usersDataSource struct {
	client *megaport.Client
}

// usersModel maps the data source schema data.
type usersModel struct {
	EmployeeID types.Int64  `tfsdk:"employee_id"`
	FirstName  types.String `tfsdk:"first_name"`
	LastName   types.String `tfsdk:"last_name"`
	Email      types.String `tfsdk:"email"`
	Phone      types.String `tfsdk:"phone"`
	Position   types.String `tfsdk:"position"`
	UID        types.String `tfsdk:"uid"`
	Name       types.String `tfsdk:"name"`
	Users      types.List   `tfsdk:"users"`
}

// usersUserModel maps the individual user data.
type usersUserModel struct {
	EmployeeID                 types.Int64  `tfsdk:"employee_id"`
	PartyID                    types.Int64  `tfsdk:"party_id"`
	FirstName                  types.String `tfsdk:"first_name"`
	LastName                   types.String `tfsdk:"last_name"`
	Email                      types.String `tfsdk:"email"`
	Phone                      types.String `tfsdk:"phone"`
	Mobile                     types.String `tfsdk:"mobile"`
	Position                   types.String `tfsdk:"position"`
	Salutation                 types.String `tfsdk:"salutation"`
	Username                   types.String `tfsdk:"username"`
	Description                types.String `tfsdk:"description"`
	Active                     types.Bool   `tfsdk:"active"`
	UID                        types.String `tfsdk:"uid"`
	SalesforceID               types.String `tfsdk:"salesforce_id"`
	ChannelManager             types.Bool   `tfsdk:"channel_manager"`
	RequireTotp                types.Bool   `tfsdk:"require_totp"`
	NotificationEnabled        types.Bool   `tfsdk:"notification_enabled"`
	Newsletter                 types.Bool   `tfsdk:"newsletter"`
	Promotions                 types.Bool   `tfsdk:"promotions"`
	MfaEnabled                 types.Bool   `tfsdk:"mfa_enabled"`
	ConfirmationPending        types.Bool   `tfsdk:"confirmation_pending"`
	InvitationPending          types.Bool   `tfsdk:"invitation_pending"`
	Name                       types.String `tfsdk:"name"`
	ReceivesChildNotifications types.Bool   `tfsdk:"receives_child_notifications"`
	SecurityRoles              types.List   `tfsdk:"security_roles"`
	FeatureFlags               types.List   `tfsdk:"feature_flags"`
	Emails                     types.List   `tfsdk:"emails"`
	CompanyID                  types.Int64  `tfsdk:"company_id"`
	EmploymentID               types.Int64  `tfsdk:"employment_id"`
	PositionID                 types.Int64  `tfsdk:"position_id"`
	PersonAltID                types.String `tfsdk:"person_alt_id"`
	EmploymentType             types.String `tfsdk:"employment_type"`
	CompanyName                types.String `tfsdk:"company_name"`
}

// usersEmailModel represents an email address associated with a user
type usersEmailModel struct {
	EmailAddressID types.Int64  `tfsdk:"email_address_id"`
	Email          types.String `tfsdk:"email"`
	Primary        types.Bool   `tfsdk:"primary"`
	BadEmail       types.Bool   `tfsdk:"bad_email"`
	BadEmailType   types.String `tfsdk:"bad_email_type"`
	BadEmailReason types.String `tfsdk:"bad_email_reason"`
}

func (m *usersUserModel) fromAPIUser(ctx context.Context, u *megaport.User) diag.Diagnostics {
	diags := diag.Diagnostics{}

	// Map API fields to model fields
	// Use EmploymentId as employee_id since that's what's used in the resource
	m.EmployeeID = types.Int64Value(int64(u.EmploymentId))
	// Use PersonId as party_id if available, otherwise use PartyId
	if u.PersonId > 0 {
		m.PartyID = types.Int64Value(int64(u.PersonId))
	} else {
		m.PartyID = types.Int64Value(int64(u.PartyId))
	}
	m.FirstName = types.StringValue(u.FirstName)
	m.LastName = types.StringValue(u.LastName)
	m.Email = types.StringValue(u.Email)
	m.Phone = types.StringValue(u.Phone)
	m.Mobile = types.StringValue(u.Mobile)
	m.Position = types.StringValue(u.Position)
	m.Salutation = types.StringValue(u.Salutation)
	m.Username = types.StringValue(u.Username)
	m.Description = types.StringValue(u.Description)
	m.Active = types.BoolValue(u.Active)
	// Use PersonUid if available, otherwise use UID
	if u.PersonUid != "" {
		m.UID = types.StringValue(u.PersonUid)
	} else {
		m.UID = types.StringValue(u.UID)
	}
	m.SalesforceID = types.StringValue(u.SalesforceId)
	m.ChannelManager = types.BoolValue(u.ChannelManager)
	m.RequireTotp = types.BoolValue(u.RequireTotp)
	m.NotificationEnabled = types.BoolValue(u.NotificationEnabled)
	m.Newsletter = types.BoolValue(u.Newsletter)
	m.Promotions = types.BoolValue(u.Promotions)
	m.MfaEnabled = types.BoolValue(u.MfaEnabled)
	m.ConfirmationPending = types.BoolValue(u.ConfirmationPending)
	m.InvitationPending = types.BoolValue(u.InvitationPending)
	m.Name = types.StringValue(u.Name)
	m.ReceivesChildNotifications = types.BoolValue(u.ReceivesChildNotifications)
	m.CompanyID = types.Int64Value(int64(u.CompanyId))
	m.EmploymentID = types.Int64Value(int64(u.EmploymentId))
	m.PositionID = types.Int64Value(int64(u.PositionId))
	m.PersonAltID = types.StringValue(u.PersonAltId)
	m.EmploymentType = types.StringValue(u.EmploymentType)
	m.CompanyName = types.StringValue(u.CompanyName)

	// Convert security roles list
	if len(u.SecurityRoles) > 0 {
		securityRolesList, securityRolesDiags := types.ListValueFrom(ctx, types.StringType, u.SecurityRoles)
		diags = append(diags, securityRolesDiags...)
		m.SecurityRoles = securityRolesList
	} else {
		m.SecurityRoles = types.ListNull(types.StringType)
	}

	// Convert feature flags list
	if len(u.FeatureFlags) > 0 {
		featureFlagsList, featureFlagsDiags := types.ListValueFrom(ctx, types.StringType, u.FeatureFlags)
		diags = append(diags, featureFlagsDiags...)
		m.FeatureFlags = featureFlagsList
	} else {
		m.FeatureFlags = types.ListNull(types.StringType)
	}

	// Convert emails list
	if len(u.Emails) > 0 {
		emailModels := make([]usersEmailModel, len(u.Emails))
		for i, email := range u.Emails {
			emailModels[i] = usersEmailModel{
				EmailAddressID: types.Int64Value(int64(email.EmailAddressId)),
				Email:          types.StringValue(email.Email),
				Primary:        types.BoolValue(email.Primary),
				BadEmail:       types.BoolValue(email.BadEmail),
			}
			if email.BadEmailType != nil {
				emailModels[i].BadEmailType = types.StringValue(*email.BadEmailType)
			} else {
				emailModels[i].BadEmailType = types.StringNull()
			}
			if email.BadEmailReason != nil {
				emailModels[i].BadEmailReason = types.StringValue(*email.BadEmailReason)
			} else {
				emailModels[i].BadEmailReason = types.StringNull()
			}
		}
		emailsList, emailsDiags := types.ListValueFrom(ctx, types.ObjectType{}.WithAttributeTypes(usersEmailAttrs), emailModels)
		diags = append(diags, emailsDiags...)
		m.Emails = emailsList
	} else {
		m.Emails = types.ListNull(types.ObjectType{}.WithAttributeTypes(usersEmailAttrs))
	}

	return diags
}

// filterUsers applies the configured filters to the list of users
func (d *usersDataSource) filterUsers(users []*megaport.User, filters usersModel) []*megaport.User {
	var filteredUsers []*megaport.User

	for _, user := range users {
		matchesFilters := true

		// Filter by employee_id (using EmploymentId from API)
		if !filters.EmployeeID.IsNull() && !filters.EmployeeID.IsUnknown() {
			if int64(user.EmploymentId) != filters.EmployeeID.ValueInt64() {
				matchesFilters = false
			}
		}

		// Filter by first_name (exact match, case-sensitive)
		if !filters.FirstName.IsNull() && !filters.FirstName.IsUnknown() {
			if user.FirstName != filters.FirstName.ValueString() {
				matchesFilters = false
			}
		}

		// Filter by last_name (exact match, case-sensitive)
		if !filters.LastName.IsNull() && !filters.LastName.IsUnknown() {
			if user.LastName != filters.LastName.ValueString() {
				matchesFilters = false
			}
		}

		// Filter by email (exact match, case-insensitive)
		if !filters.Email.IsNull() && !filters.Email.IsUnknown() {
			if !strings.EqualFold(user.Email, filters.Email.ValueString()) {
				matchesFilters = false
			}
		}

		// Filter by phone (exact match)
		if !filters.Phone.IsNull() && !filters.Phone.IsUnknown() {
			if user.Phone != filters.Phone.ValueString() {
				matchesFilters = false
			}
		}

		// Filter by position (exact match, case-sensitive)
		if !filters.Position.IsNull() && !filters.Position.IsUnknown() {
			if user.Position != filters.Position.ValueString() {
				matchesFilters = false
			}
		}

		// Filter by uid (exact match, use PersonUid if available, otherwise UID)
		if !filters.UID.IsNull() && !filters.UID.IsUnknown() {
			userUID := user.PersonUid
			if userUID == "" {
				userUID = user.UID
			}
			if userUID != filters.UID.ValueString() {
				matchesFilters = false
			}
		}

		// Filter by name (exact match, case-sensitive)
		if !filters.Name.IsNull() && !filters.Name.IsUnknown() {
			if user.Name != filters.Name.ValueString() {
				matchesFilters = false
			}
		}

		if matchesFilters {
			filteredUsers = append(filteredUsers, user)
		}
	}

	return filteredUsers
}

// NewUsersDataSource is a helper function to simplify the provider implementation.
func NewUsersDataSource() datasource.DataSource {
	return &usersDataSource{}
}

// Metadata returns the data source type name.
func (d *usersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_users"
}

// Schema defines the schema for the data source.
func (d *usersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = usersDataSourceSchema()
}

// Read refreshes the Terraform state with the latest data.
func (d *usersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config usersModel

	// Read configuration
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get users from API
	users, err := d.client.UserManagementService.ListCompanyUsers(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to list company users",
			err.Error(),
		)
		return
	}

	// Apply filters if any are specified
	filteredUsers := d.filterUsers(users, config)

	// Convert API users to model users
	userModels := make([]usersUserModel, len(filteredUsers))
	for i, user := range filteredUsers {
		userDiags := userModels[i].fromAPIUser(ctx, user)
		resp.Diagnostics.Append(userDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Convert user models to Terraform list
	usersList, usersDiags := types.ListValueFrom(ctx, types.ObjectType{}.WithAttributeTypes(usersUserAttrs), userModels)
	resp.Diagnostics.Append(usersDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	config.Users = usersList

	// Set state
	setDiags := resp.State.Set(ctx, &config)
	resp.Diagnostics.Append(setDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Configure adds the provider configured client to the data source.
func (d *usersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	data, ok := req.ProviderData.(*megaportProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Provider Data Type",
			fmt.Sprintf("Expected *megaportProviderData, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	client := data.client
	d.client = client
}
