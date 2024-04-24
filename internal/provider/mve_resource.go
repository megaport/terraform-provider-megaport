package provider

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
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
	LiveDate              types.String `tfsdk:"live_date"`
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

	VendorConfig types.Object `tfsdk:"vendor_config"`

	NetworkInterfaces types.List   `tfsdk:"vnics"`
	AttributeTags     types.Map    `tfsdk:"attribute_tags"`
	Resources         types.Object `tfsdk:"resources"`
}

// mveNetworkInterfaceModel represents a vNIC.
type mveNetworkInterfaceModel struct {
	Description types.String `tfsdk:"description"`
	VLAN        types.Int64  `tfsdk:"vlan"`
}

func toAPINetworkInterface(o types.Object) (*megaport.MVENetworkInterface, error) {
	vnic, err := strconv.Atoi(o.Attributes()["vlan"].String())
	if err != nil {
		return nil, err
	}
	return &megaport.MVENetworkInterface{
		Description: o.Attributes()["description"].String(),
		VLAN:        vnic,
	}, nil
}

// mveResourcesModel represents the resources associated with an MVE.
type mveResourcesModel struct {
	Interface       types.Object `tfsdk:"interface"`
	VirtualMachines types.List   `tfsdk:"virtual_machine"`
}

// mveVirtualMachineModel represents a virtual machine associated with an MVE.
type mveVirtualMachineModel struct {
	ID           types.Int64  `tfsdk:"id"`
	CpuCount     types.Int64  `tfsdk:"cpu_count"`
	Image        types.Object `tfsdk:"image"`
	ResourceType types.String `tfsdk:"resource_type"`
	Up           types.Bool   `tfsdk:"up"`
	Vnics        types.List   `tfsdk:"vnics"`
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

func (orm *mveResourceModel) fromAPIMVE(ctx context.Context, p *megaport.MVE) {
	orm.ID = types.Int64Value(int64(p.ID))
	orm.UID = types.StringValue(p.UID)
	orm.Name = types.StringValue(p.Name)
	orm.Type = types.StringValue(p.Type)
	orm.ProvisioningStatus = types.StringValue(p.ProvisioningStatus)
	orm.CreatedBy = types.StringValue(p.CreatedBy)
	orm.Market = types.StringValue(p.Market)
	orm.LocationID = types.Int64Value(int64(p.LocationID))
	orm.UsageAlgorithm = types.StringValue(p.UsageAlgorithm)
	orm.MarketplaceVisibility = types.BoolValue(p.MarketplaceVisibility)
	orm.VXCPermitted = types.BoolValue(p.VXCPermitted)
	orm.VXCAutoApproval = types.BoolValue(p.VXCAutoApproval)
	orm.SecondaryName = types.StringValue(p.SecondaryName)
	orm.CompanyUID = types.StringValue(p.CompanyUID)
	orm.CompanyName = types.StringValue(p.CompanyName)
	if p.CreateDate != nil {
		orm.CreateDate = types.StringValue(p.CreateDate.Format(time.RFC850))
	} else {
		orm.CreateDate = types.StringNull()
	}
	orm.CreateDate = types.StringValue(p.CreateDate.String())
	orm.TerminateDate = types.StringValue(p.TerminateDate.String())
	orm.LiveDate = types.StringValue(p.LiveDate.String())
	if p.ContractStartDate != nil {
		orm.ContractStartDate = types.StringValue(p.ContractStartDate.Format(time.RFC850))
	} else {
		orm.ContractStartDate = types.StringNull()
	}
	if p.ContractEndDate != nil {
		orm.ContractEndDate = types.StringValue(p.ContractEndDate.Format(time.RFC850))
	} else {
		orm.ContractEndDate = types.StringNull()
	}
	orm.ContractTermMonths = types.Int64Value(int64(p.ContractTermMonths))
	orm.Virtual = types.BoolValue(p.Virtual)
	orm.BuyoutPort = types.BoolValue(p.BuyoutPort)
	orm.Locked = types.BoolValue(p.Locked)
	orm.AdminLocked = types.BoolValue(p.AdminLocked)
	orm.Cancelable = types.BoolValue(p.Cancelable)
	orm.Vendor = types.StringValue(p.Vendor)
	orm.Size = types.StringValue(p.Size)

	if p.AttributeTags != nil {
		tags, _ := types.MapValueFrom(ctx, types.StringType, p.AttributeTags)
		orm.AttributeTags = tags
	}

	vnics := []types.Object{}
	for _, n := range p.NetworkInterfaces {
		model := &mveNetworkInterfaceModel{
			Description: types.StringValue(n.Description),
			VLAN:        types.Int64Value(int64(n.VLAN)),
		}
		vnic, _ := types.ObjectValueFrom(context.Background(), map[string]attr.Type{}, model)
		vnics = append(vnics, vnic)
	}
	orm.NetworkInterfaces, _ = types.ListValueFrom(context.Background(), types.ObjectType{}, vnics)

	if p.Resources != nil {
		resourcesModel := &mveResourcesModel{}
		if p.Resources.Interface != nil {
			interfaceModel := &portInterfaceModel{
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
			interfaceObject, _ := types.ObjectValueFrom(context.Background(), map[string]attr.Type{}, interfaceModel)
			resourcesModel.Interface = interfaceObject
		}
		virtualMachineObjects := []types.Object{}
		for _, vm := range p.Resources.VirtualMachines {
			vmModel := &mveVirtualMachineModel{
				ID:           types.Int64Value(int64(vm.ID)),
				CpuCount:     types.Int64Value(int64(vm.CpuCount)),
				ResourceType: types.StringValue(vm.ResourceType),
				Up:           types.BoolValue(vm.Up),
			}
			if vm.Image != nil {
				imageModel := &mveVirtualMachineImageModel{
					ID:      types.Int64Value(int64(vm.Image.ID)),
					Vendor:  types.StringValue(vm.Image.Vendor),
					Product: types.StringValue(vm.Image.Product),
					Version: types.StringValue(vm.Image.Version),
				}
				image, _ := types.ObjectValueFrom(context.Background(), map[string]attr.Type{}, imageModel)
				vmModel.Image = image
			}
			vnics := []types.Object{}
			for _, vnic := range vm.Vnics {
				model := &mveNetworkInterfaceModel{
					Description: types.StringValue(vnic.Description),
					VLAN:        types.Int64Value(int64(vnic.VLAN)),
				}
				vnic, _ := types.ObjectValueFrom(context.Background(), map[string]attr.Type{}, model)
				vnics = append(vnics, vnic)
			}
			vmModel.Vnics, _ = types.ListValueFrom(context.Background(), types.ObjectType{}, vnics)
			vmObject, _ := types.ObjectValueFrom(context.Background(), map[string]attr.Type{}, vmModel)
			virtualMachineObjects = append(virtualMachineObjects, vmObject)
		}
		virtualMachines, _ := types.ListValueFrom(context.Background(), types.ObjectType{}, virtualMachineObjects)
		resourcesModel.VirtualMachines = virtualMachines
		resources, _ := types.ObjectValueFrom(context.Background(), map[string]attr.Type{}, resourcesModel)
		orm.Resources = resources
	}
}

func toAPIVendorConfig(ctx context.Context, o types.Object) (megaport.VendorConfig, error) {
	vendor := o.Attributes()["vendor"].String()
	switch vendor {
	case "aruba":
		var cfg arubaConfigModel
		diag := o.As(ctx, &cfg, basetypes.ObjectAsOptions{})
		if diag.HasError() {
			return nil, errors.New("invalid vendor config")
		}
		return &megaport.ArubaConfig{
			Vendor:      vendor,
			ImageID:     int(cfg.ImageID.ValueInt64()),
			ProductSize: cfg.ProductSize.ValueString(),
			AccountName: cfg.AccountName.ValueString(),
			AccountKey:  cfg.AccountName.ValueString(),
		}, nil
	case "cisco":
		var cfg ciscoConfigModel
		diag := o.As(ctx, &cfg, basetypes.ObjectAsOptions{})
		if diag.HasError() {
			return nil, errors.New("invalid vendor config")
		}
		return &megaport.CiscoConfig{
			Vendor:            vendor,
			ImageID:           int(cfg.ImageID.ValueInt64()),
			ProductSize:       cfg.ProductSize.ValueString(),
			AdminSSHPublicKey: cfg.AdminSSHPublicKey.ValueString(),
			CloudInit:         cfg.CloudInit.ValueString(),
		}, nil
	case "fortinet":
		var cfg fortinetConfigModel
		diag := o.As(ctx, &cfg, basetypes.ObjectAsOptions{})
		if diag.HasError() {
			return nil, errors.New("invalid vendor config")
		}
		return &megaport.FortinetConfig{
			Vendor:            vendor,
			ImageID:           int(cfg.ImageID.ValueInt64()),
			ProductSize:       cfg.ProductSize.ValueString(),
			AdminSSHPublicKey: cfg.AdminSSHPublicKey.ValueString(),
			LicenseData:       cfg.LicenseData.ValueString(),
		}, nil
	case "palo_alto":
		var cfg paloAltoConfigModel
		diag := o.As(ctx, &cfg, basetypes.ObjectAsOptions{})
		if diag.HasError() {
			return nil, errors.New("invalid vendor config")
		}
		return &megaport.PaloAltoConfig{
			Vendor:            vendor,
			ImageID:           int(cfg.ImageID.ValueInt64()),
			ProductSize:       cfg.ProductSize.ValueString(),
			AdminSSHPublicKey: cfg.AdminSSHPublicKey.ValueString(),
			AdminPasswordHash: cfg.AdminPasswordHash.ValueString(),
			LicenseData:       cfg.LicenseData.ValueString(),
		}, nil
	case "versa":
		var cfg versaConfigModel
		diag := o.As(ctx, &cfg, basetypes.ObjectAsOptions{})
		if diag.HasError() {
			return nil, errors.New("invalid vendor config")
		}
		return &megaport.VersaConfig{
			Vendor:            vendor,
			ImageID:           int(cfg.ImageID.ValueInt64()),
			ProductSize:       cfg.ProductSize.ValueString(),
			DirectorAddress:   cfg.DirectorAddress.ValueString(),
			ControllerAddress: cfg.ControllerAddress.ValueString(),
			LocalAuth:         cfg.LocalAuth.ValueString(),
			RemoteAuth:        cfg.RemoteAuth.ValueString(),
			SerialNumber:      cfg.SerialNumber.ValueString(),
		}, nil
	case "vmware":
		var cfg vmwareConfig
		diag := o.As(ctx, &cfg, basetypes.ObjectAsOptions{})
		if diag.HasError() {
			return nil, errors.New("invalid vendor config")
		}
		return &megaport.VmwareConfig{
			Vendor:            vendor,
			ImageID:           int(cfg.ImageID.ValueInt64()),
			ProductSize:       cfg.ProductSize.ValueString(),
			AdminSSHPublicKey: cfg.AdminSSHPublicKey.ValueString(),
			VcoAddress:        cfg.VcoAddress.ValueString(),
			VcoActivationCode: cfg.VcoActivationCode.ValueString(),
		}, nil
	}
	return nil, errors.New("unknown vendor")
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
			"contract_start_date": schema.StringAttribute{
				Description: "The contract start date of the MVE.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"contract_end_date": schema.StringAttribute{
				Description: "The contract end date of the MVE.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
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
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"image_id": schema.Int64Attribute{
						Description: "The image ID of the MVE.",
						Required:    true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.RequiresReplace(),
						},
					},
					"product_size": schema.StringAttribute{
						Description: "The product size for the vendor config.",
						Required:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
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
								PlanModifiers: []planmodifier.Int64{
									int64planmodifier.UseStateForUnknown(),
								},
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
									PlanModifiers: []planmodifier.Int64{
										int64planmodifier.UseStateForUnknown(),
									},
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
											PlanModifiers: []planmodifier.Int64{
												int64planmodifier.UseStateForUnknown(),
											},
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

	if plan.VendorConfig.IsNull() {
		resp.Diagnostics.AddError(
			"vendor config required", "vendor config required",
		)
	}
	vendorConfig, err := toAPIVendorConfig(ctx, plan.VendorConfig)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Vendor Config",
			"Invalid Vendor Config: "+err.Error(),
		)
		return
	}
	mveReq.VendorConfig = vendorConfig

	for _, vnic := range plan.NetworkInterfaces.Elements() {
		vnic, ok := vnic.(types.Object)
		if !ok {
			resp.Diagnostics.AddError(
				"Invalid VNIC",
				"Invalid VNIC",
			)
			return
		}
		toAPI, err := toAPINetworkInterface(vnic)
		if err != nil {
			resp.Diagnostics.AddError(
				"Invalid VNIC",
				"Invalid VNIC: "+err.Error(),
			)
			return
		}
		mveReq.Vnics = append(mveReq.Vnics, *toAPI)
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
	plan.fromAPIMVE(ctx, mve)
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

	state.fromAPIMVE(ctx, mve)

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

	state.fromAPIMVE(ctx, updatedMVE)

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
	resource.ImportStatePassthroughID(ctx, path.Root("product_uid"), req, resp)
}
