package provider

import (
	"context"
	"fmt"
	"net/http"
	"strings"

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

	vnicAttrs = map[string]attr.Type{
		"description": types.StringType,
	}
)

// mveResourceModel maps the resource schema data.
type mveResourceModel struct {
	UID                   types.String `tfsdk:"product_uid"`
	Name                  types.String `tfsdk:"product_name"`
	LocationID            types.Int64  `tfsdk:"location_id"`
	MarketplaceVisibility types.Bool   `tfsdk:"marketplace_visibility"`
	CompanyUID            types.String `tfsdk:"company_uid"`
	ContractTermMonths    types.Int64  `tfsdk:"contract_term_months"`
	PromoCode             types.String `tfsdk:"promo_code"`
	CostCentre            types.String `tfsdk:"cost_centre"`
	DiversityZone         types.String `tfsdk:"diversity_zone"`

	Vendor types.String `tfsdk:"vendor"`
	Size   types.String `tfsdk:"mve_size"`

	// Per-vendor config blocks. Exactly one must be set.
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

	ResourceTags types.Map `tfsdk:"resource_tags"`
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
	AdminPassword      types.String `tfsdk:"admin_password"`
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

// mveVendorSpec is the single source of truth for one per-vendor MVE config
// block: its canonical vendor name, its schema attribute name, how to get/set
// it on the model, and how to convert it to the API vendor config type.
// Adding or removing a vendor only requires editing mveVendors below.
type mveVendorSpec struct {
	name     string
	attrName string
	// apiVendor is the value the API reports in the read response's vendor
	// field. It is not always the uppercased block name: 6wind reads back as
	// SIX_WIND, prisma as PALO_ALTO, and vmware as ARISTA. Used to compare an
	// imported MVE's actual vendor against the configured block.
	apiVendor string
	get       func(*mveResourceModel) types.Object
	set       func(*mveResourceModel, types.Object)
	toAPI     func(ctx context.Context, o types.Object, adminPassword string) (megaport.VendorConfig, diag.Diagnostics)
}

// mveVendors is the registry of all per-vendor MVE config blocks.
var mveVendors = []mveVendorSpec{
	{
		name: "aruba", attrName: "aruba_config", apiVendor: "ARUBA",
		get: func(m *mveResourceModel) types.Object { return m.ArubaConfig },
		set: func(m *mveResourceModel, o types.Object) { m.ArubaConfig = o },
		toAPI: func(ctx context.Context, o types.Object, _ string) (megaport.VendorConfig, diag.Diagnostics) {
			var cfg arubaConfigModel
			diags := o.As(ctx, &cfg, basetypes.ObjectAsOptions{})
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
		},
	},
	{
		name: "aviatrix", attrName: "aviatrix_config", apiVendor: "AVIATRIX",
		get: func(m *mveResourceModel) types.Object { return m.AviatrixConfig },
		set: func(m *mveResourceModel, o types.Object) { m.AviatrixConfig = o },
		toAPI: func(ctx context.Context, o types.Object, _ string) (megaport.VendorConfig, diag.Diagnostics) {
			var cfg aviatrixConfigModel
			diags := o.As(ctx, &cfg, basetypes.ObjectAsOptions{})
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
		},
	},
	{
		name: "cisco", attrName: "cisco_config", apiVendor: "CISCO",
		get: func(m *mveResourceModel) types.Object { return m.CiscoConfig },
		set: func(m *mveResourceModel, o types.Object) { m.CiscoConfig = o },
		toAPI: func(ctx context.Context, o types.Object, adminPassword string) (megaport.VendorConfig, diag.Diagnostics) {
			var cfg ciscoConfigModel
			diags := o.As(ctx, &cfg, basetypes.ObjectAsOptions{})
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
				AdminPassword:      adminPassword,
				ManageLocally:      cfg.ManageLocally.ValueBool(),
				CloudInit:          cfg.CloudInit.ValueString(),
				FMCIPAddress:       cfg.FMCIPAddress.ValueString(),
				FMCRegistrationKey: cfg.FMCRegistrationKey.ValueString(),
				FMCNatID:           cfg.FMCNatID.ValueString(),
			}, diags
		},
	},
	{
		name: "fortinet", attrName: "fortinet_config", apiVendor: "FORTINET",
		get: func(m *mveResourceModel) types.Object { return m.FortinetConfig },
		set: func(m *mveResourceModel, o types.Object) { m.FortinetConfig = o },
		toAPI: func(ctx context.Context, o types.Object, _ string) (megaport.VendorConfig, diag.Diagnostics) {
			var cfg fortinetConfigModel
			diags := o.As(ctx, &cfg, basetypes.ObjectAsOptions{})
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
		},
	},
	{
		name: "meraki", attrName: "meraki_config", apiVendor: "MERAKI",
		get: func(m *mveResourceModel) types.Object { return m.MerakiConfig },
		set: func(m *mveResourceModel, o types.Object) { m.MerakiConfig = o },
		toAPI: func(ctx context.Context, o types.Object, _ string) (megaport.VendorConfig, diag.Diagnostics) {
			var cfg merakiConfigModel
			diags := o.As(ctx, &cfg, basetypes.ObjectAsOptions{})
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
		},
	},
	{
		name: "palo_alto", attrName: "palo_alto_config", apiVendor: "PALO_ALTO",
		get: func(m *mveResourceModel) types.Object { return m.PaloAltoConfig },
		set: func(m *mveResourceModel, o types.Object) { m.PaloAltoConfig = o },
		toAPI: func(ctx context.Context, o types.Object, _ string) (megaport.VendorConfig, diag.Diagnostics) {
			var cfg paloAltoConfigModel
			diags := o.As(ctx, &cfg, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}
			return &megaport.PaloAltoConfig{
				Vendor:            "palo_alto",
				ImageID:           int(cfg.ImageID.ValueInt64()),
				ProductSize:       cfg.ProductSize.ValueString(),
				MVELabel:          cfg.MVELabel.ValueString(),
				SSHPublicKey:      cfg.SSHPublicKey.ValueString(),
				AdminPasswordHash: cfg.AdminPasswordHash.ValueString(),
				LicenseData:       cfg.LicenseData.ValueString(),
			}, diags
		},
	},
	{
		name: "prisma", attrName: "prisma_config", apiVendor: "PALO_ALTO",
		get: func(m *mveResourceModel) types.Object { return m.PrismaConfig },
		set: func(m *mveResourceModel, o types.Object) { m.PrismaConfig = o },
		toAPI: func(ctx context.Context, o types.Object, _ string) (megaport.VendorConfig, diag.Diagnostics) {
			var cfg prismaConfigModel
			diags := o.As(ctx, &cfg, basetypes.ObjectAsOptions{})
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
		},
	},
	{
		name: "6wind", attrName: "sixwind_config", apiVendor: "SIX_WIND",
		get: func(m *mveResourceModel) types.Object { return m.SixwindConfig },
		set: func(m *mveResourceModel, o types.Object) { m.SixwindConfig = o },
		toAPI: func(ctx context.Context, o types.Object, _ string) (megaport.VendorConfig, diag.Diagnostics) {
			var cfg sixwindConfigModel
			diags := o.As(ctx, &cfg, basetypes.ObjectAsOptions{})
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
		},
	},
	{
		name: "versa", attrName: "versa_config", apiVendor: "VERSA",
		get: func(m *mveResourceModel) types.Object { return m.VersaConfig },
		set: func(m *mveResourceModel, o types.Object) { m.VersaConfig = o },
		toAPI: func(ctx context.Context, o types.Object, _ string) (megaport.VendorConfig, diag.Diagnostics) {
			var cfg versaConfigModel
			diags := o.As(ctx, &cfg, basetypes.ObjectAsOptions{})
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
		},
	},
	{
		name: "vmware", attrName: "vmware_config", apiVendor: "ARISTA",
		get: func(m *mveResourceModel) types.Object { return m.VmwareConfig },
		set: func(m *mveResourceModel, o types.Object) { m.VmwareConfig = o },
		toAPI: func(ctx context.Context, o types.Object, _ string) (megaport.VendorConfig, diag.Diagnostics) {
			var cfg vmwareConfigModel
			diags := o.As(ctx, &cfg, basetypes.ObjectAsOptions{})
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
		},
	},
}

