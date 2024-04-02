package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	megaport "github.com/megaport/megaportgo"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &mveResource{}
	_ resource.ResourceWithConfigure   = &mveResource{}
	_ resource.ResourceWithImportState = &mveResource{}
)

// mveResourceModel maps the resource schema data.
type mveResourceModel struct {
	LastUpdated types.String `tfsdk:"last_updated"`

	ID                    types.Int64  `tfsdk:"product_id"`
	UID                   types.String `tfsdk:"product_uid"`
	Name                  types.String `tfsdk:"product_name"`
	Type                  types.String `tfsdk:"product_type"`
	ProvisioningStatus    types.String `tfsdk:"provisioning_status"`
	CreateDate            types.String `tfsdk:"create_date"`
	CreatedBy             types.String `tfsdk:"created_by"`
	TerminateDate         types.String `tfsdk:"terminate_date"`
	LiveDate              types.Int64  `tfsdk:"live_date"`
	Market                types.String `tfsdk:"market"`
	LocationID            types.Int64  `tfsdk:"location_id"`
	UsageAlgorithm        types.String `tfsdk:"usage_algorithm"`
	MarketplaceVisibility types.Bool   `tfsdk:"marketplace_visibility"`
	VXCPermitted          types.Bool   `tfsdk:"vxcpermitted"`
	VXCAutoApproval       types.Bool   `tfsdk:"vxc_auto_approval"`
	SecondaryName         types.String `tfsdk:"secondary_name"`
	CompanyUID            types.String `tfsdk:"company_uid"`
	CompanyName           types.String `tfsdk:"company_name"`
	ContractStartDate     types.String `tfsdk:"contract_start_date"`
	ContractEndDate       types.String `tfsdk:"contract_end_date"`
	ContractTermMonths    types.Int64  `tfsdk:"contract_term_months"`

	Virtual     types.Bool `tfsdk:"virtual"`
	BuyoutPort  types.Bool `tfsdk:"buyout_port"`
	Locked      types.Bool `tfsdk:"locked"`
	AdminLocked types.Bool `tfsdk:"admin_locked"`
	Cancelable  types.Bool `tfsdk:"cancelable"`

	Vendor types.String `tfsdk:"vendor"`
	Size   types.String `tfsdk:"mve_size"`

	VendorConfig vendorConfigModel `tfsdk:"vendor_config"`

	NetworkInterfaces []*mveNetworkInterfaceModel   `tfsdk:"vnics"`
	AttributeTags     map[types.String]types.String `tfsdk:"attribute_tags"`
	Resources         *mveResourcesModel            `tfsdk:"resources"`
}

// mveNetworkInterfaceModel represents a vNIC.
type mveNetworkInterfaceModel struct {
	Description types.String `tfsdk:"description"`
	VLAN        types.Int64  `tfsdk:"vlan"`
}

func (orm *mveNetworkInterfaceModel) toAPINetworkInterface() megaport.MVENetworkInterface {
	return megaport.MVENetworkInterface{
		Description: orm.Description.ValueString(),
		VLAN:        int(orm.VLAN.ValueInt64()),
	}
}

// mveResourcesModel represents the resources associated with an MVE.
type mveResourcesModel struct {
	Interface       *portInterfaceModel       `tfsdk:"interface"`
	VirtualMachines []*mveVirtualMachineModel `tfsdk:"virtual_machine"`
}

// mveVirtualMachineModel represents a virtual machine associated with an MVE.
type mveVirtualMachineModel struct {
	ID           types.Int64                  `tfsdk:"id"`
	CpuCount     types.Int64                  `tfsdk:"cpu_count"`
	Image        *mveVirtualMachineImageModel `tfsdk:"image"`
	ResourceType types.String                 `tfsdk:"resource_type"`
	Up           types.Bool                   `tfsdk:"up"`
	Vnics        []*mveNetworkInterfaceModel  `tfsdk:"vnics"`
}

// MVVEVirtualMachineImage represents the image associated with an MVE virtual machine.
type mveVirtualMachineImageModel struct {
	ID      types.Int64  `tfsdk:"id"`
	Vendor  types.String `tfsdk:"vendor"`
	Product types.String `tfsdk:"product"`
	Version types.String `tfsdk:"version"`
}

