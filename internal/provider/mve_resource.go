package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
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
	_ resource.ResourceWithMoveState   = &mveResource{}

	vnicAttrs = map[string]attr.Type{
		"description": types.StringType,
	}
)

// mveResourceModel maps the resource schema data.
type mveResourceModel struct {
	UID                   types.String `tfsdk:"product_uid"`
	Name                  types.String `tfsdk:"product_name"`
	CreatedBy             types.String `tfsdk:"created_by"`
	TerminateDate         types.String `tfsdk:"terminate_date"`
	Market                types.String `tfsdk:"market"`
	LocationID            types.Int64  `tfsdk:"location_id"`
	MarketplaceVisibility types.Bool   `tfsdk:"marketplace_visibility"`
	VXCAutoApproval       types.Bool   `tfsdk:"vxc_auto_approval"`
	CompanyUID            types.String `tfsdk:"company_uid"`
	ContractTermMonths    types.Int64  `tfsdk:"contract_term_months"`
	PromoCode             types.String `tfsdk:"promo_code"`
	CostCentre            types.String `tfsdk:"cost_centre"`
	DiversityZone         types.String `tfsdk:"diversity_zone"`

	Vendor types.String `tfsdk:"vendor"`
	Size   types.String `tfsdk:"mve_size"`

	// Per-vendor config blocks (exactly one must be set).
	ArubaConfig    types.Object `tfsdk:"aruba_config"`
	AviatrixConfig types.Object `tfsdk:"aviatrix_config"`
	CiscoConfig    types.Object `tfsdk:"cisco_config"`
	FortinetConfig types.Object `tfsdk:"fortinet_config"`
	MerakiConfig   types.Object `tfsdk:"meraki_config"`
	PaloAltoConfig types.Object `tfsdk:"palo_alto_config"`
	PrismaConfig   types.Object `tfsdk:"prisma_config"`
	SixwindConfig  types.Object `tfsdk:"sixwind_config"`
	VersaConfig    types.Object `tfsdk:"versa_config"`
	VmwareConfig   types.Object `tfsdk:"vmware_config"`

	NetworkInterfaces types.List `tfsdk:"vnics"`
	AttributeTags     types.Map  `tfsdk:"attribute_tags"`
	ResourceTags      types.Map  `tfsdk:"resource_tags"`
}

// mveNetworkInterfaceModel represents a vNIC.
type mveNetworkInterfaceModel struct {
	Description types.String `tfsdk:"description"`
}

// Per-vendor config model structs.

type arubaConfigModel struct {
	ImageID     types.Int64  `tfsdk:"image_id"`
	ProductSize types.String `tfsdk:"product_size"`
	MVELabel    types.String `tfsdk:"mve_label"`
	AccountName types.String `tfsdk:"account_name"`
	AccountKey  types.String `tfsdk:"account_key"`
	SystemTag   types.String `tfsdk:"system_tag"`
}

type aviatrixConfigModel struct {
	ImageID     types.Int64  `tfsdk:"image_id"`
	ProductSize types.String `tfsdk:"product_size"`
	MVELabel    types.String `tfsdk:"mve_label"`
	CloudInit   types.String `tfsdk:"cloud_init"`
}

type ciscoConfigModel struct {
	ImageID            types.Int64  `tfsdk:"image_id"`
	ProductSize        types.String `tfsdk:"product_size"`
	MVELabel           types.String `tfsdk:"mve_label"`
	AdminSSHPublicKey  types.String `tfsdk:"admin_ssh_public_key"`
	SSHPublicKey       types.String `tfsdk:"ssh_public_key"`
	ManageLocally      types.Bool   `tfsdk:"manage_locally"`
	CloudInit          types.String `tfsdk:"cloud_init"`
	FMCIPAddress       types.String `tfsdk:"fmc_ip_address"`
	FMCRegistrationKey types.String `tfsdk:"fmc_registration_key"`
	FMCNatID           types.String `tfsdk:"fmc_nat_id"`
}

type fortinetConfigModel struct {
	ImageID           types.Int64  `tfsdk:"image_id"`
	ProductSize       types.String `tfsdk:"product_size"`
	MVELabel          types.String `tfsdk:"mve_label"`
	AdminSSHPublicKey types.String `tfsdk:"admin_ssh_public_key"`
	SSHPublicKey      types.String `tfsdk:"ssh_public_key"`
	LicenseData       types.String `tfsdk:"license_data"`
}

type merakiConfigModel struct {
	ImageID     types.Int64  `tfsdk:"image_id"`
	ProductSize types.String `tfsdk:"product_size"`
	MVELabel    types.String `tfsdk:"mve_label"`
	Token       types.String `tfsdk:"token"`
}

type paloAltoConfigModel struct {
	ImageID           types.Int64  `tfsdk:"image_id"`
	ProductSize       types.String `tfsdk:"product_size"`
	MVELabel          types.String `tfsdk:"mve_label"`
	AdminSSHPublicKey types.String `tfsdk:"admin_ssh_public_key"`
	SSHPublicKey      types.String `tfsdk:"ssh_public_key"`
	AdminPasswordHash types.String `tfsdk:"admin_password_hash"`
	LicenseData       types.String `tfsdk:"license_data"`
}

type prismaConfigModel struct {
	ImageID     types.Int64  `tfsdk:"image_id"`
	ProductSize types.String `tfsdk:"product_size"`
	MVELabel    types.String `tfsdk:"mve_label"`
	IONKey      types.String `tfsdk:"ion_key"`
	SecretKey   types.String `tfsdk:"secret_key"`
}

type sixwindConfigModel struct {
	ImageID      types.Int64  `tfsdk:"image_id"`
	ProductSize  types.String `tfsdk:"product_size"`
	MVELabel     types.String `tfsdk:"mve_label"`
	SSHPublicKey types.String `tfsdk:"ssh_public_key"`
}

type versaConfigModel struct {
	ImageID           types.Int64  `tfsdk:"image_id"`
	ProductSize       types.String `tfsdk:"product_size"`
	MVELabel          types.String `tfsdk:"mve_label"`
	DirectorAddress   types.String `tfsdk:"director_address"`
	ControllerAddress types.String `tfsdk:"controller_address"`
	LocalAuth         types.String `tfsdk:"local_auth"`
	RemoteAuth        types.String `tfsdk:"remote_auth"`
	SerialNumber      types.String `tfsdk:"serial_number"`
}

type vmwareConfigModel struct {
	ImageID           types.Int64  `tfsdk:"image_id"`
	ProductSize       types.String `tfsdk:"product_size"`
	MVELabel          types.String `tfsdk:"mve_label"`
	AdminSSHPublicKey types.String `tfsdk:"admin_ssh_public_key"`
	SSHPublicKey      types.String `tfsdk:"ssh_public_key"`
	VcoAddress        types.String `tfsdk:"vco_address"`
	VcoActivationCode types.String `tfsdk:"vco_activation_code"`
}