// specForVendor returns the registry entry for the given canonical vendor
// name (case-insensitive), or false if no such vendor exists.
func specForVendor(name string) (mveVendorSpec, bool) {
	for _, v := range mveVendors {
		if strings.EqualFold(v.name, name) {
			return v, true
		}
	}
	return mveVendorSpec{}, false
}

// mveVendorConfigPaths lists path expressions for all per-vendor config blocks,
// used with ExactlyOneOf validators.
var mveVendorConfigPaths = func() []path.Expression {
	paths := make([]path.Expression, len(mveVendors))
	for i, v := range mveVendors {
		paths[i] = path.MatchRoot(v.attrName)
	}
	return paths
}()

func toAPINetworkInterface(orm *mveNetworkInterfaceModel) *megaport.MVENetworkInterface {
	return &megaport.MVENetworkInterface{
		Description: orm.Description.ValueString(),
	}
}

// vendorObjects returns all vendor config blocks from the model.
func (m *mveResourceModel) vendorObjects() []types.Object {
	objs := make([]types.Object, len(mveVendors))
	for i, v := range mveVendors {
		objs[i] = v.get(m)
	}
	return objs
}

func objectSet(o types.Object) bool { return !o.IsNull() && !o.IsUnknown() }

// allVendorConfigsNull reports whether all 10 vendor config blocks are null or unknown.
func allVendorConfigsNull(m mveResourceModel) bool {
	for _, o := range m.vendorObjects() {
		if objectSet(o) {
			return false
		}
	}
	return true
}