// vendorConfigModel is an interface for MVE vendor configuration.
type vendorConfigModel interface {
	IsVendorConfig()
}

// arubaConfigModel represents the configuration for an Aruba MVE.
type arubaConfigModel struct {
	vendorConfigModel
	Vendor      types.String `tfsdk:"vendor"`
	ImageID     types.Int64  `tfsdk:"image_id"`
	ProductSize types.String `tfsdk:"product_size"`
	AccountName types.String `tfsdk:"account_name"`
	AccountKey  types.String `tfsdk:"account_key"`
}

// ciscoConfigModel represents the configuration for a Cisco MVE.
type ciscoConfigModel struct {
	vendorConfigModel
	Vendor            types.String `tfsdk:"vendor"`
	ImageID           types.Int64  `tfsdk:"image_id"`
	ProductSize       types.String `tfsdk:"product_size"`
	AdminSSHPublicKey types.String `tfsdk:"admin_ssh_public_key"`
	CloudInit         types.String `tfsdk:"cloud_init"`
}

// fortinetConfigModel represents the configuration for a Fortinet MVE.
type fortinetConfigModel struct {
	vendorConfigModel
	Vendor            types.String `tfsdk:"vendor"`
	ImageID           types.Int64  `tfsdk:"image_id"`
	ProductSize       types.String `tfsdk:"product_size"`
	AdminSSHPublicKey types.String `tfsdk:"admin_ssh_public_key"`
	LicenseData       types.String `tfsdk:"license_data"`
}

// paloAltoConfigModel represents the configuration for a Palo Alto MVE.
type paloAltoConfigModel struct {
	vendorConfigModel
	Vendor            types.String `tfsdk:"vendor"`
	ImageID           types.Int64  `tfsdk:"image_id"`
	ProductSize       types.String `tfsdk:"product_size"`
	AdminSSHPublicKey types.String `tfsdk:"admin_ssh_public_key"`
	AdminPasswordHash types.String `tfsdk:"admin_password_hash"`
	LicenseData       types.String `tfsdk:"license_data"`
}

// versaConfigModel represents the configuration for a Versa MVE.
type versaConfigModel struct {
	vendorConfigModel
	Vendor            types.String `tfsdk:"vendor"`
	ImageID           types.Int64  `tfsdk:"image_id"`
	ProductSize       types.String `tfsdk:"product_size"`
	DirectorAddress   types.String `tfsdk:"director_address"`
	ControllerAddress types.String `tfsdk:"controller_address"`
	LocalAuth         types.String `tfsdk:"local_auth"`
	RemoteAuth        types.String `tfsdk:"remote_auth"`
	SerialNumber      types.String `tfsdk:"serial_number"`
}

// vmwareConfigModel represents the configuration for a VMware MVE.
type vmwareConfig struct {
	vendorConfigModel
	Vendor            types.String `tfsdk:"vendor"`
	ImageID           types.Int64  `tfsdk:"image_id"`
	ProductSize       types.String `tfsdk:"product_size"`
	AdminSSHPublicKey types.String `tfsdk:"admin_ssh_public_key"`
	VcoAddress        types.String `tfsdk:"vco_address"`
	VcoActivationCode types.String `tfsdk:"vco_activation_code"`
}

