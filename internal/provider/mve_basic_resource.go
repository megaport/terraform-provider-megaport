package provider

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	megaport "github.com/megaport/megaportgo"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &mveBasicResource{}
	_ resource.ResourceWithConfigure   = &mveBasicResource{}
	_ resource.ResourceWithImportState = &mveBasicResource{}
)

// mveBasicResourceModel maps the resource schema data.
type mveBasicResourceModel struct {
	ID                    types.Int64  `tfsdk:"product_id"`
	UID                   types.String `tfsdk:"product_uid"`
	Name                  types.String `tfsdk:"product_name"`
	Type                  types.String `tfsdk:"product_type"`
	Market                types.String `tfsdk:"market"`
	LocationID            types.Int64  `tfsdk:"location_id"`
	UsageAlgorithm        types.String `tfsdk:"usage_algorithm"`
	MarketplaceVisibility types.Bool   `tfsdk:"marketplace_visibility"`
	VXCPermitted          types.Bool   `tfsdk:"vxc_permitted"`
	VXCAutoApproval       types.Bool   `tfsdk:"vxc_auto_approval"`
	CompanyUID            types.String `tfsdk:"company_uid"`
	ContractTermMonths    types.Int64  `tfsdk:"contract_term_months"`
	PromoCode             types.String `tfsdk:"promo_code"`
	CostCentre            types.String `tfsdk:"cost_centre"`
	DiversityZone         types.String `tfsdk:"diversity_zone"`

	Locked      types.Bool `tfsdk:"locked"`
	AdminLocked types.Bool `tfsdk:"admin_locked"`
	Cancelable  types.Bool `tfsdk:"cancelable"`

	Vendor types.String `tfsdk:"vendor"`
	Size   types.String `tfsdk:"mve_size"`

	VendorConfig types.Object `tfsdk:"vendor_config"`

	NetworkInterfaces types.List `tfsdk:"vnics"`

	ResourceTags types.Map `tfsdk:"resource_tags"`
}

func (orm *mveBasicResourceModel) fromAPIMVE(ctx context.Context, p *megaport.MVE, tags map[string]string) diag.Diagnostics {
	apiDiags := diag.Diagnostics{}
	orm.ID = types.Int64Value(int64(p.ID))
	orm.UID = types.StringValue(p.UID)
	orm.Name = types.StringValue(p.Name)
	orm.Type = types.StringValue(p.Type)
	orm.Market = types.StringValue(p.Market)
	orm.LocationID = types.Int64Value(int64(p.LocationID))
	orm.UsageAlgorithm = types.StringValue(p.UsageAlgorithm)
	orm.MarketplaceVisibility = types.BoolValue(p.MarketplaceVisibility)
	orm.VXCPermitted = types.BoolValue(p.VXCPermitted)
	orm.VXCAutoApproval = types.BoolValue(p.VXCAutoApproval)
	orm.CompanyUID = types.StringValue(p.CompanyUID)
	orm.ContractTermMonths = types.Int64Value(int64(p.ContractTermMonths))

	orm.Locked = types.BoolValue(p.Locked)
	orm.AdminLocked = types.BoolValue(p.AdminLocked)
	orm.Cancelable = types.BoolValue(p.Cancelable)
	orm.Vendor = types.StringValue(p.Vendor)
	orm.Size = types.StringValue(p.Size)
	orm.CostCentre = types.StringValue(p.CostCentre)
	orm.DiversityZone = types.StringValue(p.DiversityZone)

	if len(tags) > 0 {
		resourceTags, tagDiags := types.MapValueFrom(ctx, types.StringType, tags)
		apiDiags = append(apiDiags, tagDiags...)
		orm.ResourceTags = resourceTags
	} else {
		orm.ResourceTags = types.MapNull(types.StringType)
	}

	vnics := []types.Object{}
	for _, n := range p.NetworkInterfaces {
		model := &mveNetworkInterfaceModel{
			Description: types.StringValue(n.Description),
			VLAN:        types.Int64Value(int64(n.VLAN)),
		}
		vnic, vnicDiags := types.ObjectValueFrom(ctx, vnicAttrs, model)
		apiDiags = append(apiDiags, vnicDiags...)
		vnics = append(vnics, vnic)
	}
	networkInterfaceList, listDiags := types.ListValueFrom(ctx, types.ObjectType{}.WithAttributeTypes(vnicAttrs), vnics)
	apiDiags = append(apiDiags, listDiags...)
	orm.NetworkInterfaces = networkInterfaceList

	return apiDiags
}

