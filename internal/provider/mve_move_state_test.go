package provider

import (
	"context"
	"encoding/json"
	"sort"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mveV2Schema() schema.Schema {
	r := &mveResource{}
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)
	return resp.Schema
}

func newMVETargetState(t *testing.T) tfsdk.State {
	t.Helper()
	s := mveV2Schema()
	return tfsdk.State{
		Schema: s,
		Raw:    tftypes.NewValue(s.Type().TerraformType(context.Background()), nil),
	}
}

func invokeMVEStateMover(t *testing.T, providerAddr, typeName string, rawJSON []byte) (*resource.MoveStateResponse, *mveResourceModel) {
	t.Helper()
	ctx := context.Background()
	r := &mveResource{}
	movers := r.MoveState(ctx)
	require.Len(t, movers, 1, "expected exactly 1 state mover")

	req := resource.MoveStateRequest{
		SourceProviderAddress: providerAddr,
		SourceTypeName:        typeName,
		SourceRawState: &tfprotov6.RawState{
			JSON: rawJSON,
		},
	}

	resp := &resource.MoveStateResponse{
		TargetState: newMVETargetState(t),
	}

	movers[0].StateMover(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		return resp, nil
	}

	if resp.TargetState.Raw.IsNull() {
		return resp, nil
	}

	var model mveResourceModel
	diags := resp.TargetState.Get(ctx, &model)
	if diags.HasError() {
		t.Fatalf("failed to get target state: %v", diags)
	}
	return resp, &model
}

// baseV1MVEState returns a minimal V1 MVE state with common fields set.
func baseV1MVEState() map[string]interface{} {
	return map[string]interface{}{
		// Fields kept in V2
		"product_uid":            "mve-uid-abc123",
		"product_name":           "My MVE",
		"location_id":            float64(65),
		"contract_term_months":   float64(12),
		"cost_centre":            "CC-99",
		"diversity_zone":         "red",
		"promo_code":             "PROMO2025",
		"vendor":                 "aruba",
		"mve_size":               "MEDIUM",
		"company_uid":            "company-uid-xyz",
		"marketplace_visibility": true,
		"vxc_auto_approval":      false,
		"market":                 "AU",
		"created_by":             "user@example.com",
		"terminate_date":         nil,
		"attribute_tags": map[string]interface{}{
			"account": "prod",
		},
		"resource_tags": map[string]interface{}{
			"env": "staging",
		},
		// Fields removed in V2
		"last_updated":        "2024-01-01T00:00:00Z",
		"product_id":          float64(12345),
		"product_type":        "MVE",
		"provisioning_status": "LIVE",
		"usage_algorithm":     "POST_PAID_FIXED",
		"company_name":        "Test Company",
		"locked":              false,
		"admin_locked":        false,
		"cancelable":          true,
		"live_date":           "2024-01-01T00:00:00Z",
		"create_date":         "2024-01-01T00:00:00Z",
		"contract_start_date": "2024-01-01T00:00:00Z",
		"contract_end_date":   "2025-01-01T00:00:00Z",
		"vxc_permitted":       true,
		"buyout_port":         false,
		"virtual":             false,
		"secondary_name":      "",
	}
}

