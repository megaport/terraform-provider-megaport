package provider

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
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
	_ resource.Resource                = &mveResource{}
	_ resource.ResourceWithConfigure   = &mveResource{}
	_ resource.ResourceWithImportState = &mveResource{}

	vnicAttrs = map[string]attr.Type{
		"description": types.StringType,
		"vlan":        types.Int64Type,
	}
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
	VXCPermitted          types.Bool   `tfsdk:"vxc_permitted"`
	VXCAutoApproval       types.Bool   `tfsdk:"vxc_auto_approval"`
	SecondaryName         types.String `tfsdk:"secondary_name"`
	CompanyUID            types.String `tfsdk:"company_uid"`
	CompanyName           types.String `tfsdk:"company_name"`
	ContractStartDate     types.String `tfsdk:"contract_start_date"`
	ContractEndDate       types.String `tfsdk:"contract_end_date"`
	ContractTermMonths    types.Int64  `tfsdk:"contract_term_months"`
	PromoCode             types.String `tfsdk:"promo_code"`
	CostCentre            types.String `tfsdk:"cost_centre"`
	DiversityZone         types.String `tfsdk:"diversity_zone"`

	Virtual     types.Bool `tfsdk:"virtual"`
	BuyoutPort  types.Bool `tfsdk:"buyout_port"`
	Locked      types.Bool `tfsdk:"locked"`
	AdminLocked types.Bool `tfsdk:"admin_locked"`
	Cancelable  types.Bool `tfsdk:"cancelable"`

	Vendor types.String `tfsdk:"vendor"`
	Size   types.String `tfsdk:"mve_size"`

	VendorConfig types.Object `tfsdk:"vendor_config"`

	NetworkInterfaces types.List `tfsdk:"vnics"`
	AttributeTags     types.Map  `tfsdk:"attribute_tags"`

	ResourceTags types.Map `tfsdk:"resource_tags"`
}

// mveNetworkInterfaceModel represents a vNIC.
type mveNetworkInterfaceModel struct {
	Description types.String `tfsdk:"description"`
	VLAN        types.Int64  `tfsdk:"vlan"`
}

func toAPINetworkInterface(orm *mveNetworkInterfaceModel) *megaport.MVENetworkInterface {
	return &megaport.MVENetworkInterface{
		Description: orm.Description.ValueString(),
	}
}

// vendorConfigModel represents the vendor configuration for an MVE.
type vendorConfigModel struct {
	Vendor             types.String `tfsdk:"vendor"`
	ImageID            types.Int64  `tfsdk:"image_id"`
	ProductSize        types.String `tfsdk:"product_size"`
	MVELabel           types.String `tfsdk:"mve_label"`
	AccountName        types.String `tfsdk:"account_name"`
	AccountKey         types.String `tfsdk:"account_key"`
	AdminSSHPublicKey  types.String `tfsdk:"admin_ssh_public_key"`
	CloudInit          types.String `tfsdk:"cloud_init"`
	LicenseData        types.String `tfsdk:"license_data"`
	AdminPasswordHash  types.String `tfsdk:"admin_password_hash"`
	DirectorAddress    types.String `tfsdk:"director_address"`
	ControllerAddress  types.String `tfsdk:"controller_address"`
	LocalAuth          types.String `tfsdk:"local_auth"`
	RemoteAuth         types.String `tfsdk:"remote_auth"`
	ManageLocally      types.Bool   `tfsdk:"manage_locally"`
	SerialNumber       types.String `tfsdk:"serial_number"`
	SSHPublicKey       types.String `tfsdk:"ssh_public_key"`
	SystemTag          types.String `tfsdk:"system_tag"`
	VcoAddress         types.String `tfsdk:"vco_address"`
	VcoActivationCode  types.String `tfsdk:"vco_activation_code"`
	Token              types.String `tfsdk:"token"`
	FMCIPAddress       types.String `tfsdk:"fmc_ip_address"`
	FMCRegistrationKey types.String `tfsdk:"fmc_registration_key"`
	FMCNatID           types.String `tfsdk:"fmc_nat_id"`
	IONKey             types.String `tfsdk:"ion_key"`
	SecretKey          types.String `tfsdk:"secret_key"`
}