func (orm *mveResourceModel) fromAPIMVE(p *megaport.MVE) {
	orm.ID = types.Int64Value(int64(p.ID))
	orm.UID = types.StringValue(p.UID)
	orm.Name = types.StringValue(p.Name)
	orm.Type = types.StringValue(p.Type)
	orm.ProvisioningStatus = types.StringValue(p.ProvisioningStatus)
	orm.CreateDate = types.StringValue(p.CreateDate.String())
	orm.CreatedBy = types.StringValue(p.CreatedBy)
	orm.TerminateDate = types.StringValue(p.TerminateDate.String())
	orm.LiveDate = types.Int64Value(int64(p.LiveDate))
	orm.Market = types.StringValue(p.Market)
	orm.LocationID = types.Int64Value(int64(p.LocationID))
	orm.UsageAlgorithm = types.StringValue(p.UsageAlgorithm)
	orm.MarketplaceVisibility = types.BoolValue(p.MarketplaceVisibility)
	orm.VXCPermitted = types.BoolValue(p.VXCPermitted)
	orm.VXCAutoApproval = types.BoolValue(p.VXCAutoApproval)
	orm.SecondaryName = types.StringValue(p.SecondaryName)
	orm.CompanyUID = types.StringValue(p.CompanyUID)
	orm.CompanyName = types.StringValue(p.CompanyName)
	orm.ContractStartDate = types.StringValue(p.ContractStartDate.String())
	orm.ContractEndDate = types.StringValue(p.ContractEndDate.String())
	orm.ContractTermMonths = types.Int64Value(int64(p.ContractTermMonths))
	orm.Virtual = types.BoolValue(p.Virtual)
	orm.BuyoutPort = types.BoolValue(p.BuyoutPort)
	orm.Locked = types.BoolValue(p.Locked)
	orm.AdminLocked = types.BoolValue(p.AdminLocked)
	orm.Cancelable = types.BoolValue(p.Cancelable)
	orm.Vendor = types.StringValue(p.Vendor)
	orm.Size = types.StringValue(p.Size)

	for k, v := range p.AttributeTags {
		orm.AttributeTags[types.StringValue(k)] = types.StringValue(v)
	}

	for _, vnic := range p.NetworkInterfaces {
		orm.NetworkInterfaces = append(orm.NetworkInterfaces, &mveNetworkInterfaceModel{
			Description: types.StringValue(vnic.Description),
			VLAN:        types.Int64Value(int64(vnic.VLAN)),
		})
	}

	if p.Resources != nil {
		r := &mveResourcesModel{}
		if p.Resources.Interface != nil {
			r.Interface = &portInterfaceModel{
				Demarcation:  types.StringValue(p.Resources.Interface.Demarcation),
				Description:  types.StringValue(p.Resources.Interface.Description),
				ID:           types.Int64Value(int64(p.Resources.Interface.ID)),
				LOATemplate:  types.StringValue(p.Resources.Interface.LOATemplate),
				Media:        types.StringValue(p.Resources.Interface.Media),
				Name:         types.StringValue(p.Resources.Interface.Name),
				PortSpeed:    types.Int64Value(int64(p.Resources.Interface.PortSpeed)),
				ResourceName: types.StringValue(p.Resources.Interface.ResourceName),
				ResourceType: types.StringValue(p.Resources.Interface.ResourceType),
			}
		}
		for _, vm := range p.Resources.VirtualMachines {
			r.VirtualMachines = append(r.VirtualMachines, &mveVirtualMachineModel{
				ID:           types.Int64Value(int64(vm.ID)),
				CpuCount:     types.Int64Value(int64(vm.CpuCount)),
				ResourceType: types.StringValue(vm.ResourceType),
				Up:           types.BoolValue(vm.Up),
			})
		}
		orm.Resources = r
	}
}

func toAPIVendorConfig(v vendorConfigModel) megaport.VendorConfig {
	switch c := v.(type) {
	case *arubaConfigModel:
		return &megaport.ArubaConfig{
			Vendor:      c.Vendor.ValueString(),
			ImageID:     int(c.ImageID.ValueInt64()),
			ProductSize: c.ProductSize.ValueString(),
			AccountName: c.AccountName.ValueString(),
			AccountKey:  c.AccountKey.ValueString(),
		}
	case *ciscoConfigModel:
		return &megaport.CiscoConfig{
			Vendor:            c.Vendor.ValueString(),
			ImageID:           int(c.ImageID.ValueInt64()),
			ProductSize:       c.ProductSize.ValueString(),
			AdminSSHPublicKey: c.AdminSSHPublicKey.ValueString(),
			CloudInit:         c.CloudInit.ValueString(),
		}
	case *fortinetConfigModel:
		return &megaport.FortinetConfig{
			Vendor:            c.Vendor.ValueString(),
			ImageID:           int(c.ImageID.ValueInt64()),
			ProductSize:       c.ProductSize.ValueString(),
			AdminSSHPublicKey: c.AdminSSHPublicKey.ValueString(),
			LicenseData:       c.LicenseData.ValueString(),
		}
	case *paloAltoConfigModel:
		return &megaport.PaloAltoConfig{
			Vendor:            c.Vendor.ValueString(),
			ImageID:           int(c.ImageID.ValueInt64()),
			ProductSize:       c.ProductSize.ValueString(),
			AdminSSHPublicKey: c.AdminSSHPublicKey.ValueString(),
			AdminPasswordHash: c.AdminPasswordHash.ValueString(),
			LicenseData:       c.LicenseData.ValueString(),
		}
	case *versaConfigModel:
		return &megaport.VersaConfig{
			Vendor:            c.Vendor.ValueString(),
			ImageID:           int(c.ImageID.ValueInt64()),
			ProductSize:       c.ProductSize.ValueString(),
			DirectorAddress:   c.DirectorAddress.ValueString(),
			ControllerAddress: c.ControllerAddress.ValueString(),
			LocalAuth:         c.LocalAuth.ValueString(),
			RemoteAuth:        c.RemoteAuth.ValueString(),
			SerialNumber:      c.SerialNumber.ValueString(),
		}
	case *vmwareConfig:
		return &megaport.VmwareConfig{
			Vendor:            c.Vendor.ValueString(),
			ImageID:           int(c.ImageID.ValueInt64()),
			ProductSize:       c.ProductSize.ValueString(),
			AdminSSHPublicKey: c.AdminSSHPublicKey.ValueString(),
			VcoAddress:        c.VcoAddress.ValueString(),
			VcoActivationCode: c.VcoActivationCode.ValueString(),
		}
	}
	return nil
}