func TestMoveState_MVE_V1ToV2_Aruba(t *testing.T) {
	v1 := baseV1MVEState()
	v1["vendor"] = "aruba"
	v1["vendor_config"] = map[string]interface{}{
		"vendor":       "aruba",
		"image_id":     float64(123),
		"product_size": "MEDIUM",
		"mve_label":    "test-label",
		"account_name": "acme-corp",
		"account_key":  "supersecret",
		"system_tag":   "tag1",
	}
	v1["vnics"] = []interface{}{
		map[string]interface{}{"description": "Data Plane", "vlan": float64(100)},
		map[string]interface{}{"description": "Control Plane", "vlan": float64(200)},
	}

	rawJSON, err := json.Marshal(v1)
	require.NoError(t, err)

	resp, model := invokeMVEStateMover(t, "registry.terraform.io/megaport/megaport", "megaport_mve", rawJSON)
	require.False(t, resp.Diagnostics.HasError(), "unexpected diagnostics: %v", resp.Diagnostics)
	require.NotNil(t, model)

	// Verify scalar fields
	assert.Equal(t, "mve-uid-abc123", model.UID.ValueString())
	assert.Equal(t, "My MVE", model.Name.ValueString())
	assert.Equal(t, int64(65), model.LocationID.ValueInt64())
	assert.Equal(t, int64(12), model.ContractTermMonths.ValueInt64())
	assert.Equal(t, "CC-99", model.CostCentre.ValueString())
	assert.Equal(t, "red", model.DiversityZone.ValueString())
	assert.Equal(t, "PROMO2025", model.PromoCode.ValueString())
	assert.Equal(t, "aruba", model.Vendor.ValueString())
	assert.Equal(t, "MEDIUM", model.Size.ValueString())
	assert.Equal(t, "company-uid-xyz", model.CompanyUID.ValueString())
	assert.True(t, model.MarketplaceVisibility.ValueBool())
	assert.False(t, model.VXCAutoApproval.ValueBool())
	assert.Equal(t, "AU", model.Market.ValueString())
	assert.Equal(t, "user@example.com", model.CreatedBy.ValueString())

	// aruba_config populated
	assert.False(t, model.ArubaConfig.IsNull(), "aruba_config should be populated")
	var aruba arubaConfigModel
	diags := model.ArubaConfig.As(context.Background(), &aruba, basetypes.ObjectAsOptions{})
	require.False(t, diags.HasError())
	assert.Equal(t, int64(123), aruba.ImageID.ValueInt64())
	assert.Equal(t, "MEDIUM", aruba.ProductSize.ValueString())
	assert.Equal(t, "test-label", aruba.MVELabel.ValueString())
	assert.Equal(t, "acme-corp", aruba.AccountName.ValueString())
	assert.Equal(t, "supersecret", aruba.AccountKey.ValueString())
	assert.Equal(t, "tag1", aruba.SystemTag.ValueString())

	// All other vendor config blocks null
	assert.True(t, model.AviatrixConfig.IsNull(), "aviatrix_config should be null")
	assert.True(t, model.CiscoConfig.IsNull(), "cisco_config should be null")
	assert.True(t, model.FortinetConfig.IsNull(), "fortinet_config should be null")
	assert.True(t, model.MerakiConfig.IsNull(), "meraki_config should be null")
	assert.True(t, model.PaloAltoConfig.IsNull(), "palo_alto_config should be null")
	assert.True(t, model.PrismaConfig.IsNull(), "prisma_config should be null")
	assert.True(t, model.SixwindConfig.IsNull(), "sixwind_config should be null")
	assert.True(t, model.VersaConfig.IsNull(), "versa_config should be null")
	assert.True(t, model.VmwareConfig.IsNull(), "vmware_config should be null")

	// vnics: 2 elements, no vlan
	assert.False(t, model.NetworkInterfaces.IsNull())
	vnics := model.NetworkInterfaces.Elements()
	assert.Len(t, vnics, 2)
	// vlan should NOT be present as a key in the object
	vnicObj, ok := vnics[0].(types.Object)
	require.True(t, ok, "expected vNIC element to be types.Object")
	assert.NotContains(t, vnicObj.AttributeTypes(context.Background()), "vlan")
	vnicAttrsMap := vnicObj.Attributes()
	desc, ok := vnicAttrsMap["description"].(types.String)
	require.True(t, ok)
	assert.Equal(t, "Data Plane", desc.ValueString())
}

