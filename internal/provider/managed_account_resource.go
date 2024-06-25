package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	megaport "github.com/megaport/megaportgo"
)

var (
	// Ensure the implementation satisfies the expected interfaces.
	_ resource.Resource                = &managedAccountResource{}
	_ resource.ResourceWithConfigure   = &managedAccountResource{}
	_ resource.ResourceWithImportState = &managedAccountResource{}
)

// managedAccountResourceModel maps the resource schema data.
type managedAccountResourceModel struct {
	LastUpdated types.String `tfsdk:"last_updated"`

	// Managed Account Attributes
	CompanyUID  types.String `tfsdk:"company_uid"`
	AccountName types.String `tfsdk:"company_name"`
	AccountRef  types.String `tfsdk:"account_ref"`
}

// fromAPIManagedAccount maps the API MCR response to the resource schema.
func (orm *managedAccountResourceModel) fromAPIManagedAccount(m *megaport.ManagedAccount) {
	orm.CompanyUID = types.StringValue(m.CompanyUID)
	orm.AccountName = types.StringValue(m.AccountName)
	orm.AccountRef = types.StringValue(m.AccountRef)
}

// NewManagedAccountResource is a helper function to simplify the provider implemeantation.
func NewManagedAccountResource() resource.Resource {
	return &managedAccountResource{}
}

// managedAccountResource is the resource implementation.
type managedAccountResource struct {
	client *megaport.Client
}

// Metadata returns the resource type name.
func (r *managedAccountResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_managed_account"
}

// Schema defines the schema for the resource.
func (r *managedAccountResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Managed Account Resource for the Megaport Terraform Provider.The Megaport API contains several endpoints to manage partner accounts. Partners can list, order, edit, and terminate services for a managed account. Additionally, as a partner, you can act on behalf of a managed account.",
		Attributes: map[string]schema.Attribute{
			"last_updated": schema.StringAttribute{
				Description: "Last updated by the Terraform provider.",
				Computed:    true,
			},
			"company_uid": schema.StringAttribute{
				Description: "Company UID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"account_name": schema.StringAttribute{
				Description: "Managed Account Name.",
				Required:    true,
			},
			"account_ref": schema.StringAttribute{
				Description: "Managed Account Reference.",
				Required:    true,
			},
		},
	}
}

// Create a new resource.
func (r *managedAccountResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan managedAccountResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &megaport.ManagedAccountRequest{
		AccountName: plan.AccountName.ValueString(),
		AccountRef:  plan.AccountRef.ValueString(),
	}

	createdManagedAccount, err := r.client.ManagedAccountService.CreateManagedAccount(ctx, createReq)

	createdID := createdManagedAccount.CompanyUID

	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Managed Account",
			"Could not create Managed Account: "+err.Error(),
		)
	}

	getRes, err := r.client.ManagedAccountService.GetManagedAccount(ctx, createdID, plan.AccountName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Managed Account",
			"Could not read Managed Account with ID "+createdID+": "+err.Error(),
		)
	}

	plan.fromAPIManagedAccount(getRes)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read resource information.
func (r *managedAccountResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state managedAccountResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get refreshed mcr value from API
	managedAccount, err := r.client.ManagedAccountService.GetManagedAccount(ctx, state.CompanyUID.ValueString(), state.AccountName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Managed Account",
			"Could not read Managed Account with ID "+state.CompanyUID.ValueString()+": "+err.Error(),
		)
	}

	state.fromAPIManagedAccount(managedAccount)

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *managedAccountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state managedAccountResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Check on changes
	var name, accountRef string

	if !plan.AccountName.Equal(state.AccountName) {
		name = plan.AccountName.ValueString()
	} else {
		name = state.AccountName.ValueString()
	}
	if !plan.AccountRef.Equal(state.AccountRef) {
		accountRef = plan.AccountRef.ValueString()
	} else {
		accountRef = state.AccountRef.ValueString()

	}

	_, err := r.client.ManagedAccountService.UpdateManagedAccount(ctx, state.CompanyUID.ValueString(), &megaport.ManagedAccountRequest{
		AccountName: name,
		AccountRef:  accountRef,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating MCR",
			"Could not update MCR, unexpected error: "+err.Error(),
		)
		return
	}

	// Get refreshed managed account from API
	managedAccount, err := r.client.ManagedAccountService.GetManagedAccount(ctx, state.CompanyUID.ValueString(), name)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Managed Account",
			"Could not read Managed Account with ID "+state.CompanyUID.ValueString()+": "+err.Error(),
		)
		return
	}

	state.fromAPIManagedAccount(managedAccount)
	// Update the state with the new values
	state.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *managedAccountResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// TODO
}

// Configure adds the provider configured client to the resource.
func (r *managedAccountResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *managedAccountResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Retrieve import ID and save to id attribute
	// TODO
}
