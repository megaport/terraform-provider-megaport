package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// nullAttrValues returns every attribute in attrTypes set to null.
func nullAttrValues(attrTypes map[string]attr.Type) map[string]attr.Value {
	m := make(map[string]attr.Value, len(attrTypes))
	for name, attrType := range attrTypes {
		m[name] = nullAttrValue(attrType)
	}
	return m
}

func nullAttrValue(attrType attr.Type) attr.Value {
	v, err := attrType.ValueFromTerraform(context.Background(), tftypes.NewValue(attrType.TerraformType(context.Background()), nil))
	if err != nil {
		panic(err)
	}
	return v
}

// testPartnerConfig builds a partner config object with one nested config set.
func testPartnerConfig(partner, configAttr string, config attr.Value) types.Object {
	attrs := nullAttrValues(vxcPartnerConfigAttrs)
	attrs["partner"] = types.StringValue(partner)
	if configAttr != "" {
		attrs[configAttr] = config
	}
	return types.ObjectValueMust(vxcPartnerConfigAttrs, attrs)
}

func vxcAWSPartnerConfig(connectType string, authKey types.String) types.Object {
	attrs := nullAttrValues(vxcPartnerConfigAWSAttrs)
	attrs["connect_type"] = types.StringValue(connectType)
	attrs["owner_account"] = types.StringValue("123456789012")
	attrs["auth_key"] = authKey
	return testPartnerConfig("aws", "aws_config", types.ObjectValueMust(vxcPartnerConfigAWSAttrs, attrs))
}

// vxcVrouterPartnerConfig builds a vrouter_config with one interface and one
// BGP connection per password.
func vxcVrouterPartnerConfig(passwords ...types.String) types.Object {
	return testPartnerConfig("vrouter", "vrouter_config", vxcInterfacesConfig(vxcPartnerConfigVrouterAttrs, vxcVrouterInterfaceAttrs, bgpVrouterConnectionConfig, passwords...))
}

// vxcAEndPartnerConfig builds the deprecated partner_a_end_config shape.
func vxcAEndPartnerConfig(passwords ...types.String) types.Object {
	return testPartnerConfig("a-end", "partner_a_end_config", vxcInterfacesConfig(vxcPartnerConfigAEndAttrs, vxcPartnerConfigAEndInterfaceAttrs, bgpConnectionConfig, passwords...))
}

func vxcInterfacesConfig(configAttrs, ifaceAttrs, bgpAttrs map[string]attr.Type, passwords ...types.String) types.Object {
	bgpType := types.ObjectType{}.WithAttributeTypes(bgpAttrs)
	bgps := make([]attr.Value, 0, len(passwords))
	for _, password := range passwords {
		bgp := nullAttrValues(bgpAttrs)
		bgp["peer_asn"] = types.Int64Value(64512)
		bgp["local_ip_address"] = types.StringValue("10.0.0.1")
		bgp["peer_ip_address"] = types.StringValue("10.0.0.2")
		bgp["password"] = password
		bgps = append(bgps, types.ObjectValueMust(bgpAttrs, bgp))
	}
	iface := nullAttrValues(ifaceAttrs)
	iface["ip_addresses"] = types.ListValueMust(types.StringType, []attr.Value{types.StringValue("10.0.0.1/30")})
	iface["bgp_connections"] = types.ListValueMust(bgpType, bgps)
	ifaceType := types.ObjectType{}.WithAttributeTypes(ifaceAttrs)
	return types.ObjectValueMust(configAttrs, map[string]attr.Value{
		"interfaces": types.ListValueMust(ifaceType, []attr.Value{types.ObjectValueMust(ifaceAttrs, iface)}),
	})
}