func TestMoveState_MVE_V1ToV2_Cisco(t *testing.T) {
	v1 := baseV1MVEState()
	v1["vendor"] = "cisco"
	v1["vendor_config"] = map[string]interface{}{
		"vendor":               "cisco",
		"image_id":             float64(456),
		"product_size":         "LARGE",
		"mve_label":            "cisco-mve",
		"admin_ssh_public_key": "ssh-rsa AAAA...",
		"ssh_public_key":       "ssh-rsa BBBB...",
		"manage_locally":       true,
		"cloud_init":           "",
		"fmc_ip_address":       "10.0.0.1",
		"fmc_registration_key": "regkey123",
		"fmc_nat_id":           "natid456",
	}
	v1["vnics"] = []interface{}{
		map[string]interface{}{"description": "eth0", "vlan": float64(0)},
	}

	rawJSON, err := json.Marshal(v1)
	require.NoError(t, err)

	resp, model := invokeMVEStateMover(t, "registry.terraform.io/megaport/megaport", "megaport_mve", rawJSON)
	require.False(t, resp.Diagnostics.HasError(), "unexpected diagnostics: %v", resp.Diagnostics)
	require.NotNil(t, model)

	assert.False(t, model.CiscoConfig.IsNull(), "cisco_config should be populated")
	assert.True(t, model.ArubaConfig.IsNull())

	var cisco ciscoConfigModel
	diags := model.CiscoConfig.As(context.Background(), &cisco, basetypes.ObjectAsOptions{})
	require.False(t, diags.HasError())
	assert.Equal(t, int64(456), cisco.ImageID.ValueInt64())
	assert.Equal(t, "LARGE", cisco.ProductSize.ValueString())
	assert.Equal(t, "cisco-mve", cisco.MVELabel.ValueString())
	assert.Equal(t, "ssh-rsa AAAA...", cisco.AdminSSHPublicKey.ValueString())
	assert.Equal(t, "ssh-rsa BBBB...", cisco.SSHPublicKey.ValueString())
	assert.True(t, cisco.ManageLocally.ValueBool())
	assert.Equal(t, "10.0.0.1", cisco.FMCIPAddress.ValueString())
	assert.Equal(t, "regkey123", cisco.FMCRegistrationKey.ValueString())
	assert.Equal(t, "natid456", cisco.FMCNatID.ValueString())
}

func TestMoveState_MVE_V1ToV2_Versa(t *testing.T) {
	v1 := baseV1MVEState()
	v1["vendor"] = "versa"
	v1["vendor_config"] = map[string]interface{}{
		"vendor":             "versa",
		"image_id":           float64(789),
		"product_size":       "MEDIUM",
		"mve_label":          "",
		"director_address":   "192.168.1.1",
		"controller_address": "192.168.1.2",
		"local_auth":         "localpass",
		"remote_auth":        "remotepass",
		"serial_number":      "SN-12345",
	}
	v1["vnics"] = []interface{}{
		map[string]interface{}{"description": "WAN", "vlan": float64(10)},
	}

	rawJSON, err := json.Marshal(v1)
	require.NoError(t, err)

	resp, model := invokeMVEStateMover(t, "registry.terraform.io/megaport/megaport", "megaport_mve", rawJSON)
	require.False(t, resp.Diagnostics.HasError(), "unexpected diagnostics: %v", resp.Diagnostics)
	require.NotNil(t, model)

	assert.False(t, model.VersaConfig.IsNull(), "versa_config should be populated")
	assert.True(t, model.ArubaConfig.IsNull())

	var versa versaConfigModel
	diags := model.VersaConfig.As(context.Background(), &versa, basetypes.ObjectAsOptions{})
	require.False(t, diags.HasError())
	assert.Equal(t, int64(789), versa.ImageID.ValueInt64())
	assert.Equal(t, "192.168.1.1", versa.DirectorAddress.ValueString())
	assert.Equal(t, "192.168.1.2", versa.ControllerAddress.ValueString())
	assert.Equal(t, "localpass", versa.LocalAuth.ValueString())
	assert.Equal(t, "remotepass", versa.RemoteAuth.ValueString())
	assert.Equal(t, "SN-12345", versa.SerialNumber.ValueString())
}

func TestMoveState_MVE_V1ToV2_VMware(t *testing.T) {
	v1 := baseV1MVEState()
	v1["vendor"] = "vmware"
	v1["vendor_config"] = map[string]interface{}{
		"vendor":               "vmware",
		"image_id":             float64(321),
		"product_size":         "SMALL",
		"mve_label":            "vmw-label",
		"admin_ssh_public_key": "ssh-rsa CCCC...",
		"ssh_public_key":       "ssh-rsa DDDD...",
		"vco_address":          "vco.example.com",
		"vco_activation_code":  "ACT-KEY-XYZ",
	}
	v1["vnics"] = []interface{}{
		map[string]interface{}{"description": "Public", "vlan": float64(50)},
		map[string]interface{}{"description": "Private", "vlan": float64(51)},
	}

	rawJSON, err := json.Marshal(v1)
	require.NoError(t, err)

	resp, model := invokeMVEStateMover(t, "registry.terraform.io/megaport/megaport", "megaport_mve", rawJSON)
	require.False(t, resp.Diagnostics.HasError(), "unexpected diagnostics: %v", resp.Diagnostics)
	require.NotNil(t, model)

	assert.False(t, model.VmwareConfig.IsNull(), "vmware_config should be populated")
	assert.True(t, model.ArubaConfig.IsNull())

	var vmw vmwareConfigModel
	diags := model.VmwareConfig.As(context.Background(), &vmw, basetypes.ObjectAsOptions{})
	require.False(t, diags.HasError())
	assert.Equal(t, int64(321), vmw.ImageID.ValueInt64())
	assert.Equal(t, "ssh-rsa CCCC...", vmw.AdminSSHPublicKey.ValueString())
	assert.Equal(t, "ssh-rsa DDDD...", vmw.SSHPublicKey.ValueString())
	assert.Equal(t, "vco.example.com", vmw.VcoAddress.ValueString())
	assert.Equal(t, "ACT-KEY-XYZ", vmw.VcoActivationCode.ValueString())
}