// NewPortResource is a helper function to simplify the provider implementation.
func NewMVEResource() resource.Resource {
	return &mveResource{}
}

// mveResource is the resource implementation.
type mveResource struct {
	client *megaport.Client
}

// Metadata returns the resource type name.
func (r *mveResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mve"
}

// Schema defines the schema for the resource.
func (r *mveResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Megaport Virtual Edge (MVE) resource for Megaport Terraform provider.",
		Attributes: map[string]schema.Attribute{
			"last_updated": schema.StringAttribute{
				Description: "The last time the MVE was updated by the Terraform Provider.",
				Computed:    true,
			},
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
			"provisioning_status": schema.StringAttribute{
				Description: "The provisioning status of the MVE.",
				Computed:    true,
			},
			"create_date": schema.StringAttribute{
				Description: "The date the MVE was created.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_by": schema.StringAttribute{
				Description: "The user who created the MVE.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"terminate_date": schema.StringAttribute{
				Description: "The date the MVE will be terminated.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"live_date": schema.StringAttribute{
				Description: "The date the MVE went live.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"market": schema.StringAttribute{
				Description: "The market the MVE is in.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"location_id": schema.Int64Attribute{
				Description: "The location ID of the MVE.",
				Required:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"contract_term_months": schema.Int64Attribute{
				Description: "The contract term in months.",
				Required:    true,
				Validators: []validator.Int64{
					int64validator.OneOf(1, 12, 24, 36),
				},
			},
			"usage_algorithm": schema.StringAttribute{
				Description: "The usage algorithm of the MVE.",
				Computed:    true,
			},
			"company_uid": schema.StringAttribute{
				Description: "The company UID of the MVE.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"contract_start_date": schema.BoolAttribute{
				Description: "The contract start date of the MVE.",
				Computed:    true,
			},
			"contract_end_date": schema.BoolAttribute{
				Description: "The contract end date of the MVE.",
				Computed:    true,
			},
			"marketplace_visibility": schema.BoolAttribute{
				Description: "Whether the MVE is visible in the marketplace.",
				Computed:    true,
			},
			"vxc_permitted": schema.BoolAttribute{
				Description: "Whether VXC is permitted.",
				Computed:    true,
			},
			"vxc_auto_approval": schema.BoolAttribute{
				Description: "Whether VXC is auto approved.",
				Computed:    true,
			},
			"secondary_name": schema.StringAttribute{
				Description: "The secondary name of the MVE.",
				Computed:    true,
			},
			"company_name": schema.StringAttribute{
				Description: "The company name of the MVE.",
				Computed:    true,
			},
			"virtual": schema.BoolAttribute{
				Description: "Whether the MVE is virtual.",
				Computed:    true,
			},
			"buyout_port": schema.BoolAttribute{
				Description: "Whether the port is buyout.",
				Computed:    true,
			},
			"locked": schema.BoolAttribute{
				Description: "Whether the MVE is locked.",
				Computed:    true,
			},
			"admin_locked": schema.BoolAttribute{
				Description: "Whether the MVE is admin locked.",
				Computed:    true,
			},
			"cancelable": schema.BoolAttribute{
				Description: "Whether the MVE is cancelable.",
				Computed:    true,
			},
			"vendor": schema.StringAttribute{
				Description: "The vendor of the MVE.",
				Computed:    true,
			},
			"mve_size": schema.StringAttribute{
				Description: "The size of the MVE.",
				Computed:    true,
			},
			"attribute_tags": schema.MapAttribute{
				Description: "The attribute tags of the MVE.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"network_interfaces": schema.ListNestedAttribute{
				Description: "The network interfaces of the MVE.",
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"description": schema.StringAttribute{
							Description: "The description of the network interface.",
							Required:    true,
						},
						"vlan": schema.Int64Attribute{
							Description: "The VLAN of the network interface.",
							Required:    true,
						},
					},
				},
			},
			"vendor_config": schema.SingleNestedAttribute{
				Description: "The vendor configuration of the MVE.",
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"vendor": schema.StringAttribute{
						Description: "The vendor of the MVE.",
						Required:    true,
					},
					"image_id": schema.Int64Attribute{
						Description: "The image ID of the MVE.",
						Required:    true,
					},
					"product_size": schema.StringAttribute{
						Description: "The product size for the vendor config. Required for Aruba, Cisco, and Fortinet MVEs.",
						Optional:    true,
					},
					"account_name": schema.StringAttribute{
						Description: "The account name for the vendor config. Required for Aruba MVE.",
						Optional:    true,
					},
					"account_key": schema.StringAttribute{
						Description: "The account key for the vendor config. Required for Aruba MVE.",
						Optional:    true,
					},
					"admin_ssh_public_key": schema.StringAttribute{
						Description: "The admin SSH public key for the vendor config. Required for Cisco, Fortinet, Palo Alto, and Vmware MVEs.",
						Optional:    true,
					},
					"cloud_init": schema.StringAttribute{
						Description: "The cloud init for the vendor config. Required for Cisco MVE.",
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
						Description: "The director address for the vendor config. Required for Versa MVE.",
						Optional:    true,
					},
					"controller_address": schema.StringAttribute{
						Description: "The controller address for the vendor config. Required for Versa MVE.",
						Optional:    true,
					},
					"local_auth": schema.StringAttribute{
						Description: "The local auth for the vendor config. Required for Versa MVE.",
						Optional:    true,
					},
					"remote_auth": schema.StringAttribute{
						Description: "The remote auth for the vendor config. Required for Versa MVE.",
						Optional:    true,
					},
					"serial_number": schema.StringAttribute{
						Description: "The serial number for the vendor config. Required for Versa MVE.",
						Optional:    true,
					},
					"vco_address": schema.StringAttribute{
						Description: "The VCO address for the vendor config. Required for VMware MVE.",
						Optional:    true,
					},
					"vco_activation_code": schema.StringAttribute{
						Description: "The VCO activation code for the vendor config. Required for VMware MVE.",
						Optional:    true,
					},
				},
			},
			"resources": schema.SingleNestedAttribute{
				Description: "The resources associated with the MVE.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"interface": schema.SingleNestedAttribute{
						Description: "The port interface of the MVE.",
						Computed:    true,
						Attributes: map[string]schema.Attribute{
							"demarcation": schema.StringAttribute{
								Description: "The demarcation of the port interface.",
								Computed:    true,
							},
							"description": schema.StringAttribute{
								Description: "The description of the port interface.",
								Computed:    true,
							},
							"id": schema.Int64Attribute{
								Description: "The ID of the port interface.",
								Computed:    true,
							},
							"loa_template": schema.StringAttribute{
								Description: "The LOA template of the port interface.",
								Computed:    true,
							},
							"media": schema.StringAttribute{
								Description: "The media of the port interface.",
								Computed:    true,
							},
							"name": schema.StringAttribute{
								Description: "The name of the port interface.",
								Computed:    true,
							},
							"port_speed": schema.Int64Attribute{
								Description: "The port speed of the port interface.",
								Computed:    true,
							},
							"resource_name": schema.StringAttribute{
								Description: "The resource name of the port interface.",
								Computed:    true,
							},
							"resource_type": schema.StringAttribute{
								Description: "The resource type of the port interface.",
								Computed:    true,
							},
							"up": schema.BoolAttribute{
								Description: "Whether the port interface is up.",
								Computed:    true,
							},
						},
					},
					"virtual_machine": schema.ListNestedAttribute{
						Description: "The virtual machines associated with the MVE.",
						Computed:    true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"id": schema.Int64Attribute{
									Description: "The ID of the virtual machine.",
									Computed:    true,
								},
								"cpu_count": schema.Int64Attribute{
									Description: "The CPU count of the virtual machine.",
									Computed:    true,
								},
								"image": schema.SingleNestedAttribute{
									Description: "The image of the virtual machine.",
									Computed:    true,
									Attributes: map[string]schema.Attribute{
										"id": schema.Int64Attribute{
											Description: "The ID of the image.",
											Computed:    true,
										},
										"vendor": schema.StringAttribute{
											Description: "The vendor of the image.",
											Computed:    true,
										},
										"product": schema.StringAttribute{
											Description: "The product of the image.",
											Computed:    true,
										},
										"version": schema.StringAttribute{
											Description: "The version of the image.",
											Computed:    true,
										},
									},
								},
								"resource_type": schema.StringAttribute{
									Description: "The resource type of the virtual machine.",
									Computed:    true,
								},
								"up": schema.BoolAttribute{
									Description: "Whether the virtual machine is up.",
									Computed:    true,
								},
								"vnics": schema.ListNestedAttribute{
									Description: "The network interfaces of the virtual machine.",
									Computed:    true,
									NestedObject: schema.NestedAttributeObject{
										Attributes: map[string]schema.Attribute{
											"description": schema.StringAttribute{
												Description: "The description of the network interface.",
												Computed:    true,
											},
											"vlan": schema.Int64Attribute{
												Description: "The VLAN of the network interface.",
												Computed:    true,
											},
										},
									},
								},
							},
						},
					},
				},
			},
			// TODO - OTHER STRUCTS AND VENDOR CONFIGS
		},
	}
}