// copyVendorConfigs copies all vendor config blocks from src to dst.
func copyVendorConfigs(dst, src *mveResourceModel) {
	for _, v := range mveVendors {
		v.set(dst, v.get(src))
	}
}

// vendorNameFromModel returns the canonical vendor name based on which config
// block is set in the model, or "" if none is set.
func vendorNameFromModel(m mveResourceModel) string {
	for _, v := range mveVendors {
		if objectSet(v.get(&m)) {
			return v.name
		}
	}
	return ""
}

// vendorConfigPath returns the root path for the config block matching the given
// vendor name. Used for RequiresReplace in ModifyPlan.
func vendorConfigPath(vendorName string) path.Path {
	spec, ok := specForVendor(vendorName)
	if !ok {
		return path.Empty()
	}
	return path.Root(spec.attrName)
}

// blockForVendor returns the config block object for the given vendor name.
func blockForVendor(m mveResourceModel, vendorName string) types.Object {
	spec, ok := specForVendor(vendorName)
	if !ok {
		return types.ObjectNull(nil)
	}
	return spec.get(&m)
}

// vendorBlockEqualIgnoringSizeCase compares two vendor config objects, treating
// product_size case-insensitively. The API normalizes product_size to uppercase,
// so a case-only difference must not trigger a spurious replace.
func vendorBlockEqualIgnoringSizeCase(a, b types.Object) bool {
	if a.IsNull() != b.IsNull() {
		return false
	}
	am := a.Attributes()
	bm := b.Attributes()
	if len(am) != len(bm) {
		return false
	}
	for k, av := range am {
		bv, ok := bm[k]
		if !ok {
			return false
		}
		if k == "product_size" {
			as, aok := av.(types.String)
			bs, bok := bv.(types.String)
			if aok && bok {
				if !strings.EqualFold(as.ValueString(), bs.ValueString()) {
					return false
				}
				continue
			}
		}
		if !av.Equal(bv) {
			return false
		}
	}
	return true
}

// toAPIVendorConfigFromModel converts the set per-vendor config block into a
// megaport.VendorConfig. adminPassword is the write-only Cisco admin password
// re-read from config in Create (it is never persisted to plan or state).
func toAPIVendorConfigFromModel(ctx context.Context, m *mveResourceModel, adminPassword string) (megaport.VendorConfig, diag.Diagnostics) {
	var diags diag.Diagnostics

	count := 0
	for _, o := range m.vendorObjects() {
		if objectSet(o) {
			count++
		}
	}
	if count != 1 {
		diags.AddError(
			"Invalid vendor configuration",
			fmt.Sprintf("Exactly one vendor configuration block must be set, got %d.", count),
		)
		return nil, diags
	}

	spec, ok := specForVendor(vendorNameFromModel(*m))
	if !ok {
		diags.AddError("No vendor config set", "Exactly one vendor config block must be set")
		return nil, diags
	}
	return spec.toAPI(ctx, spec.get(m), adminPassword)
}

