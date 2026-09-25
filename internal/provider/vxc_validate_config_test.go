package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// vxcConfigType returns the tftypes shape of the megaport_vxc schema.
func vxcConfigType(t *testing.T) tftypes.Object {
	t.Helper()
	schemaResp := &resource.SchemaResponse{}
	(&vxcResource{}).Schema(context.Background(), resource.SchemaRequest{}, schemaResp)
	require.False(t, schemaResp.Diagnostics.HasError())
	objType, ok := schemaResp.Schema.Type().TerraformType(context.Background()).(tftypes.Object)
	require.True(t, ok)
	return objType
}

// nullAttrs returns every attribute of obj set to null.
func nullAttrs(obj tftypes.Object) map[string]tftypes.Value {
	attrs := make(map[string]tftypes.Value, len(obj.AttributeTypes))
	for name, attrType := range obj.AttributeTypes {
		attrs[name] = tftypes.NewValue(attrType, nil)
	}
	return attrs
}

// emptyBlock returns a present block with every attribute null. The validator
// only reads whether a block is set, not what is inside it.
func emptyBlock(t *testing.T, obj tftypes.Object, name string) tftypes.Value {
	t.Helper()
	blockType, ok := obj.AttributeTypes[name].(tftypes.Object)
	require.True(t, ok)
	return tftypes.NewValue(blockType, nullAttrs(blockType))
}

// validateVXCConfig runs ValidateConfig over a megaport_vxc config whose only
// set attribute is the named partner config, built by mutate.
func validateVXCConfig(t *testing.T, attribute string, mutate func(t *testing.T, partner tftypes.Object, attrs map[string]tftypes.Value)) []string {
	t.Helper()
	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	(&vxcResource{}).Schema(ctx, resource.SchemaRequest{}, schemaResp)

	vxcType := vxcConfigType(t)
	partnerType, ok := vxcType.AttributeTypes[attribute].(tftypes.Object)
	require.True(t, ok)

	partnerAttrs := nullAttrs(partnerType)
	mutate(t, partnerType, partnerAttrs)

	vxcAttrs := nullAttrs(vxcType)
	vxcAttrs[attribute] = tftypes.NewValue(partnerType, partnerAttrs)

	resp := &resource.ValidateConfigResponse{}
	(&vxcResource{}).ValidateConfig(ctx, resource.ValidateConfigRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    tftypes.NewValue(vxcType, vxcAttrs),
		},
	}, resp)

	details := []string{}
	for _, d := range resp.Diagnostics.Errors() {
		details = append(details, d.Detail())
	}
	return details
}

func TestVXCValidateConfigRejectsBlockTheBEndPartnerDoesNotUse(t *testing.T) {
	details := validateVXCConfig(t, "b_end_partner_config", func(t *testing.T, partner tftypes.Object, attrs map[string]tftypes.Value) {
		attrs["partner"] = tftypes.NewValue(tftypes.String, "azure")
		attrs["azure_config"] = emptyBlock(t, partner, "azure_config")
		attrs["vrouter_config"] = emptyBlock(t, partner, "vrouter_config")
	})

	require.Len(t, details, 1)
	assert.Contains(t, details[0], "b_end_partner_config sets vrouter_config.")
	assert.Contains(t, details[0], `Partner "azure" uses azure_config.`)
}

func TestVXCValidateConfigRejectsBlockTheAEndPartnerDoesNotUse(t *testing.T) {
	details := validateVXCConfig(t, "a_end_partner_config", func(t *testing.T, partner tftypes.Object, attrs map[string]tftypes.Value) {
		attrs["partner"] = tftypes.NewValue(tftypes.String, "vrouter")
		attrs["vrouter_config"] = emptyBlock(t, partner, "vrouter_config")
		attrs["aws_config"] = emptyBlock(t, partner, "aws_config")
	})

	require.Len(t, details, 1)
	assert.Contains(t, details[0], "a_end_partner_config sets aws_config.")
	assert.Contains(t, details[0], `Partner "vrouter" uses vrouter_config.`)
}

func TestVXCValidateConfigReportsEveryBlockThePartnerDoesNotUse(t *testing.T) {
	details := validateVXCConfig(t, "b_end_partner_config", func(t *testing.T, partner tftypes.Object, attrs map[string]tftypes.Value) {
		attrs["partner"] = tftypes.NewValue(tftypes.String, "aws")
		attrs["aws_config"] = emptyBlock(t, partner, "aws_config")
		attrs["vrouter_config"] = emptyBlock(t, partner, "vrouter_config")
		attrs["partner_a_end_config"] = emptyBlock(t, partner, "partner_a_end_config")
	})

	assert.Len(t, details, 2)
}

func TestVXCValidateConfigAcceptsTheBlockThePartnerUses(t *testing.T) {
	details := validateVXCConfig(t, "a_end_partner_config", func(t *testing.T, partner tftypes.Object, attrs map[string]tftypes.Value) {
		attrs["partner"] = tftypes.NewValue(tftypes.String, "a-end")
		attrs["partner_a_end_config"] = emptyBlock(t, partner, "partner_a_end_config")
	})

	assert.Empty(t, details)
}

func TestVXCValidateConfigAcceptsTransitWithNoBlock(t *testing.T) {
	details := validateVXCConfig(t, "b_end_partner_config", func(t *testing.T, partner tftypes.Object, attrs map[string]tftypes.Value) {
		attrs["partner"] = tftypes.NewValue(tftypes.String, "transit")
	})

	assert.Empty(t, details)
}

func TestVXCValidateConfigRejectsABlockTransitDoesNotUse(t *testing.T) {
	details := validateVXCConfig(t, "b_end_partner_config", func(t *testing.T, partner tftypes.Object, attrs map[string]tftypes.Value) {
		attrs["partner"] = tftypes.NewValue(tftypes.String, "transit")
		attrs["vrouter_config"] = emptyBlock(t, partner, "vrouter_config")
	})

	require.Len(t, details, 1)
	assert.Contains(t, details[0], `Partner "transit" uses no configuration block.`)
}

func TestVXCValidateConfigSkipsUnknownValues(t *testing.T) {
	unknownBlock := validateVXCConfig(t, "b_end_partner_config", func(t *testing.T, partner tftypes.Object, attrs map[string]tftypes.Value) {
		attrs["partner"] = tftypes.NewValue(tftypes.String, "azure")
		attrs["azure_config"] = emptyBlock(t, partner, "azure_config")
		attrs["vrouter_config"] = tftypes.NewValue(partner.AttributeTypes["vrouter_config"], tftypes.UnknownValue)
	})
	assert.Empty(t, unknownBlock)

	unknownPartner := validateVXCConfig(t, "b_end_partner_config", func(t *testing.T, partner tftypes.Object, attrs map[string]tftypes.Value) {
		attrs["partner"] = tftypes.NewValue(tftypes.String, tftypes.UnknownValue)
		attrs["vrouter_config"] = emptyBlock(t, partner, "vrouter_config")
	})
	assert.Empty(t, unknownPartner)
}

func TestVXCValidateConfigSkipsAnAbsentPartnerConfig(t *testing.T) {
	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	(&vxcResource{}).Schema(ctx, resource.SchemaRequest{}, schemaResp)

	vxcType := vxcConfigType(t)
	resp := &resource.ValidateConfigResponse{}
	(&vxcResource{}).ValidateConfig(ctx, resource.ValidateConfigRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    tftypes.NewValue(vxcType, nullAttrs(vxcType)),
		},
	}, resp)

	assert.False(t, resp.Diagnostics.HasError())
}