func (orm *mveResourceModel) fromAPIMVE(ctx context.Context, p *megaport.MVE, tags map[string]string) diag.Diagnostics {
	apiDiags := diag.Diagnostics{}
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
	orm.ContractTermMonths = types.Int64Value(int64(p.ContractTermMonths))
	orm.Virtual = types.BoolValue(p.Virtual)
	orm.BuyoutPort = types.BoolValue(p.BuyoutPort)
	orm.Locked = types.BoolValue(p.Locked)
	orm.AdminLocked = types.BoolValue(p.AdminLocked)
	orm.Cancelable = types.BoolValue(p.Cancelable)
	orm.Vendor = types.StringValue(p.Vendor)
	orm.Size = types.StringValue(p.Size)
	orm.LiveDate = types.StringValue("")
	orm.TerminateDate = types.StringValue("")
	orm.CostCentre = types.StringValue(p.CostCentre)
	orm.DiversityZone = types.StringValue(p.DiversityZone)

	if p.CreateDate != nil {
		orm.CreateDate = types.StringValue(p.CreateDate.Format(time.RFC850))
	} else {
		orm.CreateDate = types.StringValue("")
	}
	if p.TerminateDate != nil {
		orm.TerminateDate = types.StringValue(p.TerminateDate.Format(time.RFC850))
	}
	if p.LiveDate != nil {
		orm.LiveDate = types.StringValue(p.LiveDate.Format(time.RFC850))
	}
	if p.ContractStartDate != nil {
		orm.ContractStartDate = types.StringValue(p.ContractStartDate.Format(time.RFC850))
	} else {
		orm.ContractStartDate = types.StringValue("")
	}
	if p.ContractEndDate != nil {
		orm.ContractEndDate = types.StringValue(p.ContractEndDate.Format(time.RFC850))
	} else {
		orm.ContractEndDate = types.StringValue("")
	}

	if p.AttributeTags != nil {
		tags, tagDiags := types.MapValueFrom(ctx, types.StringType, p.AttributeTags)
		apiDiags = append(apiDiags, tagDiags...)
		orm.AttributeTags = tags
	} else {
		orm.AttributeTags = types.MapNull(types.StringType)
	}

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

func toAPIVendorConfig(v *vendorConfigModel) (megaport.VendorConfig, diag.Diagnostics) {
	apiDiags := diag.Diagnostics{}
	vendor := strings.ToLower(v.Vendor.ValueString()) // Allow for uppercase vendor names for more flexibility.
	switch vendor {
	case "6wind":
		vsrConfig := &megaport.SixwindVSRConfig{
			Vendor:       v.Vendor.ValueString(),
			ImageID:      int(v.ImageID.ValueInt64()),
			ProductSize:  v.ProductSize.ValueString(),
			MVELabel:     v.MVELabel.ValueString(),
			SSHPublicKey: v.SSHPublicKey.ValueString(),
		}
		return vsrConfig, apiDiags
	case "aruba":
		arubaConfig := &megaport.ArubaConfig{
			Vendor:      v.Vendor.ValueString(),
			ImageID:     int(v.ImageID.ValueInt64()),
			ProductSize: v.ProductSize.ValueString(),
			MVELabel:    v.MVELabel.ValueString(),
			AccountName: v.AccountName.ValueString(),
			AccountKey:  v.AccountKey.ValueString(),
			SystemTag:   v.SystemTag.ValueString(),
		}
		return arubaConfig, apiDiags
	case "aviatrix":
		aviatrixConfig := &megaport.AviatrixConfig{
			Vendor:      v.Vendor.ValueString(),
			ImageID:     int(v.ImageID.ValueInt64()),
			ProductSize: v.ProductSize.ValueString(),
			MVELabel:    v.MVELabel.ValueString(),
			CloudInit:   v.CloudInit.ValueString(),
		}
		return aviatrixConfig, apiDiags
	case "cisco":
		ciscoConfig := &megaport.CiscoConfig{
			Vendor:             v.Vendor.ValueString(),
			ImageID:            int(v.ImageID.ValueInt64()),
			ProductSize:        v.ProductSize.ValueString(),
			MVELabel:           v.MVELabel.ValueString(),
			AdminSSHPublicKey:  v.AdminSSHPublicKey.ValueString(),
			SSHPublicKey:       v.SSHPublicKey.ValueString(),
			ManageLocally:      v.ManageLocally.ValueBool(),
			CloudInit:          v.CloudInit.ValueString(),
			FMCIPAddress:       v.FMCIPAddress.ValueString(),
			FMCNatID:           v.FMCNatID.ValueString(),
			FMCRegistrationKey: v.FMCRegistrationKey.ValueString(),
		}
		return ciscoConfig, apiDiags
	case "fortinet":
		fortinetConfig := &megaport.FortinetConfig{
			Vendor:            v.Vendor.ValueString(),
			ImageID:           int(v.ImageID.ValueInt64()),
			ProductSize:       v.ProductSize.ValueString(),
			MVELabel:          v.MVELabel.ValueString(),
			AdminSSHPublicKey: v.AdminSSHPublicKey.ValueString(),
			SSHPublicKey:      v.SSHPublicKey.ValueString(),
			LicenseData:       v.LicenseData.ValueString(),
		}
		return fortinetConfig, apiDiags
	case "palo_alto":
		paloAltoConfig := &megaport.PaloAltoConfig{
			Vendor:            v.Vendor.ValueString(),
			ImageID:           int(v.ImageID.ValueInt64()),
			ProductSize:       v.ProductSize.ValueString(),
			MVELabel:          v.MVELabel.ValueString(),
			SSHPublicKey:      v.SSHPublicKey.ValueString(),
			AdminPasswordHash: v.AdminPasswordHash.ValueString(),
			LicenseData:       v.LicenseData.ValueString(),
		}
		return paloAltoConfig, apiDiags
	case "prisma":
		prismaConfig := &megaport.PrismaConfig{
			Vendor:      v.Vendor.ValueString(),
			ImageID:     int(v.ImageID.ValueInt64()),
			ProductSize: v.ProductSize.ValueString(),
			MVELabel:    v.MVELabel.ValueString(),
			IONKey:      v.IONKey.ValueString(),
			SecretKey:   v.SecretKey.ValueString(),
		}
		return prismaConfig, apiDiags
	case "versa":
		versaConfig := &megaport.VersaConfig{
			Vendor:            v.Vendor.ValueString(),
			ImageID:           int(v.ImageID.ValueInt64()),
			ProductSize:       v.ProductSize.ValueString(),
			MVELabel:          v.MVELabel.ValueString(),
			DirectorAddress:   v.DirectorAddress.ValueString(),
			ControllerAddress: v.ControllerAddress.ValueString(),
			LocalAuth:         v.LocalAuth.ValueString(),
			RemoteAuth:        v.RemoteAuth.ValueString(),
			SerialNumber:      v.SerialNumber.ValueString(),
		}
		return versaConfig, apiDiags
	case "vmware":
		vmwareConfig := &megaport.VmwareConfig{
			Vendor:            v.Vendor.ValueString(),
			ImageID:           int(v.ImageID.ValueInt64()),
			ProductSize:       v.ProductSize.ValueString(),
			MVELabel:          v.MVELabel.ValueString(),
			AdminSSHPublicKey: v.AdminSSHPublicKey.ValueString(),
			SSHPublicKey:      v.SSHPublicKey.ValueString(),
			VcoAddress:        v.VcoAddress.ValueString(),
			VcoActivationCode: v.VcoActivationCode.ValueString(),
		}
		return vmwareConfig, apiDiags
	case "meraki":
		merakiConfig := &megaport.MerakiConfig{
			Vendor:      v.Vendor.ValueString(),
			ImageID:     int(v.ImageID.ValueInt64()),
			ProductSize: v.ProductSize.ValueString(),
			MVELabel:    v.MVELabel.ValueString(),
			Token:       v.Token.ValueString(),
		}
		return merakiConfig, apiDiags
	}
	apiDiags.AddError("vendor not supported",
		"vendor not supported")
	return nil, apiDiags
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
		Description: "Megaport Virtual Edge (MVE) Resource for Megaport Terraform provider. This resource allows you to create, modify, and delete Megaport MVEs. Megaport Virtual Edge (MVE) is an on-demand, vendor-neutral Network Function Virtualization (NFV) platform that provides virtual infrastructure for network services at the edge of Megaport’s global software-defined network (SDN). Network technologies such as SD-WAN and NGFW are hosted directly on Megaport’s global network via Megaport Virtual Edge.",
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
			},
			"live_date": schema.StringAttribute{
				Description: "The date the MVE went live.",
				Computed:    true,
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
			"contract_start_date": schema.StringAttribute{
				Description: "The contract start date of the MVE.",
				Computed:    true,
			},
			"contract_end_date": schema.StringAttribute{
				Description: "The contract end date of the MVE.",
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
			"secondary_name": schema.StringAttribute{
				Description: "The secondary name of the MVE.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
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
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"buyout_port": schema.BoolAttribute{
				Description: "Whether the port is buyout.",
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
			"attribute_tags": schema.MapAttribute{
				Description: "The attribute tags of the MVE.",
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
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
func (r *mveResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan mveResourceModel
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

func (r *mveResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state mveResourceModel

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

	state.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags := resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *mveResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve the state
	var state mveResourceModel
	stateDiags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(stateDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Call the API to delete the resource
	productUID := state.UID.ValueString()
	_, err := r.client.MVEService.DeleteMVE(ctx, &megaport.DeleteMVERequest{
		MVEID: productUID,
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

func (r *mveResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Get the plan and state
	var plan, state mveResourceModel
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