func TestMoveState_MVE_V1ToV2_NoVendorConfig(t *testing.T) {
	v1 := baseV1MVEState()
	v1["vendor_config"] = nil
	v1["vnics"] = []interface{}{
		map[string]interface{}{"description": "eth0", "vlan": float64(0)},
	}

	rawJSON, err := json.Marshal(v1)
	require.NoError(t, err)

	resp, model := invokeMVEStateMover(t, "registry.terraform.io/megaport/megaport", "megaport_mve", rawJSON)
	require.False(t, resp.Diagnostics.HasError(), "unexpected diagnostics: %v", resp.Diagnostics)
	require.NotNil(t, model)

	// All vendor config blocks should be null (imported resource scenario)
	assert.True(t, model.ArubaConfig.IsNull(), "aruba_config should be null")
	assert.True(t, model.AviatrixConfig.IsNull(), "aviatrix_config should be null")
	assert.True(t, model.CiscoConfig.IsNull(), "cisco_config should be null")
	assert.True(t, model.FortinetConfig.IsNull(), "fortinet_config should be null")
	assert.True(t, model.MerakiConfig.IsNull(), "meraki_config should be null")
	assert.True(t, model.PaloAltoConfig.IsNull(), "palo_alto_config should be null")
	assert.True(t, model.PrismaConfig.IsNull(), "prisma_config should be null")
	assert.True(t, model.SixwindConfig.IsNull(), "sixwind_config should be null")
	assert.True(t, model.VersaConfig.IsNull(), "versa_config should be null")
	assert.True(t, model.VmwareConfig.IsNull(), "vmware_config should be null")

	// Scalar fields still migrated
	assert.Equal(t, "mve-uid-abc123", model.UID.ValueString())
}

func TestMoveState_MVE_V1ToV2_VNICs_VLANStripped(t *testing.T) {
	v1 := baseV1MVEState()
	v1["vendor_config"] = map[string]interface{}{
		"vendor": "aruba", "image_id": float64(1), "product_size": "MEDIUM",
		"account_name": "a", "account_key": "b", "system_tag": "c",
	}
	v1["vnics"] = []interface{}{
		map[string]interface{}{"description": "vNIC-0", "vlan": float64(100)},
		map[string]interface{}{"description": "vNIC-1", "vlan": float64(200)},
		map[string]interface{}{"description": "vNIC-2", "vlan": float64(300)},
	}

	rawJSON, err := json.Marshal(v1)
	require.NoError(t, err)

	resp, model := invokeMVEStateMover(t, "registry.terraform.io/megaport/megaport", "megaport_mve", rawJSON)
	require.False(t, resp.Diagnostics.HasError(), "unexpected diagnostics: %v", resp.Diagnostics)
	require.NotNil(t, model)

	assert.False(t, model.NetworkInterfaces.IsNull())
	vnics := model.NetworkInterfaces.Elements()
	assert.Len(t, vnics, 3)

	for i, el := range vnics {
		obj, ok := el.(types.Object)
		require.True(t, ok, "vNIC %d should be types.Object", i)
		attrTypes := obj.AttributeTypes(context.Background())
		assert.NotContains(t, attrTypes, "vlan", "vNIC %d should not have vlan attribute", i)
		assert.Contains(t, attrTypes, "description", "vNIC %d should have description attribute", i)
		attrs := obj.Attributes()
		assert.Equal(t, []string{"description"}, sortedKeys(attrTypes), "vNIC %d should only have description", i)
		_ = attrs
	}
}