// Package-level attr.Type maps for each vendor config.
var (
	arubaConfigAttrs = map[string]attr.Type{
		"image_id": types.Int64Type, "product_size": types.StringType,
		"mve_label": types.StringType, "account_name": types.StringType,
		"account_key": types.StringType, "system_tag": types.StringType,
	}
	aviatrixConfigAttrs = map[string]attr.Type{
		"image_id": types.Int64Type, "product_size": types.StringType,
		"mve_label": types.StringType, "cloud_init": types.StringType,
	}
	ciscoConfigAttrs = map[string]attr.Type{
		"image_id": types.Int64Type, "product_size": types.StringType,
		"mve_label": types.StringType, "admin_ssh_public_key": types.StringType,
		"ssh_public_key": types.StringType, "manage_locally": types.BoolType,
		"cloud_init": types.StringType, "fmc_ip_address": types.StringType,
		"fmc_registration_key": types.StringType, "fmc_nat_id": types.StringType,
	}
	fortinetConfigAttrs = map[string]attr.Type{
		"image_id": types.Int64Type, "product_size": types.StringType,
		"mve_label": types.StringType, "admin_ssh_public_key": types.StringType,
		"ssh_public_key": types.StringType, "license_data": types.StringType,
	}
	merakiConfigAttrs = map[string]attr.Type{
		"image_id": types.Int64Type, "product_size": types.StringType,
		"mve_label": types.StringType, "token": types.StringType,
	}
	paloAltoConfigAttrs = map[string]attr.Type{
		"image_id": types.Int64Type, "product_size": types.StringType,
		"mve_label": types.StringType, "admin_ssh_public_key": types.StringType,
		"ssh_public_key": types.StringType, "admin_password_hash": types.StringType,
		"license_data": types.StringType,
	}
	prismaConfigAttrs = map[string]attr.Type{
		"image_id": types.Int64Type, "product_size": types.StringType,
		"mve_label": types.StringType, "ion_key": types.StringType,
		"secret_key": types.StringType,
	}
	sixwindConfigAttrs = map[string]attr.Type{
		"image_id": types.Int64Type, "product_size": types.StringType,
		"mve_label": types.StringType, "ssh_public_key": types.StringType,
	}
	versaConfigAttrs = map[string]attr.Type{
		"image_id": types.Int64Type, "product_size": types.StringType,
		"mve_label": types.StringType, "director_address": types.StringType,
		"controller_address": types.StringType, "local_auth": types.StringType,
		"remote_auth": types.StringType, "serial_number": types.StringType,
	}
	vmwareConfigAttrs = map[string]attr.Type{
		"image_id": types.Int64Type, "product_size": types.StringType,
		"mve_label": types.StringType, "admin_ssh_public_key": types.StringType,
		"ssh_public_key": types.StringType, "vco_address": types.StringType,
		"vco_activation_code": types.StringType,
	}
)

// mveVendorConfigPaths lists path expressions for all per-vendor config blocks,
// used with ExactlyOneOf validators.
var mveVendorConfigPaths = []path.Expression{
	path.MatchRoot("aruba_config"),
	path.MatchRoot("aviatrix_config"),
	path.MatchRoot("cisco_config"),
	path.MatchRoot("fortinet_config"),
	path.MatchRoot("meraki_config"),
	path.MatchRoot("palo_alto_config"),
	path.MatchRoot("prisma_config"),
	path.MatchRoot("sixwind_config"),
	path.MatchRoot("versa_config"),
	path.MatchRoot("vmware_config"),
}

func toAPINetworkInterface(orm *mveNetworkInterfaceModel) *megaport.MVENetworkInterface {
	return &megaport.MVENetworkInterface{
		Description: orm.Description.ValueString(),
	}
}