func TestVXCResourceValidateConfig(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	r := &vxcResource{}

	schemaResp := fwresource.SchemaResponse{}
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	objType, ok := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	if !ok {
		t.Fatal("schema type is not tftypes.Object")
	}

	key := types.StringValue("sharedMD5key")
	otherKey := types.StringValue("otherMD5key")

	cases := []struct {
		name       string
		aEnd       attr.Value
		bEnd       attr.Value
		wantErrors int
		wantText   []string
	}{
		{
			name:       "missing password",
			aEnd:       vxcVrouterPartnerConfig(types.StringNull()),
			bEnd:       vxcAWSPartnerConfig("AWS", key),
			wantErrors: 1,
			wantText:   []string{"Missing BGP password", "`password`", "the AWS end's `aws_config.auth_key`", "omit the vRouter end's explicit config"},
		},
		{
			name:       "missing password and no auth_key",
			aEnd:       vxcVrouterPartnerConfig(types.StringNull()),
			bEnd:       vxcAWSPartnerConfig("AWS", types.StringNull()),
			wantErrors: 1,
			wantText:   []string{"Missing BGP password"},
		},
		{
			name:       "password differs from auth_key",
			aEnd:       vxcVrouterPartnerConfig(otherKey),
			bEnd:       vxcAWSPartnerConfig("AWS", key),
			wantErrors: 1,
			wantText:   []string{"does not match auth_key", "omit the vRouter end's explicit config"},
		},
		{
			name:       "password set and no auth_key",
			aEnd:       vxcVrouterPartnerConfig(key),
			bEnd:       vxcAWSPartnerConfig("AWS", types.StringNull()),
			wantErrors: 1,
			wantText:   []string{"does not match auth_key"},
		},
		{
			// A blank password reaches the API as no password at all, so a
			// blank pair on both ends is the reported broken shape.
			name:       "empty password and empty auth_key",
			aEnd:       vxcVrouterPartnerConfig(types.StringValue("")),
			bEnd:       vxcAWSPartnerConfig("AWS", types.StringValue("")),
			wantErrors: 1,
			wantText:   []string{"Missing BGP password"},
		},
		{
			name:       "empty password and set auth_key",
			aEnd:       vxcVrouterPartnerConfig(types.StringValue("")),
			bEnd:       vxcAWSPartnerConfig("AWS", key),
			wantErrors: 1,
			wantText:   []string{"Missing BGP password"},
		},
		{
			name:       "one error per bad connection",
			aEnd:       vxcVrouterPartnerConfig(key, types.StringNull(), otherKey),
			bEnd:       vxcAWSPartnerConfig("AWS", key),
			wantErrors: 2,
		},
		{
			name:       "deprecated a-end shape is checked too",
			aEnd:       vxcAEndPartnerConfig(types.StringNull()),
			bEnd:       vxcAWSPartnerConfig("AWS", key),
			wantErrors: 1,
			wantText:   []string{"Missing BGP password"},
		},
		{
			name:       "AWS on a_end and vrouter on b_end is checked too",
			aEnd:       vxcAWSPartnerConfig("AWS", key),
			bEnd:       vxcVrouterPartnerConfig(types.StringNull()),
			wantErrors: 1,
			wantText:   []string{"Missing BGP password"},
		},
		{
			name: "password matches auth_key",
			aEnd: vxcVrouterPartnerConfig(key),
			bEnd: vxcAWSPartnerConfig("AWS", key),
		},
		{
			name: "no a_end_partner_config",
			aEnd: types.ObjectNull(vxcPartnerConfigAttrs),
			bEnd: vxcAWSPartnerConfig("AWS", types.StringNull()),
		},
		{
			name: "AWSHC is not checked",
			aEnd: vxcVrouterPartnerConfig(otherKey),
			bEnd: vxcAWSPartnerConfig("AWSHC", types.StringNull()),
		},
		{
			name: "non-AWS b-end is not checked",
			aEnd: vxcVrouterPartnerConfig(types.StringNull()),
			bEnd: testPartnerConfig("transit", "", nil),
		},
		{
			name: "unknown password is skipped",
			aEnd: vxcVrouterPartnerConfig(types.StringUnknown()),
			bEnd: vxcAWSPartnerConfig("AWS", key),
		},
		{
			name: "unknown auth_key skips only the comparison",
			aEnd: vxcVrouterPartnerConfig(key),
			bEnd: vxcAWSPartnerConfig("AWS", types.StringUnknown()),
		},
		{
			// The password is wrong whatever the key turns out to be.
			name:       "unknown auth_key still rejects a missing password",
			aEnd:       vxcVrouterPartnerConfig(types.StringNull()),
			bEnd:       vxcAWSPartnerConfig("AWS", types.StringUnknown()),
			wantErrors: 1,
			wantText:   []string{"Missing BGP password"},
		},
		{
			name:       "unknown auth_key still rejects an empty password",
			aEnd:       vxcVrouterPartnerConfig(types.StringValue("")),
			bEnd:       vxcAWSPartnerConfig("AWS", types.StringUnknown()),
			wantErrors: 1,
			wantText:   []string{"Missing BGP password"},
		},
		{
			name: "unknown a_end_partner_config is skipped",
			aEnd: types.ObjectUnknown(vxcPartnerConfigAttrs),
			bEnd: vxcAWSPartnerConfig("AWS", key),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			configAttrs := nullValueMap(objType)
			configAttrs["product_name"] = tftypes.NewValue(tftypes.String, "vxc-one")
			for name, val := range map[string]attr.Value{"a_end_partner_config": tc.aEnd, "b_end_partner_config": tc.bEnd} {
				raw, err := val.ToTerraformValue(ctx)
				if err != nil {
					t.Fatalf("%s: %v", name, err)
				}
				configAttrs[name] = raw
			}

			resp := fwresource.ValidateConfigResponse{}
			r.ValidateConfig(ctx, fwresource.ValidateConfigRequest{
				Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, configAttrs)},
			}, &resp)

			if got := resp.Diagnostics.ErrorsCount(); got != tc.wantErrors {
				t.Fatalf("expected %d errors, got %d: %v", tc.wantErrors, got, resp.Diagnostics.Errors())
			}
			if tc.wantErrors == 0 {
				return
			}
			first := resp.Diagnostics.Errors()[0]
			text := first.Summary() + " " + first.Detail()
			for _, want := range tc.wantText {
				if !strings.Contains(text, want) {
					t.Errorf("expected diagnostic to contain %q, got: %s", want, text)
				}
			}
		})
	}
}