func TestMoveState_MVE_V1ToV2_NilOptionals(t *testing.T) {
	v1 := baseV1MVEState()
	v1["cost_centre"] = nil
	v1["diversity_zone"] = nil
	v1["promo_code"] = nil
	v1["terminate_date"] = nil
	v1["attribute_tags"] = nil
	v1["resource_tags"] = nil
	v1["vendor_config"] = map[string]interface{}{
		"vendor": "meraki", "image_id": float64(99), "product_size": "SMALL",
		"mve_label": nil, "token": "tok-abc",
	}
	v1["vnics"] = []interface{}{
		map[string]interface{}{"description": "eth0", "vlan": float64(0)},
	}

	rawJSON, err := json.Marshal(v1)
	require.NoError(t, err)

	resp, model := invokeMVEStateMover(t, "registry.terraform.io/megaport/megaport", "megaport_mve", rawJSON)
	require.False(t, resp.Diagnostics.HasError(), "unexpected diagnostics: %v", resp.Diagnostics)
	require.NotNil(t, model)

	assert.True(t, model.CostCentre.IsNull())
	assert.True(t, model.DiversityZone.IsNull())
	assert.True(t, model.PromoCode.IsNull())
	assert.True(t, model.AttributeTags.IsNull())
	assert.True(t, model.ResourceTags.IsNull())

	assert.False(t, model.MerakiConfig.IsNull(), "meraki_config should be populated")
	var meraki merakiConfigModel
	diags := model.MerakiConfig.As(context.Background(), &meraki, basetypes.ObjectAsOptions{})
	require.False(t, diags.HasError())
	assert.Equal(t, "tok-abc", meraki.Token.ValueString())
	assert.True(t, meraki.MVELabel.IsNull(), "mve_label should be null")
}

func TestMoveState_MVE_WrongProvider(t *testing.T) {
	v1 := baseV1MVEState()
	v1["vendor_config"] = map[string]interface{}{"vendor": "aruba", "image_id": float64(1), "product_size": "MEDIUM", "account_name": "a", "account_key": "b", "system_tag": "c"}
	v1["vnics"] = []interface{}{}
	rawJSON, err := json.Marshal(v1)
	require.NoError(t, err)

	resp, model := invokeMVEStateMover(t, "registry.terraform.io/other/provider", "megaport_mve", rawJSON)
	require.False(t, resp.Diagnostics.HasError())
	assert.Nil(t, model, "model should be nil when provider doesn't match (skipped)")
}

func TestMoveState_MVE_WrongType(t *testing.T) {
	v1 := baseV1MVEState()
	v1["vendor_config"] = map[string]interface{}{"vendor": "aruba", "image_id": float64(1), "product_size": "MEDIUM", "account_name": "a", "account_key": "b", "system_tag": "c"}
	v1["vnics"] = []interface{}{}
	rawJSON, err := json.Marshal(v1)
	require.NoError(t, err)

	resp, model := invokeMVEStateMover(t, "registry.terraform.io/megaport/megaport", "megaport_port", rawJSON)
	require.False(t, resp.Diagnostics.HasError())
	assert.Nil(t, model, "model should be nil when type doesn't match (skipped)")
}

func TestMoveState_MVE_V1ToV2_Aviatrix(t *testing.T) {
	v1 := baseV1MVEState()
	v1["vendor"] = "aviatrix"
	v1["vendor_config"] = map[string]interface{}{
		"vendor":       "aviatrix",
		"image_id":     float64(200),
		"product_size": "MEDIUM",
		"mve_label":    "avx-label",
		"cloud_init":   "#!/bin/bash\necho hello",
	}
	v1["vnics"] = []interface{}{
		map[string]interface{}{"description": "eth0", "vlan": float64(0)},
	}

	rawJSON, err := json.Marshal(v1)
	require.NoError(t, err)

	resp, model := invokeMVEStateMover(t, "registry.terraform.io/megaport/megaport", "megaport_mve", rawJSON)
	require.False(t, resp.Diagnostics.HasError(), "unexpected diagnostics: %v", resp.Diagnostics)
	require.NotNil(t, model)

	assert.False(t, model.AviatrixConfig.IsNull(), "aviatrix_config should be populated")
	assert.True(t, model.ArubaConfig.IsNull())

	var avx aviatrixConfigModel
	diags := model.AviatrixConfig.As(context.Background(), &avx, basetypes.ObjectAsOptions{})
	require.False(t, diags.HasError())
	assert.Equal(t, int64(200), avx.ImageID.ValueInt64())
	assert.Equal(t, "avx-label", avx.MVELabel.ValueString())
	assert.Equal(t, "#!/bin/bash\necho hello", avx.CloudInit.ValueString())
}

