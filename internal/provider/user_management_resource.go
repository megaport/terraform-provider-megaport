package provider

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	megaport "github.com/megaport/megaportgo"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &userResource{}
	_ resource.ResourceWithConfigure   = &userResource{}
	_ resource.ResourceWithImportState = &userResource{}
)

// userResourceModel maps the resource schema data.
type userResourceModel struct {
	LastUpdated types.String `tfsdk:"last_updated"`

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
}

// userEmailModel represents an email address associated with a user
type userEmailModel struct {
	EmailAddressID types.Int64  `tfsdk:"email_address_id"`
	Email          types.String `tfsdk:"email"`
	Primary        types.Bool   `tfsdk:"primary"`
	BadEmail       types.Bool   `tfsdk:"bad_email"`
	BadEmailType   types.String `tfsdk:"bad_email_type"`
	BadEmailReason types.String `tfsdk:"bad_email_reason"`
}

func (orm *userResourceModel) fromAPIUser(ctx context.Context, u *megaport.User) diag.Diagnostics {
	diags := diag.Diagnostics{}

	// Set the last updated timestamp
	orm.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	// Basic user information
	orm.PartyID = types.Int64Value(int64(u.PartyId))
	orm.FirstName = types.StringValue(u.FirstName)
	orm.LastName = types.StringValue(u.LastName)
	orm.Email = types.StringValue(u.Email)
	orm.Phone = types.StringValue(u.Phone)
	orm.Mobile = types.StringValue(u.Mobile)
	orm.Position = types.StringValue(u.Position)
	orm.Salutation = types.StringValue(u.Salutation)
	orm.Username = types.StringValue(u.Username)
	orm.Description = types.StringValue(u.Description)
	orm.Active = types.BoolValue(u.Active)
	orm.UID = types.StringValue(u.UID)
	orm.SalesforceID = types.StringValue(u.SalesforceId)
	orm.ChannelManager = types.BoolValue(u.ChannelManager)
	orm.RequireTotp = types.BoolValue(u.RequireTotp)
	orm.NotificationEnabled = types.BoolValue(u.NotificationEnabled)
	orm.Newsletter = types.BoolValue(u.Newsletter)
	orm.Promotions = types.BoolValue(u.Promotions)
	orm.MfaEnabled = types.BoolValue(u.MfaEnabled)
	orm.ConfirmationPending = types.BoolValue(u.ConfirmationPending)
	orm.InvitationPending = types.BoolValue(u.InvitationPending)
	orm.Name = types.StringValue(u.Name)
	orm.ReceivesChildNotifications = types.BoolValue(u.ReceivesChildNotifications)

	// Convert security roles list
	if len(u.SecurityRoles) > 0 {
		securityRoles := make([]attr.Value, len(u.SecurityRoles))
		for i, role := range u.SecurityRoles {
			securityRoles[i] = types.StringValue(role)
		}
		securityRolesList, securityRolesDiags := types.ListValue(types.StringType, securityRoles)
		diags = append(diags, securityRolesDiags...)
		orm.SecurityRoles = securityRolesList
	} else {
		orm.SecurityRoles = types.ListNull(types.StringType)
	}

	// Convert feature flags list
	if len(u.FeatureFlags) > 0 {
		featureFlags := make([]attr.Value, len(u.FeatureFlags))
		for i, flag := range u.FeatureFlags {
			featureFlags[i] = types.StringValue(flag)
		}
		featureFlagsList, featureFlagsDiags := types.ListValue(types.StringType, featureFlags)
		diags = append(diags, featureFlagsDiags...)
		orm.FeatureFlags = featureFlagsList
	} else {
		orm.FeatureFlags = types.ListNull(types.StringType)
	}

	// Convert emails list
	if len(u.Emails) > 0 {
		emailModels := make([]userEmailModel, len(u.Emails))
		for i, email := range u.Emails {
			emailModels[i] = userEmailModel{
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

		emailValues := make([]attr.Value, len(emailModels))
		for i, emailModel := range emailModels {
			emailObject, emailObjectDiags := types.ObjectValueFrom(ctx, userEmailAttrs, emailModel)
			diags = append(diags, emailObjectDiags...)
			emailValues[i] = emailObject
		}

		emailsList, emailsListDiags := types.ListValue(
			types.ObjectType{AttrTypes: userEmailAttrs},
			emailValues,
		)
		diags = append(diags, emailsListDiags...)
		orm.Emails = emailsList
	} else {
		orm.Emails = types.ListNull(types.ObjectType{AttrTypes: userEmailAttrs})
	}

	return diags
}

// NewUserResource is a helper function to simplify the provider implementation.
func NewUserResource() resource.Resource {
	return &userResource{}
}

// userResource is the resource implementation.
type userResource struct {
	client *megaport.Client
}

// Metadata returns the resource type name.
func (r *userResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

// Schema defines the schema for the resource.
func (r *userResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = userSchema()
}

// Create a new resource.
func (r *userResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan userResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create the user request
	createReq := &megaport.CreateUserRequest{
		FirstName: plan.FirstName.ValueString(),
		LastName:  plan.LastName.ValueString(),
		Email:     plan.Email.ValueString(),
		Active:    true, // Default to active when creating
		Position:  megaport.UserPosition(plan.Position.ValueString()),
	}

	if !plan.Phone.IsNull() {
		createReq.Phone = plan.Phone.ValueString()
	}

	if !plan.Active.IsNull() {
		createReq.Active = plan.Active.ValueBool()
	}

	// Create the user
	userMgmt := r.client.UserManagementService
	createResp, err := userMgmt.CreateUser(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating user",
			"Could not create user: "+err.Error(),
		)
		return
	}

	// Set the employee ID from the create response
	plan.EmployeeID = types.Int64Value(int64(createResp.EmployeeID))

	// Get the created user details
	user, err := userMgmt.GetUser(ctx, createResp.EmployeeID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading created user",
			"Could not read user after creation: "+err.Error(),
		)
		return
	}

	// Set the employee ID from the create response
	plan.EmployeeID = types.Int64Value(int64(createResp.EmployeeID))

	// Update the plan with the user info
	diags = plan.fromAPIUser(ctx, user)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Read resource information.
func (r *userResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state userResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get refreshed user value from API
	userMgmt := r.client.UserManagementService
	user, err := userMgmt.GetUser(ctx, int(state.EmployeeID.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading user",
			"Could not read user with ID "+state.EmployeeID.String()+": "+err.Error(),
		)
		return
	}

	// Update the state with the user info
	diags = state.fromAPIUser(ctx, user)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *userResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Retrieve values from plan and current state
	var plan userResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	var state userResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	userToUpdate, err := r.client.UserManagementService.GetUser(ctx, int(state.EmployeeID.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading user",
			"Could not read user with ID "+state.EmployeeID.String()+": "+err.Error(),
		)
		return
	}
	if userToUpdate.InvitationPending {
		resp.Diagnostics.AddError(
			"Error updating user",
			"User with ID "+state.EmployeeID.String()+" has a pending invitation.",
		)
		return
	}

	// Build update request with only changed fields
	updateReq := &megaport.UpdateUserRequest{}

	// Check for changes and update accordingly
	if !plan.FirstName.Equal(state.FirstName) {
		firstName := plan.FirstName.ValueString()
		updateReq.FirstName = &firstName
	}

	if !plan.LastName.Equal(state.LastName) {
		lastName := plan.LastName.ValueString()
		updateReq.LastName = &lastName
	}

	if !plan.Email.Equal(state.Email) {
		email := plan.Email.ValueString()
		updateReq.Email = &email
	}

	if !plan.Phone.Equal(state.Phone) {
		phone := plan.Phone.ValueString()
		updateReq.Phone = &phone
	}

	if !plan.Position.Equal(state.Position) {
		position := plan.Position.ValueString()
		updateReq.Position = &position
	}

	if !plan.Active.Equal(state.Active) {
		active := plan.Active.ValueBool()
		updateReq.Active = &active
	}

	if !plan.NotificationEnabled.Equal(state.NotificationEnabled) {
		notificationEnabled := plan.NotificationEnabled.ValueBool()
		updateReq.NotificationEnabled = &notificationEnabled
	}

	if !plan.Newsletter.Equal(state.Newsletter) {
		newsletter := plan.Newsletter.ValueBool()
		updateReq.Newsletter = &newsletter
	}

	if !plan.Promotions.Equal(state.Promotions) {
		promotions := plan.Promotions.ValueBool()
		updateReq.Promotions = &promotions
	}

	if !plan.ChannelManager.Equal(state.ChannelManager) {
		channelManager := plan.ChannelManager.ValueBool()
		updateReq.ChannelManager = &channelManager
	}

	// Check for security roles changes
	if !plan.SecurityRoles.Equal(state.SecurityRoles) {
		var roles []string
		if !plan.SecurityRoles.IsNull() && !plan.SecurityRoles.IsUnknown() {
			var roleValues []basetypes.StringValue
			diags = plan.SecurityRoles.ElementsAs(ctx, &roleValues, false)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}

			for _, roleValue := range roleValues {
				roles = append(roles, roleValue.ValueString())
			}
		}
		updateReq.SecurityRoles = &roles
	}

	// Update the user
	userMgmt := r.client.UserManagementService
	err = userMgmt.UpdateUser(ctx, int(state.EmployeeID.ValueInt64()), updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating user",
			"Could not update user with ID "+state.EmployeeID.String()+": "+err.Error(),
		)
		return
	}

	// Get the updated user to refresh the state
	user, err := userMgmt.GetUser(ctx, int(state.EmployeeID.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading updated user",
			"Could not read user after update: "+err.Error(),
		)
		return
	}

	// Update the state with the user info
	diags = plan.fromAPIUser(ctx, user)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *userResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state userResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the user
	userMgmt := r.client.UserManagementService
	err := userMgmt.DeleteUser(ctx, int(state.EmployeeID.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting user",
			"Could not delete user with ID "+state.EmployeeID.String()+": "+err.Error(),
		)
		return
	}
}

// Configure adds the provider configured client to the resource.
func (r *userResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	r.client = client
}

func (r *userResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Parse the import ID as employee ID
	employeeID, err := strconv.Atoi(req.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error importing user",
			"Could not parse employee ID from import ID: "+err.Error(),
		)
		return
	}

	// Set the employee ID attribute
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("employee_id"), employeeID)...)
}