// planProductSizeFromModel extracts product_size from whichever vendor config
// block is set in the model.
func planProductSizeFromModel(m mveResourceModel) string {
	vendor := vendorNameFromModel(m)
	obj := blockForVendor(m, vendor)
	if !objectSet(obj) {
		return ""
	}
	if ps, ok := obj.Attributes()["product_size"].(types.String); ok {
		return ps.ValueString()
	}
	return ""
}

func (orm *mveResourceModel) fromAPIMVE(ctx context.Context, p *megaport.MVE, tags map[string]string) diag.Diagnostics {
	apiDiags := diag.Diagnostics{}
	orm.UID = types.StringValue(p.UID)
	orm.Name = types.StringValue(p.Name)
	orm.LocationID = types.Int64Value(int64(p.LocationID))
	orm.MarketplaceVisibility = types.BoolValue(p.MarketplaceVisibility)
	orm.CompanyUID = types.StringValue(p.CompanyUID)
	orm.ContractTermMonths = types.Int64Value(int64(p.ContractTermMonths))
	orm.Vendor = types.StringValue(p.Vendor)
	orm.Size = types.StringValue(p.Size)
	orm.CostCentre = types.StringValue(p.CostCentre)
	orm.DiversityZone = diversityZoneFromAPI(orm.DiversityZone, p.DiversityZone, p.UID, &apiDiags)

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
	client *megaport.Client
}

// Metadata returns the resource type name.
func (r *mveResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mve"
}