func TestMoveState_MVE_V1ToV2_Fortinet(t *testing.T) {
	v1 := baseV1MVEState()
	v1["vendor"] = "fortinet"
	v1["vendor_config"] = map[string]interface{}{
		"vendor":               "fortinet",
		"image_id":             float64(300),
		"product_size":         "LARGE",
		"mve_label":            "fort-label",
		"admin_ssh_public_key": "ssh-rsa AAAA...",
		"ssh_public_key":       "ssh-rsa BBBB...",
		"license_data":         "FGTVM010000000000",
	}
	v1["vnics"] = []interface{}{
		map[string]interface{}{"description": "port1", "vlan": float64(0)},
	}

	rawJSON, err := json.Marshal(v1)
	require.NoError(t, err)

	resp, model := invokeMVEStateMover(t, "registry.terraform.io/megaport/megaport", "megaport_mve", rawJSON)
	require.False(t, resp.Diagnostics.HasError(), "unexpected diagnostics: %v", resp.Diagnostics)
	require.NotNil(t, model)

	assert.False(t, model.FortinetConfig.IsNull(), "fortinet_config should be populated")
	assert.True(t, model.ArubaConfig.IsNull())

	var fort fortinetConfigModel
	diags := model.FortinetConfig.As(context.Background(), &fort, basetypes.ObjectAsOptions{})
	require.False(t, diags.HasError())
	assert.Equal(t, int64(300), fort.ImageID.ValueInt64())
	assert.Equal(t, "ssh-rsa AAAA...", fort.AdminSSHPublicKey.ValueString())
	assert.Equal(t, "FGTVM010000000000", fort.LicenseData.ValueString())
}

func TestMoveState_MVE_V1ToV2_Meraki(t *testing.T) {
	v1 := baseV1MVEState()
	v1["vendor"] = "meraki"
	v1["vendor_config"] = map[string]interface{}{
		"vendor":       "meraki",
		"image_id":     float64(400),
		"product_size": "MEDIUM",
		"mve_label":    "meraki-label",
		"token":        "tok-abc123",
	}
	v1["vnics"] = []interface{}{
		map[string]interface{}{"description": "WAN", "vlan": float64(0)},
	}

	rawJSON, err := json.Marshal(v1)
	require.NoError(t, err)

	resp, model := invokeMVEStateMover(t, "registry.terraform.io/megaport/megaport", "megaport_mve", rawJSON)
	require.False(t, resp.Diagnostics.HasError(), "unexpected diagnostics: %v", resp.Diagnostics)
	require.NotNil(t, model)

	assert.False(t, model.MerakiConfig.IsNull(), "meraki_config should be populated")
	assert.True(t, model.ArubaConfig.IsNull())

	var mer merakiConfigModel
	diags := model.MerakiConfig.As(context.Background(), &mer, basetypes.ObjectAsOptions{})
	require.False(t, diags.HasError())
	assert.Equal(t, int64(400), mer.ImageID.ValueInt64())
	assert.Equal(t, "tok-abc123", mer.Token.ValueString())
}