// TestVXCResourceCreateChecksBGPPassword covers the apply-time gate. A password
// that comes from another resource is unknown while the config is validated, so
// Create is the last place that can reject it.
func TestVXCResourceCreateChecksBGPPassword(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	r := &vxcResource{}

	schemaResp := fwresource.SchemaResponse{}
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	objType, ok := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	if !ok {
		t.Fatal("schema type is not tftypes.Object")
	}

	planAttrs := nullValueMap(objType)
	planAttrs["product_name"] = tftypes.NewValue(tftypes.String, "vxc-one")
	for name, val := range map[string]attr.Value{
		"a_end_partner_config": vxcVrouterPartnerConfig(types.StringValue("")),
		"b_end_partner_config": vxcAWSPartnerConfig("AWS", types.StringValue("")),
	} {
		raw, err := val.ToTerraformValue(ctx)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		planAttrs[name] = raw
	}

	resp := fwresource.CreateResponse{}
	r.Create(ctx, fwresource.CreateRequest{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, planAttrs)},
	}, &resp)

	if got := resp.Diagnostics.ErrorsCount(); got != 1 {
		t.Fatalf("expected 1 error, got %d: %v", got, resp.Diagnostics.Errors())
	}
	if summary := resp.Diagnostics.Errors()[0].Summary(); !strings.Contains(summary, "Missing BGP password") {
		t.Errorf("expected a missing password error, got: %s", summary)
	}
}

// TestVXCResourceUpdateChecksBGPPassword covers the apply-time gate on Update,
// which runs checkPartnerConfigUpdatable before checkAWSBGPPassword. Plan and
// state carry the same partner config so that check passes without error and
// the BGP password check still runs.
func TestVXCResourceUpdateChecksBGPPassword(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	r := &vxcResource{}

	schemaResp := fwresource.SchemaResponse{}
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	objType, ok := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	if !ok {
		t.Fatal("schema type is not tftypes.Object")
	}

	attrs := nullValueMap(objType)
	attrs["product_name"] = tftypes.NewValue(tftypes.String, "vxc-one")
	for name, val := range map[string]attr.Value{
		"a_end_partner_config": vxcVrouterPartnerConfig(types.StringValue("")),
		"b_end_partner_config": vxcAWSPartnerConfig("AWS", types.StringValue("")),
	} {
		raw, err := val.ToTerraformValue(ctx)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		attrs[name] = raw
	}
	tfValue := tftypes.NewValue(objType, attrs)

	resp := fwresource.UpdateResponse{}
	r.Update(ctx, fwresource.UpdateRequest{
		Plan:  tfsdk.Plan{Schema: schemaResp.Schema, Raw: tfValue},
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: tfValue},
	}, &resp)

	if got := resp.Diagnostics.ErrorsCount(); got != 1 {
		t.Fatalf("expected 1 error, got %d: %v", got, resp.Diagnostics.Errors())
	}
	if summary := resp.Diagnostics.Errors()[0].Summary(); !strings.Contains(summary, "Missing BGP password") {
		t.Errorf("expected a missing password error, got: %s", summary)
	}
}