// Schema defines the schema for the resource.
func (r *mveResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	vendorConfigPlanModifiers := []planmodifier.Object{
		objectplanmodifier.UseStateForUnknown(),
	}
	vendorConfigValidators := []validator.Object{
		objectvalidator.ExactlyOneOf(mveVendorConfigPaths...),
	}

	resp.Schema = schema.Schema{
		Description: "Megaport Virtual Edge (MVE) Resource for Megaport Terraform provider. This resource allows you to create, modify, and delete Megaport MVEs. Megaport Virtual Edge (MVE) is an on-demand, vendor-neutral Network Function Virtualization (NFV) platform that provides virtual infrastructure for network services at the edge of Megaport’s global software-defined network (SDN). Network technologies such as SD-WAN and NGFW are hosted directly on Megaport’s global network via Megaport Virtual Edge. Use the `megaport_mve_sizes` data source to query available MVE sizes and the `megaport_mve_images` data source to query available MVE images.",
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
			"diversity_zone": schema.StringAttribute{
				Description: "The diversity zone of the MVE. Once known, this value is preserved if a later read reports it empty, since that's typically a transient backend gap rather than a real change. If the empty value is a genuine correction rather than a gap, remove or update `diversity_zone` in your configuration first; optionally run `terraform state rm` followed by `terraform import` to reset the stored value.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
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
			// Per-vendor config blocks. Exactly one must be set. The block that is
			// populated implies the vendor; only image_id and product_size are
			// required, all other fields remain optional because the API is the
			// source of truth for what each vendor image needs.
			"aruba_config": schema.SingleNestedAttribute{
				Description:   "Aruba MVE vendor configuration. Exactly one vendor config block must be set. The MVE is destroyed and re-created if this block changes.",
				Optional:      true,
				PlanModifiers: vendorConfigPlanModifiers,
				Validators:    vendorConfigValidators,
				Attributes: map[string]schema.Attribute{
					"image_id":     schema.Int64Attribute{Description: "The image ID of the MVE. Indicates the software version.", Required: true},
					"product_size": schema.StringAttribute{Description: "The product size for the vendor config.", Required: true},
					"mve_label":    schema.StringAttribute{Description: "The MVE label for the vendor config.", Optional: true},
					"account_name": schema.StringAttribute{Description: "The account name for the vendor config. Enter the Account Name from Aruba Orchestrator.", Optional: true},
					"account_key":  schema.StringAttribute{Description: "The account key for the vendor config. Enter the Account Key from Aruba Orchestrator.", Optional: true, Sensitive: true},
					"system_tag":   schema.StringAttribute{Description: "The system tag for the vendor config. Aruba Orchestrator System Tags register the EC-V with the Cloud Portal and Orchestrator.", Optional: true},
				},
			},
			"aviatrix_config": schema.SingleNestedAttribute{
				Description:   "Aviatrix MVE vendor configuration. Exactly one vendor config block must be set. The MVE is destroyed and re-created if this block changes.",
				Optional:      true,
				PlanModifiers: vendorConfigPlanModifiers,
				Attributes: map[string]schema.Attribute{
					"image_id":     schema.Int64Attribute{Description: "The image ID of the MVE. Indicates the software version.", Required: true},
					"product_size": schema.StringAttribute{Description: "The product size for the vendor config.", Required: true},
					"mve_label":    schema.StringAttribute{Description: "The MVE label for the vendor config.", Optional: true},
					"cloud_init":   schema.StringAttribute{Description: "The Base64 encoded cloud init file for the vendor config. The bootstrap configuration file. Required for Aviatrix.", Optional: true},
				},
			},
			"cisco_config": schema.SingleNestedAttribute{
				Description:   "Cisco MVE vendor configuration. Exactly one vendor config block must be set. The MVE is destroyed and re-created if this block changes.",
				Optional:      true,
				PlanModifiers: vendorConfigPlanModifiers,
				Attributes: map[string]schema.Attribute{
					"image_id":             schema.Int64Attribute{Description: "The image ID of the MVE. Indicates the software version.", Required: true},
					"product_size":         schema.StringAttribute{Description: "The product size for the vendor config.", Required: true},
					"mve_label":            schema.StringAttribute{Description: "The MVE label for the vendor config.", Optional: true},
					"admin_ssh_public_key": schema.StringAttribute{Description: "The admin SSH public key for the vendor config.", Optional: true},
					"ssh_public_key":       schema.StringAttribute{Description: "The SSH public key for the vendor config.", Optional: true},
					"admin_password":       schema.StringAttribute{Description: "Plain-text admin password for the vendor config. Required for Cisco FTDv (Firewall) MVE only. Must be 9–100 characters and may not contain `\"`, carriage return, or line feed. This value is only consumed when the MVE is provisioned to seed the initial admin account; after deployment, manage the password via the vendor's management interface. Declared as a [write-only argument](https://developer.hashicorp.com/terraform/language/v1.11.x/resources/ephemeral/write-only) (Terraform 1.11+) so the password is not persisted in the Terraform plan or state.", Optional: true, Sensitive: true, WriteOnly: true},
					"manage_locally":       schema.BoolAttribute{Description: "Whether to manage the MVE locally. Required for Cisco MVE.", Optional: true},
					"cloud_init":           schema.StringAttribute{Description: "The Base64 encoded cloud init file for the vendor config. Required for Cisco C8000v.", Optional: true},
					"fmc_ip_address":       schema.StringAttribute{Description: "The FMC IP address for the vendor config. Required for Cisco FTDv (Firewall) MVE.", Optional: true},
					"fmc_registration_key": schema.StringAttribute{Description: "The FMC registration key for the vendor config. Required for Cisco FTDv (Firewall) MVE.", Optional: true, Sensitive: true},
					"fmc_nat_id":           schema.StringAttribute{Description: "The FMC NAT ID for the vendor config. Required for Cisco FTDv (Firewall) MVE.", Optional: true},
				},
			},
			"fortinet_config": schema.SingleNestedAttribute{
				Description:   "Fortinet MVE vendor configuration. Exactly one vendor config block must be set. The MVE is destroyed and re-created if this block changes.",
				Optional:      true,
				PlanModifiers: vendorConfigPlanModifiers,
				Attributes: map[string]schema.Attribute{
					"image_id":             schema.Int64Attribute{Description: "The image ID of the MVE. Indicates the software version.", Required: true},
					"product_size":         schema.StringAttribute{Description: "The product size for the vendor config.", Required: true},
					"mve_label":            schema.StringAttribute{Description: "The MVE label for the vendor config.", Optional: true},
					"admin_ssh_public_key": schema.StringAttribute{Description: "The admin SSH public key for the vendor config.", Optional: true},
					"ssh_public_key":       schema.StringAttribute{Description: "The SSH public key for the vendor config. Must be a 2048-bit RSA key.", Optional: true},
					"license_data":         schema.StringAttribute{Description: "The license data for the vendor config. Required for Fortinet.", Optional: true, Sensitive: true},
				},
			},
			"meraki_config": schema.SingleNestedAttribute{
				Description:   "Meraki MVE vendor configuration. Exactly one vendor config block must be set. The MVE is destroyed and re-created if this block changes.",
				Optional:      true,
				PlanModifiers: vendorConfigPlanModifiers,
				Attributes: map[string]schema.Attribute{
					"image_id":     schema.Int64Attribute{Description: "The image ID of the MVE. Indicates the software version.", Required: true},
					"product_size": schema.StringAttribute{Description: "The product size for the vendor config.", Required: true},
					"mve_label":    schema.StringAttribute{Description: "The MVE label for the vendor config.", Optional: true},
					"token":        schema.StringAttribute{Description: "The token for the vendor config. Required for Meraki.", Optional: true, Sensitive: true},
				},
			},
			"palo_alto_config": schema.SingleNestedAttribute{
				Description:   "Palo Alto MVE vendor configuration. Exactly one vendor config block must be set. The MVE is destroyed and re-created if this block changes.",
				Optional:      true,
				PlanModifiers: vendorConfigPlanModifiers,
				Attributes: map[string]schema.Attribute{
					"image_id":            schema.Int64Attribute{Description: "The image ID of the MVE. Indicates the software version.", Required: true},
					"product_size":        schema.StringAttribute{Description: "The product size for the vendor config.", Required: true},
					"mve_label":           schema.StringAttribute{Description: "The MVE label for the vendor config.", Optional: true},
					"ssh_public_key":      schema.StringAttribute{Description: "The SSH public key for the vendor config. Must be a 2048-bit RSA key.", Optional: true},
					"admin_password_hash": schema.StringAttribute{Description: "The sha256crypt-formatted admin password hash for the vendor config. Required for Palo Alto VM-Series MVE. Must match the format `$5$<salt>$<hash>`.", Optional: true, Sensitive: true},
					"license_data":        schema.StringAttribute{Description: "The license data for the vendor config. Required for Palo Alto.", Optional: true, Sensitive: true},
				},
			},
			"prisma_config": schema.SingleNestedAttribute{
				Description:   "Prisma MVE vendor configuration. Exactly one vendor config block must be set. The MVE is destroyed and re-created if this block changes.",
				Optional:      true,
				PlanModifiers: vendorConfigPlanModifiers,
				Attributes: map[string]schema.Attribute{
					"image_id":     schema.Int64Attribute{Description: "The image ID of the MVE. Indicates the software version.", Required: true},
					"product_size": schema.StringAttribute{Description: "The product size for the vendor config.", Required: true},
					"mve_label":    schema.StringAttribute{Description: "The MVE label for the vendor config.", Optional: true},
					"ion_key":      schema.StringAttribute{Description: "The vION key for the vendor config. Required for Prisma.", Optional: true, Sensitive: true},
					"secret_key":   schema.StringAttribute{Description: "The secret key for the vendor config. Required for Prisma.", Optional: true, Sensitive: true},
				},
			},
			"sixwind_config": schema.SingleNestedAttribute{
				Description:   "6WIND MVE vendor configuration. Exactly one vendor config block must be set. The MVE is destroyed and re-created if this block changes.",
				Optional:      true,
				PlanModifiers: vendorConfigPlanModifiers,
				Attributes: map[string]schema.Attribute{
					"image_id":       schema.Int64Attribute{Description: "The image ID of the MVE. Indicates the software version.", Required: true},
					"product_size":   schema.StringAttribute{Description: "The product size for the vendor config.", Required: true},
					"mve_label":      schema.StringAttribute{Description: "The MVE label for the vendor config.", Optional: true},
					"ssh_public_key": schema.StringAttribute{Description: "The SSH public key for the vendor config. Must be a 2048-bit RSA key.", Optional: true},
				},
			},
			"versa_config": schema.SingleNestedAttribute{
				Description:   "Versa MVE vendor configuration. Exactly one vendor config block must be set. The MVE is destroyed and re-created if this block changes.",
				Optional:      true,
				PlanModifiers: vendorConfigPlanModifiers,
				Attributes: map[string]schema.Attribute{
					"image_id":           schema.Int64Attribute{Description: "The image ID of the MVE. Indicates the software version.", Required: true},
					"product_size":       schema.StringAttribute{Description: "The product size for the vendor config.", Required: true},
					"mve_label":          schema.StringAttribute{Description: "The MVE label for the vendor config.", Optional: true},
					"director_address":   schema.StringAttribute{Description: "A FQDN or IPv4 address of your Versa Director. Required for Versa.", Optional: true},
					"controller_address": schema.StringAttribute{Description: "A FQDN or IPv4 address of your Versa Controller. Required for Versa.", Optional: true},
					"local_auth":         schema.StringAttribute{Description: "The local auth string as configured in your Versa Director. Required for Versa.", Optional: true, Sensitive: true},
					"remote_auth":        schema.StringAttribute{Description: "The remote auth string as configured in your Versa Director. Required for Versa.", Optional: true, Sensitive: true},
					"serial_number":      schema.StringAttribute{Description: "The serial number specified when creating the device in Versa Director. Required for Versa.", Optional: true},
				},
			},
			"vmware_config": schema.SingleNestedAttribute{
				Description:   "VMware MVE vendor configuration. Exactly one vendor config block must be set. The MVE is destroyed and re-created if this block changes.",
				Optional:      true,
				PlanModifiers: vendorConfigPlanModifiers,
				Attributes: map[string]schema.Attribute{
					"image_id":             schema.Int64Attribute{Description: "The image ID of the MVE. Indicates the software version.", Required: true},
					"product_size":         schema.StringAttribute{Description: "The product size for the vendor config.", Required: true},
					"mve_label":            schema.StringAttribute{Description: "The MVE label for the vendor config.", Optional: true},
					"admin_ssh_public_key": schema.StringAttribute{Description: "The admin SSH public key for the vendor config.", Optional: true},
					"ssh_public_key":       schema.StringAttribute{Description: "The SSH public key for the vendor config. Must be a 2048-bit RSA key.", Optional: true},
					"vco_address":          schema.StringAttribute{Description: "A FQDN or IPv4/IPv6 address for the Orchestrator where you created the edge device. Required for VMware.", Optional: true},
					"vco_activation_code":  schema.StringAttribute{Description: "The VCO activation code provided by Orchestrator after creating the edge device. Required for VMware.", Optional: true, Sensitive: true},
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

	// admin_password is a write-only attribute (Cisco FTDv), so it is null in
	// req.Plan. Re-read it from req.Config and pass it through to the API request.
	var configAdminPassword types.String
	if objectSet(plan.CiscoConfig) {
		resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("cisco_config").AtName("admin_password"), &configAdminPassword)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	vendorConfig, apiVCDiags := toAPIVendorConfigFromModel(ctx, &plan, configAdminPassword.ValueString())
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

	tags, err := r.client.MVEService.ListMVEResourceTags(ctx, createdID)
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
		// MVE has been deleted or is not found
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

	// If the MVE has been deleted
	if mve.ProvisioningStatus == megaport.STATUS_DECOMMISSIONED {
		resp.State.RemoveResource(ctx)
		return
	}

	// Get tags
	tags, err := r.client.MVEService.ListMVEResourceTags(ctx, state.UID.ValueString())
	if err != nil {
		addAPIError(&resp.Diagnostics, readErrorSummary("MVE Tags", state.UID.ValueString()), err)
		return
	}

	// fromAPIMVE does not touch the vendor config blocks; they remain from state.
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

	// Preserve the plan's vendor config blocks in state. This handles two cases:
	// 1. After Import, the vendor config blocks in state are null — adopt the plan value.
	// 2. Case-only changes (e.g., "small" → "SMALL") — adopt the plan's casing so
	//    Terraform doesn't see an inconsistent result after apply.
	copyVendorConfigs(&state, &plan)

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

	// Always use the planned cost centre value, even if it's empty/null
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
		WaitForTime:        waitForTime,
	})

	if err != nil {
		addAPIError(&resp.Diagnostics, updateErrorSummary("MVE", state.UID.ValueString()), err)
		return
	}

	updatedMVE, err := r.client.MVEService.GetMVE(ctx, state.UID.ValueString())
	if err != nil {
		addAPIError(&resp.Diagnostics, readErrorSummary("MVE", state.UID.ValueString()), err)
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
			addAPIError(&resp.Diagnostics, updateErrorSummary("MVE Tags", state.UID.ValueString()), err)
			return
		}
	}

	tags, err := r.client.MVEService.ListMVEResourceTags(ctx, state.UID.ValueString())
	if err != nil {
		addAPIError(&resp.Diagnostics, readErrorSummary("MVE Tags", state.UID.ValueString()), err)
		return
	}

	apiDiags := state.fromAPIMVE(ctx, updatedMVE, tags)
	resp.Diagnostics = append(resp.Diagnostics, apiDiags...)

	state.PromoCode = plan.PromoCode

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
	err := retryTransientDelete(ctx, 3, func() error {
		_, deleteErr := r.client.MVEService.DeleteMVE(ctx, &megaport.DeleteMVERequest{
			MVEID:      productUID,
			SafeDelete: true,
		})
		return deleteErr
	})
	if err != nil {
		addAPIError(&resp.Diagnostics, deleteErrorSummary("MVE", state.UID.ValueString()), err)
		return
	}

	resp.State.RemoveResource(ctx)
}

// Configure adds the provider configured client to the resource.
func (r *mveResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	data, ok := configureMegaportResource(req, resp)
	if !ok {
		return
	}
	r.client = data.client
}

func (r *mveResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("product_uid"), req, resp)
}