// toAPIVendorConfigFromModel converts the per-vendor config block to a megaport.VendorConfig.
func toAPIVendorConfigFromModel(ctx context.Context, m *mveResourceModel) (megaport.VendorConfig, diag.Diagnostics) {
	var diags diag.Diagnostics
	switch {
	case !m.ArubaConfig.IsNull():
		var cfg arubaConfigModel
		diags.Append(m.ArubaConfig.As(ctx, &cfg, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return nil, diags
		}
		return &megaport.ArubaConfig{
			Vendor:      "aruba",
			ImageID:     int(cfg.ImageID.ValueInt64()),
			ProductSize: cfg.ProductSize.ValueString(),
			MVELabel:    cfg.MVELabel.ValueString(),
			AccountName: cfg.AccountName.ValueString(),
			AccountKey:  cfg.AccountKey.ValueString(),
			SystemTag:   cfg.SystemTag.ValueString(),
		}, diags
	case !m.AviatrixConfig.IsNull():
		var cfg aviatrixConfigModel
		diags.Append(m.AviatrixConfig.As(ctx, &cfg, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return nil, diags
		}
		return &megaport.AviatrixConfig{
			Vendor:      "aviatrix",
			ImageID:     int(cfg.ImageID.ValueInt64()),
			ProductSize: cfg.ProductSize.ValueString(),
			MVELabel:    cfg.MVELabel.ValueString(),
			CloudInit:   cfg.CloudInit.ValueString(),
		}, diags
	case !m.CiscoConfig.IsNull():
		var cfg ciscoConfigModel
		diags.Append(m.CiscoConfig.As(ctx, &cfg, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return nil, diags
		}
		return &megaport.CiscoConfig{
			Vendor:             "cisco",
			ImageID:            int(cfg.ImageID.ValueInt64()),
			ProductSize:        cfg.ProductSize.ValueString(),
			MVELabel:           cfg.MVELabel.ValueString(),
			AdminSSHPublicKey:  cfg.AdminSSHPublicKey.ValueString(),
			SSHPublicKey:       cfg.SSHPublicKey.ValueString(),
			ManageLocally:      cfg.ManageLocally.ValueBool(),
			CloudInit:          cfg.CloudInit.ValueString(),
			FMCIPAddress:       cfg.FMCIPAddress.ValueString(),
			FMCRegistrationKey: cfg.FMCRegistrationKey.ValueString(),
			FMCNatID:           cfg.FMCNatID.ValueString(),
		}, diags
	case !m.FortinetConfig.IsNull():
		var cfg fortinetConfigModel
		diags.Append(m.FortinetConfig.As(ctx, &cfg, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return nil, diags
		}
		return &megaport.FortinetConfig{
			Vendor:            "fortinet",
			ImageID:           int(cfg.ImageID.ValueInt64()),
			ProductSize:       cfg.ProductSize.ValueString(),
			MVELabel:          cfg.MVELabel.ValueString(),
			AdminSSHPublicKey: cfg.AdminSSHPublicKey.ValueString(),
			SSHPublicKey:      cfg.SSHPublicKey.ValueString(),
			LicenseData:       cfg.LicenseData.ValueString(),
		}, diags
	case !m.MerakiConfig.IsNull():
		var cfg merakiConfigModel
		diags.Append(m.MerakiConfig.As(ctx, &cfg, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return nil, diags
		}
		return &megaport.MerakiConfig{
			Vendor:      "meraki",
			ImageID:     int(cfg.ImageID.ValueInt64()),
			ProductSize: cfg.ProductSize.ValueString(),
			MVELabel:    cfg.MVELabel.ValueString(),
			Token:       cfg.Token.ValueString(),
		}, diags
	case !m.PaloAltoConfig.IsNull():
		var cfg paloAltoConfigModel
		diags.Append(m.PaloAltoConfig.As(ctx, &cfg, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return nil, diags
		}
		return &megaport.PaloAltoConfig{
			Vendor:            "palo_alto",
			ImageID:           int(cfg.ImageID.ValueInt64()),
			ProductSize:       cfg.ProductSize.ValueString(),
			MVELabel:          cfg.MVELabel.ValueString(),
			AdminSSHPublicKey: cfg.AdminSSHPublicKey.ValueString(),
			SSHPublicKey:      cfg.SSHPublicKey.ValueString(),
			AdminPasswordHash: cfg.AdminPasswordHash.ValueString(),
			LicenseData:       cfg.LicenseData.ValueString(),
		}, diags
	case !m.PrismaConfig.IsNull():
		var cfg prismaConfigModel
		diags.Append(m.PrismaConfig.As(ctx, &cfg, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return nil, diags
		}
		return &megaport.PrismaConfig{
			Vendor:      "prisma",
			ImageID:     int(cfg.ImageID.ValueInt64()),
			ProductSize: cfg.ProductSize.ValueString(),
			MVELabel:    cfg.MVELabel.ValueString(),
			IONKey:      cfg.IONKey.ValueString(),
			SecretKey:   cfg.SecretKey.ValueString(),
		}, diags
	case !m.SixwindConfig.IsNull():
		var cfg sixwindConfigModel
		diags.Append(m.SixwindConfig.As(ctx, &cfg, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return nil, diags
		}
		return &megaport.SixwindVSRConfig{
			Vendor:       "6wind",
			ImageID:      int(cfg.ImageID.ValueInt64()),
			ProductSize:  cfg.ProductSize.ValueString(),
			MVELabel:     cfg.MVELabel.ValueString(),
			SSHPublicKey: cfg.SSHPublicKey.ValueString(),
		}, diags
	case !m.VersaConfig.IsNull():
		var cfg versaConfigModel
		diags.Append(m.VersaConfig.As(ctx, &cfg, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return nil, diags
		}
		return &megaport.VersaConfig{
			Vendor:            "versa",
			ImageID:           int(cfg.ImageID.ValueInt64()),
			ProductSize:       cfg.ProductSize.ValueString(),
			MVELabel:          cfg.MVELabel.ValueString(),
			DirectorAddress:   cfg.DirectorAddress.ValueString(),
			ControllerAddress: cfg.ControllerAddress.ValueString(),
			LocalAuth:         cfg.LocalAuth.ValueString(),
			RemoteAuth:        cfg.RemoteAuth.ValueString(),
			SerialNumber:      cfg.SerialNumber.ValueString(),
		}, diags
	case !m.VmwareConfig.IsNull():
		var cfg vmwareConfigModel
		diags.Append(m.VmwareConfig.As(ctx, &cfg, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return nil, diags
		}
		return &megaport.VmwareConfig{
			Vendor:            "vmware",
			ImageID:           int(cfg.ImageID.ValueInt64()),
			ProductSize:       cfg.ProductSize.ValueString(),
			MVELabel:          cfg.MVELabel.ValueString(),
			AdminSSHPublicKey: cfg.AdminSSHPublicKey.ValueString(),
			SSHPublicKey:      cfg.SSHPublicKey.ValueString(),
			VcoAddress:        cfg.VcoAddress.ValueString(),
			VcoActivationCode: cfg.VcoActivationCode.ValueString(),
		}, diags
	default:
		diags.AddError("No vendor config set", "Exactly one vendor config block must be set")
		return nil, diags
	}
}

// allVendorConfigsNull returns true when all 10 vendor config blocks are null.
func allVendorConfigsNull(m mveResourceModel) bool {
	return m.ArubaConfig.IsNull() && m.AviatrixConfig.IsNull() && m.CiscoConfig.IsNull() &&
		m.FortinetConfig.IsNull() && m.MerakiConfig.IsNull() && m.PaloAltoConfig.IsNull() &&
		m.PrismaConfig.IsNull() && m.SixwindConfig.IsNull() && m.VersaConfig.IsNull() &&
		m.VmwareConfig.IsNull()
}

// copyVendorConfigs copies all 10 vendor config blocks from src to dst.
func copyVendorConfigs(dst, src *mveResourceModel) {
	dst.ArubaConfig = src.ArubaConfig
	dst.AviatrixConfig = src.AviatrixConfig
	dst.CiscoConfig = src.CiscoConfig
	dst.FortinetConfig = src.FortinetConfig
	dst.MerakiConfig = src.MerakiConfig
	dst.PaloAltoConfig = src.PaloAltoConfig
	dst.PrismaConfig = src.PrismaConfig
	dst.SixwindConfig = src.SixwindConfig
	dst.VersaConfig = src.VersaConfig
	dst.VmwareConfig = src.VmwareConfig
}

// vendorNameFromModel returns the canonical vendor name string based on which
// config block is non-null in the model.
func vendorNameFromModel(m mveResourceModel) string {
	switch {
	case !m.ArubaConfig.IsNull():
		return "aruba"
	case !m.AviatrixConfig.IsNull():
		return "aviatrix"
	case !m.CiscoConfig.IsNull():
		return "cisco"
	case !m.FortinetConfig.IsNull():
		return "fortinet"
	case !m.MerakiConfig.IsNull():
		return "meraki"
	case !m.PaloAltoConfig.IsNull():
		return "palo_alto"
	case !m.PrismaConfig.IsNull():
		return "prisma"
	case !m.SixwindConfig.IsNull():
		return "6wind"
	case !m.VersaConfig.IsNull():
		return "versa"
	case !m.VmwareConfig.IsNull():
		return "vmware"
	}
	return ""
}

func (orm *mveResourceModel) fromAPIMVE(ctx context.Context, p *megaport.MVE, tags map[string]string) diag.Diagnostics {
	apiDiags := diag.Diagnostics{}
	orm.UID = types.StringValue(p.UID)
	orm.Name = types.StringValue(p.Name)
	orm.CreatedBy = types.StringValue(p.CreatedBy)
	orm.Market = types.StringValue(p.Market)
	orm.LocationID = types.Int64Value(int64(p.LocationID))
	orm.MarketplaceVisibility = types.BoolValue(p.MarketplaceVisibility)
	orm.VXCAutoApproval = types.BoolValue(p.VXCAutoApproval)
	orm.CompanyUID = types.StringValue(p.CompanyUID)
	orm.ContractTermMonths = types.Int64Value(int64(p.ContractTermMonths))
	orm.Vendor = types.StringValue(p.Vendor)
	orm.Size = types.StringValue(p.Size)
	orm.TerminateDate = types.StringValue("")
	orm.CostCentre = types.StringValue(p.CostCentre)
	orm.DiversityZone = types.StringValue(p.DiversityZone)

	if p.TerminateDate != nil {
		orm.TerminateDate = types.StringValue(p.TerminateDate.Format(time.RFC850))
	}

	if p.AttributeTags != nil {
		attrTags, tagDiags := types.MapValueFrom(ctx, types.StringType, p.AttributeTags)
		apiDiags = append(apiDiags, tagDiags...)
		orm.AttributeTags = attrTags
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

// NewMVEResource is a helper function to simplify the provider implementation.
func NewMVEResource() resource.Resource {
	return &mveResource{}
}

// mveResource is the resource implementation.
type mveResource struct {
	client      *megaport.Client
	waitForTime time.Duration
}

// Metadata returns the resource type name.
func (r *mveResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mve"
}

// Schema defines the schema for the resource.
func (r *mveResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	vendorConfigPlanModifiers := []planmodifier.Object{
		objectplanmodifier.UseStateForUnknown(),
		objectplanmodifier.RequiresReplace(),
	}
	vendorConfigValidators := []validator.Object{
		objectvalidator.ExactlyOneOf(mveVendorConfigPaths...),
	}

	resp.Schema = schema.Schema{
		Description: "Megaport Virtual Edge (MVE) Resource for Megaport Terraform provider. This resource allows you to create, modify, and delete Megaport MVEs. Megaport Virtual Edge (MVE) is an on-demand, vendor-neutral Network Function Virtualization (NFV) platform that provides virtual infrastructure for network services at the edge of Megaport's global software-defined network (SDN). Network technologies such as SD-WAN and NGFW are hosted directly on Megaport's global network via Megaport Virtual Edge. Use the `megaport_mve_sizes` data source to query available MVE sizes and the `megaport_mve_images` data source to query available MVE images.",
		Attributes: map[string]schema.Attribute{
			"product_uid": schema.StringAttribute{
				Description: "The unique identifier of the MVE.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"product_name": schema.StringAttribute{
				Description: "The name of the MVE.",
				Required:    true,
			},
			"created_by": schema.StringAttribute{
				Description: "The user who created the MVE.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"terminate_date": schema.StringAttribute{
				Description: "The date the MVE will be or was terminated.",
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
			"contract_term_months": schema.Int64Attribute{
				Description: "The term of the contract in months: valid values are 1, 12, 24, 36, 48, and 60. To set the product to a month-to-month contract with no minimum term, set the value to 1.",
				Required:    true,
				Validators: []validator.Int64{
					int64validator.OneOf(1, 12, 24, 36, 48, 60),
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
			"vxc_auto_approval": schema.BoolAttribute{
				Description: "Whether VXC is auto approved.",
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
			// Per-vendor config blocks.
			"aruba_config": schema.SingleNestedAttribute{
				Description:   "Aruba MVE vendor configuration. Exactly one vendor config block must be set.",
				Optional:      true,
				PlanModifiers: vendorConfigPlanModifiers,
				Validators:    vendorConfigValidators,
				Attributes: map[string]schema.Attribute{
					"image_id":     schema.Int64Attribute{Description: "The image ID for the Aruba MVE.", Required: true},
					"product_size": schema.StringAttribute{Description: "The product size for the Aruba MVE.", Required: true},
					"mve_label":    schema.StringAttribute{Description: "The MVE label.", Optional: true},
					"account_name": schema.StringAttribute{Description: "The Aruba account name. Enter the Account Name from Aruba Orchestrator.", Required: true},
					"account_key":  schema.StringAttribute{Description: "The Aruba account key. Enter the Account Key from Aruba Orchestrator.", Required: true, Sensitive: true},
					"system_tag":   schema.StringAttribute{Description: "The Aruba system tag.", Optional: true},
				},
			},
			"aviatrix_config": schema.SingleNestedAttribute{
				Description:   "Aviatrix MVE vendor configuration. Exactly one vendor config block must be set.",
				Optional:      true,
				PlanModifiers: vendorConfigPlanModifiers,
				Validators:    vendorConfigValidators,
				Attributes: map[string]schema.Attribute{
					"image_id":     schema.Int64Attribute{Description: "The image ID for the Aviatrix MVE.", Required: true},
					"product_size": schema.StringAttribute{Description: "The product size for the Aviatrix MVE.", Required: true},
					"mve_label":    schema.StringAttribute{Description: "The MVE label.", Optional: true},
					"cloud_init":   schema.StringAttribute{Description: "The Base64 encoded cloud init file for Aviatrix.", Required: true, Sensitive: true},
				},
			},
			"cisco_config": schema.SingleNestedAttribute{
				Description:   "Cisco MVE vendor configuration. Exactly one vendor config block must be set.",
				Optional:      true,
				PlanModifiers: vendorConfigPlanModifiers,
				Validators:    vendorConfigValidators,
				Attributes: map[string]schema.Attribute{
					"image_id":             schema.Int64Attribute{Description: "The image ID for the Cisco MVE.", Required: true},
					"product_size":         schema.StringAttribute{Description: "The product size for the Cisco MVE.", Required: true},
					"mve_label":            schema.StringAttribute{Description: "The MVE label.", Optional: true},
					"admin_ssh_public_key": schema.StringAttribute{Description: "The admin SSH public key.", Optional: true, Sensitive: true},
					"ssh_public_key":       schema.StringAttribute{Description: "The SSH public key.", Optional: true, Sensitive: true},
					"manage_locally":       schema.BoolAttribute{Description: "Whether to manage the MVE locally.", Optional: true},
					"cloud_init":           schema.StringAttribute{Description: "The Base64 encoded cloud init file. Required for Cisco C8000v.", Optional: true, Sensitive: true},
					"fmc_ip_address":       schema.StringAttribute{Description: "The FMC IP address. Required for Cisco FTDv.", Optional: true},
					"fmc_registration_key": schema.StringAttribute{Description: "The FMC registration key. Required for Cisco FTDv.", Optional: true, Sensitive: true},
					"fmc_nat_id":           schema.StringAttribute{Description: "The FMC NAT ID. Required for Cisco FTDv.", Optional: true},
				},
			},
			"fortinet_config": schema.SingleNestedAttribute{
				Description:   "Fortinet MVE vendor configuration. Exactly one vendor config block must be set.",
				Optional:      true,
				PlanModifiers: vendorConfigPlanModifiers,
				Validators:    vendorConfigValidators,
				Attributes: map[string]schema.Attribute{
					"image_id":             schema.Int64Attribute{Description: "The image ID for the Fortinet MVE.", Required: true},
					"product_size":         schema.StringAttribute{Description: "The product size for the Fortinet MVE.", Required: true},
					"mve_label":            schema.StringAttribute{Description: "The MVE label.", Optional: true},
					"admin_ssh_public_key": schema.StringAttribute{Description: "The admin SSH public key.", Required: true, Sensitive: true},
					"ssh_public_key":       schema.StringAttribute{Description: "The SSH public key.", Required: true, Sensitive: true},
					"license_data":         schema.StringAttribute{Description: "The license data.", Optional: true, Sensitive: true},
				},
			},
			"meraki_config": schema.SingleNestedAttribute{
				Description:   "Meraki MVE vendor configuration. Exactly one vendor config block must be set.",
				Optional:      true,
				PlanModifiers: vendorConfigPlanModifiers,
				Validators:    vendorConfigValidators,
				Attributes: map[string]schema.Attribute{
					"image_id":     schema.Int64Attribute{Description: "The image ID for the Meraki MVE.", Required: true},
					"product_size": schema.StringAttribute{Description: "The product size for the Meraki MVE.", Required: true},
					"mve_label":    schema.StringAttribute{Description: "The MVE label.", Optional: true},
					"token":        schema.StringAttribute{Description: "The Meraki token.", Required: true, Sensitive: true},
				},
			},
			"palo_alto_config": schema.SingleNestedAttribute{
				Description:   "Palo Alto MVE vendor configuration. Exactly one vendor config block must be set.",
				Optional:      true,
				PlanModifiers: vendorConfigPlanModifiers,
				Validators:    vendorConfigValidators,
				Attributes: map[string]schema.Attribute{
					"image_id":             schema.Int64Attribute{Description: "The image ID for the Palo Alto MVE.", Required: true},
					"product_size":         schema.StringAttribute{Description: "The product size for the Palo Alto MVE.", Required: true},
					"mve_label":            schema.StringAttribute{Description: "The MVE label.", Optional: true},
					"admin_ssh_public_key": schema.StringAttribute{Description: "The admin SSH public key.", Optional: true, Sensitive: true},
					"ssh_public_key":       schema.StringAttribute{Description: "The SSH public key. Must be a 2048-bit RSA key.", Required: true, Sensitive: true},
					"admin_password_hash":  schema.StringAttribute{Description: `The admin password hash. Must be a SHA-256 crypt hash in the format "$5$<salt>$<hash>".`, Required: true, Sensitive: true},
					"license_data":         schema.StringAttribute{Description: "The license data.", Optional: true, Sensitive: true},
				},
			},
			"prisma_config": schema.SingleNestedAttribute{
				Description:   "Prisma MVE vendor configuration. Exactly one vendor config block must be set.",
				Optional:      true,
				PlanModifiers: vendorConfigPlanModifiers,
				Validators:    vendorConfigValidators,
				Attributes: map[string]schema.Attribute{
					"image_id":     schema.Int64Attribute{Description: "The image ID for the Prisma MVE.", Required: true},
					"product_size": schema.StringAttribute{Description: "The product size for the Prisma MVE.", Required: true},
					"mve_label":    schema.StringAttribute{Description: "The MVE label.", Optional: true},
					"ion_key":      schema.StringAttribute{Description: "The vION key.", Required: true, Sensitive: true},
					"secret_key":   schema.StringAttribute{Description: "The secret key.", Required: true, Sensitive: true},
				},
			},
			"sixwind_config": schema.SingleNestedAttribute{
				Description:   "6WIND MVE vendor configuration. Exactly one vendor config block must be set.",
				Optional:      true,
				PlanModifiers: vendorConfigPlanModifiers,
				Validators:    vendorConfigValidators,
				Attributes: map[string]schema.Attribute{
					"image_id":       schema.Int64Attribute{Description: "The image ID for the 6WIND MVE.", Required: true},
					"product_size":   schema.StringAttribute{Description: "The product size for the 6WIND MVE.", Required: true},
					"mve_label":      schema.StringAttribute{Description: "The MVE label.", Optional: true},
					"ssh_public_key": schema.StringAttribute{Description: "The SSH public key. Must be a 2048-bit RSA key.", Required: true, Sensitive: true},
				},
			},
			"versa_config": schema.SingleNestedAttribute{
				Description:   "Versa MVE vendor configuration. Exactly one vendor config block must be set.",
				Optional:      true,
				PlanModifiers: vendorConfigPlanModifiers,
				Validators:    vendorConfigValidators,
				Attributes: map[string]schema.Attribute{
					"image_id":           schema.Int64Attribute{Description: "The image ID for the Versa MVE.", Required: true},
					"product_size":       schema.StringAttribute{Description: "The product size for the Versa MVE.", Required: true},
					"mve_label":          schema.StringAttribute{Description: "The MVE label.", Optional: true},
					"director_address":   schema.StringAttribute{Description: "A FQDN or IPv4 address of your Versa Director.", Required: true},
					"controller_address": schema.StringAttribute{Description: "A FQDN or IPv4 address of your Versa Controller.", Required: true},
					"local_auth":         schema.StringAttribute{Description: "The Local Auth string as configured in your Versa Director.", Required: true, Sensitive: true},
					"remote_auth":        schema.StringAttribute{Description: "The Remote Auth string as configured in your Versa Director.", Required: true, Sensitive: true},
					"serial_number":      schema.StringAttribute{Description: "The serial number specified when creating the device in Versa Director.", Required: true},
				},
			},
			"vmware_config": schema.SingleNestedAttribute{
				Description:   "VMware MVE vendor configuration. Exactly one vendor config block must be set.",
				Optional:      true,
				PlanModifiers: vendorConfigPlanModifiers,
				Validators:    vendorConfigValidators,
				Attributes: map[string]schema.Attribute{
					"image_id":             schema.Int64Attribute{Description: "The image ID for the VMware MVE.", Required: true},
					"product_size":         schema.StringAttribute{Description: "The product size for the VMware MVE.", Required: true},
					"mve_label":            schema.StringAttribute{Description: "The MVE label.", Optional: true},
					"admin_ssh_public_key": schema.StringAttribute{Description: "The admin SSH public key.", Required: true, Sensitive: true},
					"ssh_public_key":       schema.StringAttribute{Description: "The SSH public key. Must be a 2048-bit RSA key.", Required: true, Sensitive: true},
					"vco_address":          schema.StringAttribute{Description: "A FQDN or IPv4/IPv6 address for the Orchestrator.", Required: true},
					"vco_activation_code":  schema.StringAttribute{Description: "The VCO activation code provided by Orchestrator.", Required: true, Sensitive: true},
				},
			},
		},
	}
}

// Create a new resource.
func (r *mveResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
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
		WaitForTime:      r.waitForTime,
	}

	if !plan.ResourceTags.IsNull() {
		tagMap, tagDiags := toResourceTagMap(ctx, plan.ResourceTags)
		resp.Diagnostics.Append(tagDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		mveReq.ResourceTags = tagMap
	}

	vendorConfig, vcDiags := toAPIVendorConfigFromModel(ctx, &plan)
	resp.Diagnostics.Append(vcDiags...)
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
		addAPIError(&resp.Diagnostics, createErrorSummary("MVE", plan.Name.ValueString()), err)
		return
	}

	createdMVE, err := r.client.MVEService.BuyMVE(ctx, mveReq)
	if err != nil {
		addAPIError(&resp.Diagnostics, createErrorSummary("MVE", plan.Name.ValueString()), err)
		return
	}

	createdID := createdMVE.TechnicalServiceUID

	mve, err := r.client.MVEService.GetMVE(ctx, createdID)
	if err != nil {
		addAPIError(&resp.Diagnostics, readErrorSummary("MVE", createdID), err)
		return
	}

	tags, err := r.fetchResourceTags(ctx, createdID)
	if err != nil {
		addAPIError(&resp.Diagnostics, readErrorSummary("MVE Tags", createdID), err)
		return
	}

	apiDiags := plan.fromAPIMVE(ctx, mve, tags)
	resp.Diagnostics = append(resp.Diagnostics, apiDiags...)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read resource information.
func (r *mveResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state mveResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	mve, err := r.client.MVEService.GetMVE(ctx, state.UID.ValueString())
	if err != nil {
		if mpErr, ok := err.(*megaport.ErrorResponse); ok {
			if mpErr.Response.StatusCode == http.StatusNotFound ||
				(mpErr.Response.StatusCode == http.StatusBadRequest && strings.Contains(mpErr.Message, "Could not find a service with UID")) {
				resp.State.RemoveResource(ctx)
				return
			}
		}
		addAPIError(&resp.Diagnostics, readErrorSummary("MVE", state.UID.ValueString()), err)
		return
	}

	if mve.ProvisioningStatus == megaport.STATUS_DECOMMISSIONED {
		resp.State.RemoveResource(ctx)
		return
	}

	tags, err := r.fetchResourceTags(ctx, state.UID.ValueString())
	if err != nil {
		addAPIError(&resp.Diagnostics, readErrorSummary("MVE Tags", state.UID.ValueString()), err)
		return
	}

	// fromAPIMVE does NOT touch vendor config blocks — they remain from state.
	apiDiags := state.fromAPIMVE(ctx, mve, tags)
	resp.Diagnostics = append(resp.Diagnostics, apiDiags...)

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

	// If imported, vendor config blocks will be null — copy from plan.
	if allVendorConfigsNull(state) {
		copyVendorConfigs(&state, &plan)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	var name, costCentre string
	var contractTermMonths *int
	if !plan.Name.Equal(state.Name) {
		name = plan.Name.ValueString()
	} else {
		name = state.Name.ValueString()
	}

	costCentre = plan.CostCentre.ValueString()

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
		WaitForTime:        r.waitForTime,
	})

	if err != nil {
		addAPIError(&resp.Diagnostics, updateErrorSummary("MVE", plan.UID.ValueString()), err)
		return
	}

	updatedMVE, err := r.client.MVEService.GetMVE(ctx, state.UID.ValueString())
	if err != nil {
		addAPIError(&resp.Diagnostics, readErrorSummary("MVE", plan.UID.ValueString()), err)
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
			addAPIError(&resp.Diagnostics, updateErrorSummary("MVE Tags", plan.UID.ValueString()), err)
			return
		}
	}

	tags, err := r.fetchResourceTags(ctx, state.UID.ValueString())
	if err != nil {
		addAPIError(&resp.Diagnostics, readErrorSummary("MVE Tags", plan.UID.ValueString()), err)
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
func (r *mveResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state mveResourceModel
	stateDiags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(stateDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	productUID := state.UID.ValueString()
	_, err := r.client.MVEService.DeleteMVE(ctx, &megaport.DeleteMVERequest{
		MVEID:      productUID,
		SafeDelete: true,
	})
	if err != nil {
		addAPIError(&resp.Diagnostics, deleteErrorSummary("MVE", state.UID.ValueString()), err)
		return
	}

	resp.State.RemoveResource(ctx)
}

// Configure adds the provider configured client to the resource.
func (r *mveResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	providerData, ok := configureMegaportResource(req, resp)
	if !ok {
		return
	}
	r.client = providerData.client
	r.waitForTime = providerData.waitForTime
}

func (r *mveResource) fetchResourceTags(ctx context.Context, id string) (map[string]string, error) {
	return r.client.MVEService.ListMVEResourceTags(ctx, id)
}

func (r *mveResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("product_uid"), req, resp)
}

func (r *mveResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
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
		// If all vendor config blocks are null in state (e.g., after import),
		// compare plan's vendor/size against the API-returned state values and
		// require replacement if they differ.
		if allVendorConfigsNull(state) {
			if allVendorConfigsNull(plan) {
				// Both state and plan have null vendor configs — nothing to do.
				return
			}
			planVendor := vendorNameFromModel(plan)

			if !strings.EqualFold(state.Vendor.ValueString(), planVendor) {
				resp.RequiresReplace = append(resp.RequiresReplace, path.Root("vendor"))
			}

			if !state.Size.IsNull() && !state.Size.IsUnknown() {
				// Extract product_size from the non-null plan config block.
				planSize := planProductSizeFromModel(ctx, plan)
				if planSize != "" && !strings.EqualFold(state.Size.ValueString(), planSize) {
					resp.RequiresReplace = append(resp.RequiresReplace, path.Root("mve_size"))
				}
			}

			copyVendorConfigs(&state, &plan)
		}

		diags := req.State.Set(ctx, &state)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
}

// planProductSizeFromModel extracts the product_size string from whichever
// vendor config block is non-null in the plan model.
func planProductSizeFromModel(ctx context.Context, m mveResourceModel) string {
	switch {
	case !m.ArubaConfig.IsNull():
		var cfg arubaConfigModel
		if m.ArubaConfig.As(ctx, &cfg, basetypes.ObjectAsOptions{}) == nil {
			return cfg.ProductSize.ValueString()
		}
	case !m.AviatrixConfig.IsNull():
		var cfg aviatrixConfigModel
		if m.AviatrixConfig.As(ctx, &cfg, basetypes.ObjectAsOptions{}) == nil {
			return cfg.ProductSize.ValueString()
		}
	case !m.CiscoConfig.IsNull():
		var cfg ciscoConfigModel
		if m.CiscoConfig.As(ctx, &cfg, basetypes.ObjectAsOptions{}) == nil {
			return cfg.ProductSize.ValueString()
		}
	case !m.FortinetConfig.IsNull():
		var cfg fortinetConfigModel
		if m.FortinetConfig.As(ctx, &cfg, basetypes.ObjectAsOptions{}) == nil {
			return cfg.ProductSize.ValueString()
		}
	case !m.MerakiConfig.IsNull():
		var cfg merakiConfigModel
		if m.MerakiConfig.As(ctx, &cfg, basetypes.ObjectAsOptions{}) == nil {
			return cfg.ProductSize.ValueString()
		}
	case !m.PaloAltoConfig.IsNull():
		var cfg paloAltoConfigModel
		if m.PaloAltoConfig.As(ctx, &cfg, basetypes.ObjectAsOptions{}) == nil {
			return cfg.ProductSize.ValueString()
		}
	case !m.PrismaConfig.IsNull():
		var cfg prismaConfigModel
		if m.PrismaConfig.As(ctx, &cfg, basetypes.ObjectAsOptions{}) == nil {
			return cfg.ProductSize.ValueString()
		}
	case !m.SixwindConfig.IsNull():
		var cfg sixwindConfigModel
		if m.SixwindConfig.As(ctx, &cfg, basetypes.ObjectAsOptions{}) == nil {
			return cfg.ProductSize.ValueString()
		}
	case !m.VersaConfig.IsNull():
		var cfg versaConfigModel
		if m.VersaConfig.As(ctx, &cfg, basetypes.ObjectAsOptions{}) == nil {
			return cfg.ProductSize.ValueString()
		}
	case !m.VmwareConfig.IsNull():
		var cfg vmwareConfigModel
		if m.VmwareConfig.As(ctx, &cfg, basetypes.ObjectAsOptions{}) == nil {
			return cfg.ProductSize.ValueString()
		}
	}
	return ""
}

// mveModelFromV1RawState converts a V1 raw JSON state map into a V2 mveResourceModel.
// The V1 schema stored all vendor-specific fields in a single vendor_config object keyed by
// the "vendor" field. V2 uses separate per-vendor blocks.
func mveModelFromV1RawState(ctx context.Context, raw map[string]json.RawMessage) (*mveResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	getString := func(key string) string {
		b, ok := raw[key]
		if !ok {
			return ""
		}
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return ""
		}
		return s
	}

	getInt64 := func(key string) int64 {
		b, ok := raw[key]
		if !ok {
			return 0
		}
		var v int64
		if err := json.Unmarshal(b, &v); err != nil {
			return 0
		}
		return v
	}

	getBool := func(key string) bool {
		b, ok := raw[key]
		if !ok {
			return false
		}
		var v bool
		if err := json.Unmarshal(b, &v); err != nil {
			return false
		}
		return v
	}

	model := &mveResourceModel{
		UID:                   types.StringValue(getString("product_uid")),
		Name:                  types.StringValue(getString("product_name")),
		CreatedBy:             types.StringValue(getString("created_by")),
		TerminateDate:         types.StringValue(getString("terminate_date")),
		Market:                types.StringValue(getString("market")),
		LocationID:            types.Int64Value(getInt64("location_id")),
		MarketplaceVisibility: types.BoolValue(getBool("marketplace_visibility")),
		VXCAutoApproval:       types.BoolValue(getBool("vxc_auto_approval")),
		CompanyUID:            types.StringValue(getString("company_uid")),
		ContractTermMonths:    types.Int64Value(getInt64("contract_term_months")),
		CostCentre:            unmarshalStringAttr(raw, "cost_centre"),
		DiversityZone:         unmarshalStringAttr(raw, "diversity_zone"),
		PromoCode:             unmarshalStringAttr(raw, "promo_code"),
		Vendor:                types.StringValue(getString("vendor")),
		Size:                  types.StringValue(getString("mve_size")),
		// All vendor config blocks default to null; one will be set below.
		ArubaConfig:    types.ObjectNull(arubaConfigAttrs),
		AviatrixConfig: types.ObjectNull(aviatrixConfigAttrs),
		CiscoConfig:    types.ObjectNull(ciscoConfigAttrs),
		FortinetConfig: types.ObjectNull(fortinetConfigAttrs),
		MerakiConfig:   types.ObjectNull(merakiConfigAttrs),
		PaloAltoConfig: types.ObjectNull(paloAltoConfigAttrs),
		PrismaConfig:   types.ObjectNull(prismaConfigAttrs),
		SixwindConfig:  types.ObjectNull(sixwindConfigAttrs),
		VersaConfig:    types.ObjectNull(versaConfigAttrs),
		VmwareConfig:   types.ObjectNull(vmwareConfigAttrs),
		// attribute_tags and resource_tags default to null.
		AttributeTags: types.MapNull(types.StringType),
		ResourceTags:  types.MapNull(types.StringType),
		// vnics default to null list — populated below if present.
		NetworkInterfaces: types.ListNull(types.ObjectType{}.WithAttributeTypes(vnicAttrs)),
	}

	// Migrate attribute_tags if present.
	if b, ok := raw["attribute_tags"]; ok {
		var tagMap map[string]string
		if err := json.Unmarshal(b, &tagMap); err == nil && tagMap != nil {
			attrTags, tagDiags := types.MapValueFrom(ctx, types.StringType, tagMap)
			diags.Append(tagDiags...)
			model.AttributeTags = attrTags
		}
	}

	// Migrate resource_tags if present.
	if b, ok := raw["resource_tags"]; ok {
		var tagMap map[string]string
		if err := json.Unmarshal(b, &tagMap); err == nil && tagMap != nil {
			resourceTags, tagDiags := types.MapValueFrom(ctx, types.StringType, tagMap)
			diags.Append(tagDiags...)
			model.ResourceTags = resourceTags
		}
	}

	// Migrate vnics — V1 had {description, vlan}; V2 has only {description}.
	if b, ok := raw["vnics"]; ok {
		var rawVnics []map[string]json.RawMessage
		if err := json.Unmarshal(b, &rawVnics); err == nil && rawVnics != nil {
			vnics := []types.Object{}
			for _, rv := range rawVnics {
				var desc string
				if db, ok := rv["description"]; ok {
					_ = json.Unmarshal(db, &desc)
				}
				vnicModel := &mveNetworkInterfaceModel{Description: types.StringValue(desc)}
				vnic, vnicDiags := types.ObjectValueFrom(ctx, vnicAttrs, vnicModel)
				diags.Append(vnicDiags...)
				vnics = append(vnics, vnic)
			}
			networkInterfaceList, listDiags := types.ListValueFrom(ctx, types.ObjectType{}.WithAttributeTypes(vnicAttrs), vnics)
			diags.Append(listDiags...)
			model.NetworkInterfaces = networkInterfaceList
		}
	}

	// Migrate the vendor_config object.
	vcRaw, hasVC := raw["vendor_config"]
	if !hasVC {
		// No vendor_config in state — leave all vendor blocks null.
		// The provider will treat this like an import and copy from plan on next apply.
		return model, diags
	}

	// A JSON null vendor_config means the resource was imported — leave all vendor blocks null.
	if string(vcRaw) == "null" {
		return model, diags
	}

	var vcMap map[string]json.RawMessage
	if err := json.Unmarshal(vcRaw, &vcMap); err != nil {
		diags.AddError("Unable to parse V1 vendor_config", err.Error())
		return nil, diags
	}

	vcGetString := func(key string) string {
		b, ok := vcMap[key]
		if !ok {
			return ""
		}
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return ""
		}
		return s
	}

	vcGetImageID := func() int64 {
		b, ok := vcMap["image_id"]
		if !ok {
			return 0
		}
		var v int64
		if err := json.Unmarshal(b, &v); err != nil {
			return 0
		}
		return v
	}

	vcGetBool := func(key string) bool {
		b, ok := vcMap[key]
		if !ok {
			return false
		}
		var v bool
		if err := json.Unmarshal(b, &v); err != nil {
			return false
		}
		return v
	}

	vendor := strings.ToLower(vcGetString("vendor"))

	switch vendor {
	case "aruba":
		cfg := arubaConfigModel{
			ImageID:     types.Int64Value(vcGetImageID()),
			ProductSize: types.StringValue(vcGetString("product_size")),
			MVELabel:    unmarshalStringAttr(vcMap, "mve_label"),
			AccountName: types.StringValue(vcGetString("account_name")),
			AccountKey:  types.StringValue(vcGetString("account_key")),
			SystemTag:   types.StringValue(vcGetString("system_tag")),
		}
		obj, objDiags := types.ObjectValueFrom(ctx, arubaConfigAttrs, cfg)
		diags.Append(objDiags...)
		model.ArubaConfig = obj

	case "aviatrix":
		cfg := aviatrixConfigModel{
			ImageID:     types.Int64Value(vcGetImageID()),
			ProductSize: types.StringValue(vcGetString("product_size")),
			MVELabel:    unmarshalStringAttr(vcMap, "mve_label"),
			CloudInit:   types.StringValue(vcGetString("cloud_init")),
		}
		obj, objDiags := types.ObjectValueFrom(ctx, aviatrixConfigAttrs, cfg)
		diags.Append(objDiags...)
		model.AviatrixConfig = obj

	case "cisco":
		cfg := ciscoConfigModel{
			ImageID:            types.Int64Value(vcGetImageID()),
			ProductSize:        types.StringValue(vcGetString("product_size")),
			MVELabel:           types.StringValue(vcGetString("mve_label")),
			AdminSSHPublicKey:  types.StringValue(vcGetString("admin_ssh_public_key")),
			SSHPublicKey:       types.StringValue(vcGetString("ssh_public_key")),
			ManageLocally:      types.BoolValue(vcGetBool("manage_locally")),
			CloudInit:          types.StringValue(vcGetString("cloud_init")),
			FMCIPAddress:       types.StringValue(vcGetString("fmc_ip_address")),
			FMCRegistrationKey: types.StringValue(vcGetString("fmc_registration_key")),
			FMCNatID:           types.StringValue(vcGetString("fmc_nat_id")),
		}
		obj, objDiags := types.ObjectValueFrom(ctx, ciscoConfigAttrs, cfg)
		diags.Append(objDiags...)
		model.CiscoConfig = obj

	case "fortinet":
		cfg := fortinetConfigModel{
			ImageID:           types.Int64Value(vcGetImageID()),
			ProductSize:       types.StringValue(vcGetString("product_size")),
			MVELabel:          types.StringValue(vcGetString("mve_label")),
			AdminSSHPublicKey: types.StringValue(vcGetString("admin_ssh_public_key")),
			SSHPublicKey:      types.StringValue(vcGetString("ssh_public_key")),
			LicenseData:       types.StringValue(vcGetString("license_data")),
		}
		obj, objDiags := types.ObjectValueFrom(ctx, fortinetConfigAttrs, cfg)
		diags.Append(objDiags...)
		model.FortinetConfig = obj

	case "meraki":
		cfg := merakiConfigModel{
			ImageID:     types.Int64Value(vcGetImageID()),
			ProductSize: types.StringValue(vcGetString("product_size")),
			MVELabel:    unmarshalStringAttr(vcMap, "mve_label"),
			Token:       types.StringValue(vcGetString("token")),
		}
		obj, objDiags := types.ObjectValueFrom(ctx, merakiConfigAttrs, cfg)
		diags.Append(objDiags...)
		model.MerakiConfig = obj

	case "palo_alto":
		cfg := paloAltoConfigModel{
			ImageID:           types.Int64Value(vcGetImageID()),
			ProductSize:       types.StringValue(vcGetString("product_size")),
			MVELabel:          types.StringValue(vcGetString("mve_label")),
			AdminSSHPublicKey: types.StringValue(vcGetString("admin_ssh_public_key")),
			SSHPublicKey:      types.StringValue(vcGetString("ssh_public_key")),
			AdminPasswordHash: types.StringValue(vcGetString("admin_password_hash")),
			LicenseData:       types.StringValue(vcGetString("license_data")),
		}
		obj, objDiags := types.ObjectValueFrom(ctx, paloAltoConfigAttrs, cfg)
		diags.Append(objDiags...)
		model.PaloAltoConfig = obj

	case "prisma":
		cfg := prismaConfigModel{
			ImageID:     types.Int64Value(vcGetImageID()),
			ProductSize: types.StringValue(vcGetString("product_size")),
			MVELabel:    unmarshalStringAttr(vcMap, "mve_label"),
			IONKey:      types.StringValue(vcGetString("ion_key")),
			SecretKey:   types.StringValue(vcGetString("secret_key")),
		}
		obj, objDiags := types.ObjectValueFrom(ctx, prismaConfigAttrs, cfg)
		diags.Append(objDiags...)
		model.PrismaConfig = obj

	case "6wind":
		cfg := sixwindConfigModel{
			ImageID:      types.Int64Value(vcGetImageID()),
			ProductSize:  types.StringValue(vcGetString("product_size")),
			MVELabel:     types.StringValue(vcGetString("mve_label")),
			SSHPublicKey: types.StringValue(vcGetString("ssh_public_key")),
		}
		obj, objDiags := types.ObjectValueFrom(ctx, sixwindConfigAttrs, cfg)
		diags.Append(objDiags...)
		model.SixwindConfig = obj

	case "versa":
		cfg := versaConfigModel{
			ImageID:           types.Int64Value(vcGetImageID()),
			ProductSize:       types.StringValue(vcGetString("product_size")),
			MVELabel:          types.StringValue(vcGetString("mve_label")),
			DirectorAddress:   types.StringValue(vcGetString("director_address")),
			ControllerAddress: types.StringValue(vcGetString("controller_address")),
			LocalAuth:         types.StringValue(vcGetString("local_auth")),
			RemoteAuth:        types.StringValue(vcGetString("remote_auth")),
			SerialNumber:      types.StringValue(vcGetString("serial_number")),
		}
		obj, objDiags := types.ObjectValueFrom(ctx, versaConfigAttrs, cfg)
		diags.Append(objDiags...)
		model.VersaConfig = obj

	case "vmware":
		cfg := vmwareConfigModel{
			ImageID:           types.Int64Value(vcGetImageID()),
			ProductSize:       types.StringValue(vcGetString("product_size")),
			MVELabel:          types.StringValue(vcGetString("mve_label")),
			AdminSSHPublicKey: types.StringValue(vcGetString("admin_ssh_public_key")),
			SSHPublicKey:      types.StringValue(vcGetString("ssh_public_key")),
			VcoAddress:        types.StringValue(vcGetString("vco_address")),
			VcoActivationCode: types.StringValue(vcGetString("vco_activation_code")),
		}
		obj, objDiags := types.ObjectValueFrom(ctx, vmwareConfigAttrs, cfg)
		diags.Append(objDiags...)
		model.VmwareConfig = obj

	default:
		diags.AddError(
			"Unknown vendor in V1 state",
			"Cannot migrate V1 MVE state: unrecognised vendor '"+vendor+"'. Supported vendors: aruba, aviatrix, cisco, fortinet, meraki, palo_alto, prisma, 6wind, versa, vmware.",
		)
		return nil, diags
	}

	return model, diags
}

// MoveState returns state movers that handle migration from V1 to V2 of the MVE resource.
func (r *mveResource) MoveState(ctx context.Context) []resource.StateMover {
	return []resource.StateMover{
		{
			StateMover: func(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
				if req.SourceProviderAddress != "registry.terraform.io/megaport/megaport" || req.SourceTypeName != "megaport_mve" {
					return
				}
				rawJSON := req.SourceRawState.JSON
				if len(rawJSON) == 0 {
					resp.Diagnostics.AddError("Unable to migrate V1 state", "Source raw state JSON is empty")
					return
				}
				var raw map[string]json.RawMessage
				if err := json.Unmarshal(rawJSON, &raw); err != nil {
					resp.Diagnostics.AddError("Unable to unmarshal V1 MVE state", err.Error())
					return
				}
				model, diags := mveModelFromV1RawState(ctx, raw)
				resp.Diagnostics.Append(diags...)
				if resp.Diagnostics.HasError() {
					return
				}
				resp.Diagnostics.Append(resp.TargetState.Set(ctx, model)...)
			},
		},
	}
}