// Create a new resource.
func (r *mveResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan mveResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	mveReq := &megaport.BuyMVERequest{
		LocationID: int(plan.LocationID.ValueInt64()),
		Name:       plan.Name.ValueString(),
		Term:       int(plan.ContractTermMonths.ValueInt64()),

		WaitForProvision: true,
		WaitForTime:      10 * time.Minute,
	}

	if plan.VendorConfig == nil {
		resp.Diagnostics.AddError(
			"vendor config required", "vendor config required",
		)
	}
	mveReq.VendorConfig = toAPIVendorConfig(plan.VendorConfig)

	for _, vnic := range plan.NetworkInterfaces {
		mveReq.Vnics = append(mveReq.Vnics, vnic.toAPINetworkInterface())
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

	// update the plan with the MVE info
	plan.fromAPIMVE(mve)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read resource information.
func (r *mveResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state mveResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get refreshed MVE value from API
	mve, err := r.client.MVEService.GetMVE(ctx, state.UID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading MVE",
			"Could not read MVE with ID "+state.UID.ValueString()+": "+err.Error(),
		)
		return
	}

	state.fromAPIMVE(mve)

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *mveResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state mveResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Check on changes
	var name types.String
	if !plan.Name.Equal(state.Name) {
		name = plan.Name
	}

	_, err := r.client.MVEService.ModifyMVE(ctx, &megaport.ModifyMVERequest{
		MVEID:         state.UID.ValueString(),
		Name:          name.String(),
		WaitForUpdate: true,
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

	state.fromAPIMVE(updatedMVE)

	state.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags := resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *mveResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state mveResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete existing order
	_, err := r.client.MVEService.DeleteMVE(ctx, &megaport.DeleteMVERequest{
		MVEID: state.UID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting MVE",
			"Could not delete MVE, unexpected error: "+err.Error(),
		)
		return
	}
}

// Configure adds the provider configured client to the resource.
func (r *mveResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *mveResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Retrieve import ID and save to id attribute
	resource.ImportStatePassthroughID(ctx, path.Root("uid"), req, resp)
}