func (r *mveResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		// Destroy operation — nothing to do.
		return
	}

	var plan, state mveResourceModel
	planDiags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(planDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !req.State.Raw.IsNull() {
		stateDiags := req.State.Get(ctx, &state)
		resp.Diagnostics.Append(stateDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Only evaluate immutability on an existing resource.
	if state.UID.IsNull() {
		return
	}

	planVendor := vendorNameFromModel(plan)
	cfgPath := vendorConfigPath(planVendor)

	if allVendorConfigsNull(state) {
		// Import scenario: vendor config blocks are null in state. Compare the
		// plan's vendor/size against the API-derived state values (which are
		// uppercase), requiring replacement only when they differ. The vendor is
		// mapped to the value the API reports (e.g. 6wind -> SIX_WIND), and size
		// is compared case-insensitively because the API normalizes to uppercase.
		if allVendorConfigsNull(plan) {
			return
		}

		if spec, ok := specForVendor(planVendor); ok && spec.apiVendor != "" &&
			!state.Vendor.IsNull() && !state.Vendor.IsUnknown() &&
			!strings.EqualFold(state.Vendor.ValueString(), spec.apiVendor) {
			resp.RequiresReplace = append(resp.RequiresReplace, cfgPath)
		}

		if !state.Size.IsNull() && !state.Size.IsUnknown() {
			planSize := planProductSizeFromModel(plan)
			if planSize != "" && !strings.EqualFold(state.Size.ValueString(), planSize) {
				resp.RequiresReplace = append(resp.RequiresReplace, cfgPath)
			}
		}
		return
	}

	// Existing resource with a vendor config block in state. The vendor config is
	// immutable, so any change requires replacement. A change of vendor (a
	// different block being set) always requires replacement; otherwise compare
	// the block contents, treating product_size case-insensitively.
	stateVendor := vendorNameFromModel(state)
	if !strings.EqualFold(stateVendor, planVendor) {
		resp.RequiresReplace = append(resp.RequiresReplace, cfgPath)
		return
	}

	planObj := blockForVendor(plan, planVendor)
	stateObj := blockForVendor(state, stateVendor)
	if planObj.IsNull() || planObj.IsUnknown() {
		return
	}
	if !vendorBlockEqualIgnoringSizeCase(planObj, stateObj) {
		resp.RequiresReplace = append(resp.RequiresReplace, cfgPath)
	}
}