// NewPortResource is a helper function to simplify the provider implementation.
func NewMVEBasicResource() resource.Resource {
	return &mveBasicResource{}
}

// mveBasicResource is the resource implementation.
type mveBasicResource struct {
	client *megaport.Client
}

// Metadata returns the resource type name.
func (r *mveBasicResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mve_basic"
}

// Schema defines the schema for the resource.
func (r *mveBasicResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Megaport Virtual Edge (MVE) Resource for Megaport Terraform provider. This resource allows you to create, modify, and delete Megaport MVEs. Megaport Virtual Edge (MVE) is an on-demand, vendor-neutral Network Function Virtualization (NFV) platform that provides virtual infrastructure for network services at the edge of Megaport’s global software-defined network (SDN). Network technologies such as SD-WAN and NGFW are hosted directly on Megaport’s global network via Megaport Virtual Edge.",
		Attributes: map[string]schema.Attribute{
			"product_uid": schema.StringAttribute{
				Description: "The unique identifier of the MVE.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"product_id": schema.Int64Attribute{
				Description: "The Numeric ID of the MVE.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"product_name": schema.StringAttribute{
				Description: "The name of the MVE.",
				Required:    true,
			},
			"diversity_zone": schema.StringAttribute{
				Description: "The diversity zone of the MVE.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"market": schema.StringAttribute{
				Description: "The market the MVE is in.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"location_id": schema.Int64Attribute{
				Description: "The numeric location ID of the product. This value can be retrieved from the data source megaport_location.",
				Required:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"product_type": schema.StringAttribute{
				Description: "The type of product (MVE).",
				Computed:    true,
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
			"usage_algorithm": schema.StringAttribute{
				Description: "The usage algorithm of the MVE.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"company_uid": schema.StringAttribute{
				Description: "The company UID of the MVE.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cost_centre": schema.StringAttribute{
				Description: "The cost centre of the MVE.",
				Optional:    true,
				Computed:    true,
			},
			"marketplace_visibility": schema.BoolAttribute{
				Description: "Whether the MVE is visible in the marketplace.",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"vxc_permitted": schema.BoolAttribute{
				Description: "Whether VXC is permitted.",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"vxc_auto_approval": schema.BoolAttribute{
				Description: "Whether VXC is auto approved.",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"locked": schema.BoolAttribute{
				Description: "Whether the MVE is locked.",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"admin_locked": schema.BoolAttribute{
				Description: "Whether the MVE is admin locked.",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"cancelable": schema.BoolAttribute{
				Description: "Whether the MVE is cancelable.",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"vendor": schema.StringAttribute{
				Description: "The vendor of the MVE.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"promo_code": schema.StringAttribute{
				Description: "Promo code is an optional string that can be used to enter a promotional code for the service order. The code is not validated, so if the code doesn't exist or doesn't work for the service, the request will still be successful.",
				Optional:    true,
			},
			"mve_size": schema.StringAttribute{
				Description: "The size of the MVE.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"vnics": schema.ListNestedAttribute{
				Description: "The network interfaces of the MVE. The number of elements in the array is the number of vNICs the user wants to provision. Description can be null. The maximum number of vNICs allowed is 5. If the array is not supplied (i.e. null), it will default to the minimum number of vNICs for the supplier - 2 for Palo Alto and 1 for the others.",
				Optional:    true,
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"description": schema.StringAttribute{
							Description: "The description of the network interface.",
							Required:    true,
						},
						"vlan": schema.Int64Attribute{
							Description: "The VLAN of the network interface.",
							Computed:    true,
							PlanModifiers: []planmodifier.Int64{
								int64planmodifier.UseStateForUnknown(),
							},
						},
					},
				},
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
					listplanmodifier.RequiresReplace(),
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
			"vendor_config": schema.SingleNestedAttribute{
				Description: "The vendor configuration of the MVE. Vendor-specific information required to bootstrap the MVE. These values will be different for each vendor, and can include vendor name, size of VM, license/activation code, software version, and SSH keys. This field cannot be changed after the MVE is created and if it is modified, the MVE will be deleted and re-created. Imported MVEs do not have this field populated by the API, so the initially provided configuration will be ignored as it can't be verified to be correct. If the user wants to change the configuration after importing the resource, they can then do so by changing the field after importing the resource and running terraform apply.",
				Required:    true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"vendor": schema.StringAttribute{
						Description: `The name of vendor of the MVE. Currently supported values: "6wind", "aruba", "aviatrix", "cisco", "fortinet", "palo_alto", "prisma", "versa", "vmware", "meraki".`,
						Required:    true,
					},
					"image_id": schema.Int64Attribute{
						Description: "The image ID of the MVE. Indicates the software version.",
						Required:    true,
					},
					"product_size": schema.StringAttribute{
						Description: "The product size for the vendor config. The size defines the MVE specifications including number of cores, bandwidth, and number of connections.",
						Required:    true,
					},
					"mve_label": schema.StringAttribute{
						Description: "The MVE label for the vendor config.",
						Optional:    true,
					},
					"account_name": schema.StringAttribute{
						Description: "The account name for the vendor config. Enter the Account Name from Aruba Orchestrator. To view your Account Name, log in to Orchestrator and choose Orchestrator > Licensing | Cloud Portal. Required for Aruba MVE.",
						Optional:    true,
					},
					"account_key": schema.StringAttribute{
						Description: "The account key for the vendor config. Enter the Account Key from Aruba Orchestrator. The key is linked to the Account Name. Required for Aruba MVE.",
						Optional:    true,
					},
					"admin_ssh_public_key": schema.StringAttribute{
						Description: "The admin SSH public key for the vendor config. Required for Cisco, Fortinet, and Vmware MVEs.",
						Optional:    true,
					},
					"ssh_public_key": schema.StringAttribute{
						Description: "The SSH public key for the vendor config. Required for 6WIND, VMWare, Palo Alto, and Fortinet MVEs. Megaport supports the 2048-bit RSA key type.",
						Optional:    true,
					},
					"cloud_init": schema.StringAttribute{
						Description: "The Base64 encoded cloud init file for the vendor config. The bootstrap configuration file. Required for Aviatrix and Cisco C8000v.",
						Optional:    true,
					},
					"license_data": schema.StringAttribute{
						Description: "The license data for the vendor config. Required for Fortinet and Palo Alto MVEs.",
						Optional:    true,
					},
					"admin_password_hash": schema.StringAttribute{
						Description: "The admin password hash for the vendor config. Required for Palo Alto MVE.",
						Optional:    true,
					},
					"director_address": schema.StringAttribute{
						Description: "The director address for the vendor config. A FQDN (Fully Qualified Domain Name) or IPv4 address of your Versa Director. Required for Versa MVE.",
						Optional:    true,
					},
					"controller_address": schema.StringAttribute{
						Description: "The controldler address for the vendor config. A FQDN (Fully Qualified Domain Name) or IPv4 address of your Versa Controller. Required for Versa MVE.",
						Optional:    true,
					},
					"manage_locally": schema.BoolAttribute{
						Description: "Whether to manage the MVE locally. Required for Cisco MVE.",
						Optional:    true,
					},
					"local_auth": schema.StringAttribute{
						Description: "The local auth for the vendor config. Enter the Local Auth string as configured in your Versa Director. Required for Versa MVE.",
						Optional:    true,
					},
					"remote_auth": schema.StringAttribute{
						Description: "The remote auth for the vendor config. Enter the Remote Auth string as configured in your Versa Director. Required for Versa MVE.",
						Optional:    true,
					},
					"serial_number": schema.StringAttribute{
						Description: "The serial number for the vendor config. Enter the serial number that you specified when creating the device in Versa Director. Required for Versa MVE.",
						Optional:    true,
					},
					"system_tag": schema.StringAttribute{
						Description: "The system tag for the vendor config. Aruba Orchestrator System Tags and preconfiguration templates register the EC-V with the Cloud Portal and Orchestrator, and enable Orchestrator to automatically accept and configure newly discovered EC-V appliances. If you created a preconfiguration template in Orchestrator, enter the System Tag you specified here. Required for Aruba MVE.",
						Optional:    true,
					},
					"vco_address": schema.StringAttribute{
						Description: "The VCO address for the vendor config. A FQDN (Fully Qualified Domain Name) or IPv4 or IPv6 address for the Orchestrator where you created the edge device. Required for VMware MVE.",
						Optional:    true,
					},
					"vco_activation_code": schema.StringAttribute{
						Description: "The VCO activation code for the vendor config. This is provided by Orchestrator after creating the edge device. Required for VMware MVE.",
						Optional:    true,
					},
					"fmc_ip_address": schema.StringAttribute{
						Description: "The FMC IP address for the vendor config. Required for Cisco FTDv (Firewall) MVE.",
						Optional:    true,
					},
					"fmc_registration_key": schema.StringAttribute{
						Description: "The FMC registration key for the vendor config. Required for Cisco FTDv (Firewall) MVE.",
						Optional:    true,
					},
					"fmc_nat_id": schema.StringAttribute{
						Description: "The FMC NAT ID for the vendor config. Required for Cisco FTDv (Firewall) MVE.",
						Optional:    true,
					},
					"token": schema.StringAttribute{
						Description: "The token for the vendor config. Required for Meraki MVE.",
						Optional:    true,
					},
					"ion_key": schema.StringAttribute{
						Description: "The vION key for the vendor config. Required for Prisma MVE.",
						Optional:    true,
					},
					"secret_key": schema.StringAttribute{
						Description: "The secret key for the vendor config. Required for Prisma MVE.",
						Optional:    true,
					},
				},
			},
		},
	}
}

// Create a new resource.
func (r *mveBasicResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan mveBasicResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	mveReq := &megaport.BuyMVERequest{
		LocationID:    int(plan.LocationID.ValueInt64()),
		Name:          plan.Name.ValueString(),
		Term:          int(plan.ContractTermMonths.ValueInt64()),
		PromoCode:     plan.PromoCode.ValueString(),
		CostCentre:    plan.CostCentre.ValueString(),
		DiversityZone: plan.DiversityZone.ValueString(),

		WaitForProvision: true,
		WaitForTime:      waitForTime,
	}

	if !plan.ResourceTags.IsNull() {
		tagMap, tagDiags := toResourceTagMap(ctx, plan.ResourceTags)
		resp.Diagnostics.Append(tagDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		mveReq.ResourceTags = tagMap
	}

	if plan.VendorConfig.IsNull() {
		resp.Diagnostics.AddError(
			"vendor config required", "vendor config required",
		)
	}
	vcModel := &vendorConfigModel{}
	vcDiags := plan.VendorConfig.As(ctx, vcModel, basetypes.ObjectAsOptions{})
	resp.Diagnostics = append(resp.Diagnostics, vcDiags...)
	vendorConfig, apiVCDiags := toAPIVendorConfig(vcModel)
	resp.Diagnostics = append(resp.Diagnostics, apiVCDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	mveReq.VendorConfig = vendorConfig

	if !plan.NetworkInterfaces.IsNull() && len(plan.NetworkInterfaces.Elements()) > 0 {
		vnicModels := []*mveNetworkInterfaceModel{}
		vnicDiags := plan.NetworkInterfaces.ElementsAs(ctx, &vnicModels, false)
		resp.Diagnostics = append(resp.Diagnostics, vnicDiags...)
		for _, vnicModel := range vnicModels {
			vnic := toAPINetworkInterface(vnicModel)
			mveReq.Vnics = append(mveReq.Vnics, *vnic)
		}
	}

	err := r.client.MVEService.ValidateMVEOrder(ctx, mveReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Validation error while attempting to create MVE",
			"Validation error while attempting to create MVE with name "+plan.Name.ValueString()+": "+err.Error(),
		)
		return
	}

	createdMVE, err := r.client.MVEService.BuyMVE(ctx, mveReq)

	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading MVE",
			"Could not create MVE with name "+plan.Name.ValueString()+": "+err.Error(),
		)
		return
	}

	createdID := createdMVE.TechnicalServiceUID

	// get the created MVE
	mve, err := r.client.MVEService.GetMVE(ctx, createdID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading newly created MVE",
			"Could not read newly created MVE with ID "+createdID+": "+err.Error(),
		)
		return
	}

	tags, err := r.client.MVEService.ListMVEResourceTags(ctx, createdID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading tags for newly created MVE",
			"Could not read tags for newly created MVE with ID "+createdID+": "+err.Error(),
		)
		return
	}

	// update the plan with the MVE info
	apiDiags := plan.fromAPIMVE(ctx, mve, tags)
	resp.Diagnostics = append(resp.Diagnostics, apiDiags...)

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read resource information.
func (r *mveBasicResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state mveBasicResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get refreshed MVE value from API
	mve, err := r.client.MVEService.GetMVE(ctx, state.UID.ValueString())
	if err != nil {
		// MVE has been deleted or is not found
		if mpErr, ok := err.(*megaport.ErrorResponse); ok {
			if mpErr.Response.StatusCode == http.StatusNotFound ||
				(mpErr.Response.StatusCode == http.StatusBadRequest && strings.Contains(mpErr.Message, "Could not find a service with UID")) {
				resp.State.RemoveResource(ctx)
				return
			}
		}

		resp.Diagnostics.AddError(
			"Error Reading MVE",
			"Could not read MVE with ID "+state.UID.ValueString()+": "+err.Error(),
		)
		return
	}

	// If the MVE has been deleted
	if mve.ProvisioningStatus == megaport.STATUS_DECOMMISSIONED {
		resp.State.RemoveResource(ctx)
		return
	}

	// Get tags
	tags, err := r.client.MVEService.ListMVEResourceTags(ctx, state.UID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading tags for MVE",
			"Could not read tags for MVE with ID "+state.UID.ValueString()+": "+err.Error(),
		)
		return
	}

	apiDiags := state.fromAPIMVE(ctx, mve, tags)
	resp.Diagnostics = append(resp.Diagnostics, apiDiags...)

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *mveBasicResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state mveBasicResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	// If Imported, VendorConfig will be Null. Set VendorConfig in state to existing one in plan.
	if state.VendorConfig.IsNull() {
		state.VendorConfig = plan.VendorConfig
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Check on changes
	var name, costCentre string
	var contractTermMonths *int
	if !plan.Name.Equal(state.Name) {
		name = plan.Name.ValueString()
	} else {
		name = state.Name.ValueString()
	}

	if !plan.CostCentre.Equal(state.CostCentre) {
		costCentre = plan.CostCentre.ValueString()
	} else {
		costCentre = state.CostCentre.ValueString()
	}

	if !plan.ContractTermMonths.Equal(state.ContractTermMonths) {
		months := int(plan.ContractTermMonths.ValueInt64())
		contractTermMonths = &months
	}

	_, err := r.client.MVEService.ModifyMVE(ctx, &megaport.ModifyMVERequest{
		MVEID:              state.UID.ValueString(),
		Name:               name,
		CostCentre:         costCentre,
		ContractTermMonths: contractTermMonths,
		WaitForUpdate:      true,
		WaitForTime:        waitForTime,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating MVE",
			"Could not update MVE with ID "+state.UID.ValueString()+": "+err.Error(),
		)
		return
	}

	updatedMVE, err := r.client.MVEService.GetMVE(ctx, state.UID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading updated MVE",
			"Could not read updated MVE with ID "+state.UID.ValueString()+": "+err.Error(),
		)
		return
	}

	if !plan.ResourceTags.Equal(state.ResourceTags) {
		tagMap, tagDiags := toResourceTagMap(ctx, plan.ResourceTags)
		resp.Diagnostics.Append(tagDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		err = r.client.MVEService.UpdateMVEResourceTags(ctx, state.UID.ValueString(), tagMap)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating tags for MVE",
				"Could not update tags for MVE with ID "+state.UID.ValueString()+": "+err.Error(),
			)
			return
		}
	}

	tags, err := r.client.MVEService.ListMVEResourceTags(ctx, state.UID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading tags for updated MVE",
			"Could not read tags for updated MVE with ID "+state.UID.ValueString()+": "+err.Error(),
		)
		return
	}

	apiDiags := state.fromAPIMVE(ctx, updatedMVE, tags)
	resp.Diagnostics = append(resp.Diagnostics, apiDiags...)

	diags := resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *mveBasicResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve the state
	var state mveBasicResourceModel
	stateDiags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(stateDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Call the API to delete the resource
	productUID := state.UID.ValueString()
	_, err := r.client.MVEService.DeleteMVE(ctx, &megaport.DeleteMVERequest{
		MVEID:      productUID,
		SafeDelete: true,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting MVE",
			fmt.Sprintf("Could not delete MVE with product UID %s: %s", productUID, err),
		)
		return
	}

	// Remove the resource from the state
	resp.State.RemoveResource(ctx)
}

// Configure adds the provider configured client to the resource.
func (r *mveBasicResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *mveBasicResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Retrieve import ID and save to id attribute
	resource.ImportStatePassthroughID(ctx, path.Root("product_uid"), req, resp)
}

func (r *mveBasicResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Get the plan and state
	var plan, state mveBasicResourceModel
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

	if !state.UID.IsNull() {
		// If VendorConfig is null in the state, set it to the value from the plan
		if state.VendorConfig.IsNull() {
			var planVendorConfig vendorConfigModel
			planVendorConfigDiags := plan.VendorConfig.As(ctx, &planVendorConfig, basetypes.ObjectAsOptions{})
			resp.Diagnostics = append(resp.Diagnostics, planVendorConfigDiags...)
			if resp.Diagnostics.HasError() {
				return
			}
			// Check the computed vendor/size values from the API. If the vendor or size changes, require a replace - check with case insensitivity.

			if !strings.EqualFold(state.Size.ValueString(), planVendorConfig.ProductSize.ValueString()) {
				resp.RequiresReplace = append(resp.RequiresReplace, path.Root("vendor_config"))
			}

			if !strings.EqualFold(state.Vendor.ValueString(), planVendorConfig.Vendor.ValueString()) {
				resp.RequiresReplace = append(resp.RequiresReplace, path.Root("vendor_config"))
			}
			state.VendorConfig = plan.VendorConfig
		} else if !plan.VendorConfig.Equal(state.VendorConfig) {
			resp.RequiresReplace = append(resp.RequiresReplace, path.Root("vendor_config"))
		}
		diags := req.State.Set(ctx, &state)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
}