func TestMoveState_MVE_V1ToV2_PaloAlto(t *testing.T) {
	v1 := baseV1MVEState()
	v1["vendor"] = "palo_alto"
	v1["vendor_config"] = map[string]interface{}{
		"vendor":               "palo_alto",
		"image_id":             float64(500),
		"product_size":         "LARGE",
		"mve_label":            "pa-label",
		"admin_ssh_public_key": "ssh-rsa CCCC...",
		"ssh_public_key":       "ssh-rsa DDDD...",
		"admin_password_hash":  "$6$hash...",
		"license_data":         "PA-VM-LICENSE",
	}
	v1["vnics"] = []interface{}{
		map[string]interface{}{"description": "eth0", "vlan": float64(0)},
		map[string]interface{}{"description": "eth1", "vlan": float64(0)},
	}

	rawJSON, err := json.Marshal(v1)
	require.NoError(t, err)

	resp, model := invokeMVEStateMover(t, "registry.terraform.io/megaport/megaport", "megaport_mve", rawJSON)
	require.False(t, resp.Diagnostics.HasError(), "unexpected diagnostics: %v", resp.Diagnostics)
	require.NotNil(t, model)

	assert.False(t, model.PaloAltoConfig.IsNull(), "palo_alto_config should be populated")
	assert.True(t, model.ArubaConfig.IsNull())

	var pa paloAltoConfigModel
	diags := model.PaloAltoConfig.As(context.Background(), &pa, basetypes.ObjectAsOptions{})
	require.False(t, diags.HasError())
	assert.Equal(t, int64(500), pa.ImageID.ValueInt64())
	assert.Equal(t, "$6$hash...", pa.AdminPasswordHash.ValueString())
	assert.Equal(t, "PA-VM-LICENSE", pa.LicenseData.ValueString())
}

func TestMoveState_MVE_V1ToV2_Prisma(t *testing.T) {
	v1 := baseV1MVEState()
	v1["vendor"] = "prisma"
	v1["vendor_config"] = map[string]interface{}{
		"vendor":       "prisma",
		"image_id":     float64(600),
		"product_size": "MEDIUM",
		"mve_label":    "prisma-label",
		"ion_key":      "ion-key-xyz",
		"secret_key":   "secret-key-abc",
	}
	v1["vnics"] = []interface{}{
		map[string]interface{}{"description": "eth0", "vlan": float64(0)},
	}

	rawJSON, err := json.Marshal(v1)
	require.NoError(t, err)

	resp, model := invokeMVEStateMover(t, "registry.terraform.io/megaport/megaport", "megaport_mve", rawJSON)
	require.False(t, resp.Diagnostics.HasError(), "unexpected diagnostics: %v", resp.Diagnostics)
	require.NotNil(t, model)

	assert.False(t, model.PrismaConfig.IsNull(), "prisma_config should be populated")
	assert.True(t, model.ArubaConfig.IsNull())

	var pr prismaConfigModel
	diags := model.PrismaConfig.As(context.Background(), &pr, basetypes.ObjectAsOptions{})
	require.False(t, diags.HasError())
	assert.Equal(t, int64(600), pr.ImageID.ValueInt64())
	assert.Equal(t, "ion-key-xyz", pr.IONKey.ValueString())
	assert.Equal(t, "secret-key-abc", pr.SecretKey.ValueString())
}

func TestMoveState_MVE_V1ToV2_Sixwind(t *testing.T) {
	v1 := baseV1MVEState()
	v1["vendor"] = "6wind"
	v1["vendor_config"] = map[string]interface{}{
		"vendor":         "6wind",
		"image_id":       float64(700),
		"product_size":   "MEDIUM",
		"mve_label":      "6wind-label",
		"ssh_public_key": "ssh-rsa EEEE...",
	}
	v1["vnics"] = []interface{}{
		map[string]interface{}{"description": "eth0", "vlan": float64(0)},
	}

	rawJSON, err := json.Marshal(v1)
	require.NoError(t, err)

	resp, model := invokeMVEStateMover(t, "registry.terraform.io/megaport/megaport", "megaport_mve", rawJSON)
	require.False(t, resp.Diagnostics.HasError(), "unexpected diagnostics: %v", resp.Diagnostics)
	require.NotNil(t, model)

	assert.False(t, model.SixwindConfig.IsNull(), "sixwind_config should be populated")
	assert.True(t, model.ArubaConfig.IsNull())

	var sw sixwindConfigModel
	diags := model.SixwindConfig.As(context.Background(), &sw, basetypes.ObjectAsOptions{})
	require.False(t, diags.HasError())
	assert.Equal(t, int64(700), sw.ImageID.ValueInt64())
	assert.Equal(t, "ssh-rsa EEEE...", sw.SSHPublicKey.ValueString())
}

// sortedKeys returns sorted keys of a map for deterministic comparison.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
