package provider

import (
	"context"
	"slices"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// awsInterconnectProductType is the product type the API reports for an AWS
// Interconnect. megaportgo has no constant for it.
const awsInterconnectProductType = "AWS_INTERCONNECT"

// vrouterPartner is a vrouter partner config with one interface on ipAddress.
func (b *vxcValueBuilder) vrouterPartner(t *testing.T, ipAddress string) tftypes.Value {
	t.Helper()
	vrouterType, ok := b.partnerTyp.AttributeTypes["vrouter_config"].(tftypes.Object)
	if !ok {
		t.Fatal("vrouter_config type is not tftypes.Object")
	}
	listType, ok := vrouterType.AttributeTypes["interfaces"].(tftypes.List)
	if !ok {
		t.Fatal("interfaces type is not tftypes.List")
	}
	ifaceType, ok := listType.ElementType.(tftypes.Object)
	if !ok {
		t.Fatal("interface type is not tftypes.Object")
	}
	iface := nullValueMap(ifaceType)
	iface["ip_addresses"] = tftypes.NewValue(tftypes.List{ElementType: tftypes.String},
		[]tftypes.Value{tftypes.NewValue(tftypes.String, ipAddress)})
	vrouter := nullValueMap(vrouterType)
	vrouter["interfaces"] = tftypes.NewValue(listType, []tftypes.Value{tftypes.NewValue(ifaceType, iface)})
	attrs := nullValueMap(b.partnerTyp)
	attrs["partner"] = tftypes.NewValue(tftypes.String, "vrouter")
	attrs["vrouter_config"] = tftypes.NewValue(vrouterType, vrouter)
	return tftypes.NewValue(b.partnerTyp, attrs)
}

func TestVXCCreate_AWSInterconnectEndVRouterConfig(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	ps := newPreflightServer(t, nil)
	ps.productTypeOf = map[string]string{"interconnect-a": awsInterconnectProductType}
	b := newVXCValueBuilder(t)

	plan := b.with(t, b.vxc(
		b.end(vxcEndSpec{productUID: "interconnect-a"}),
		b.end(vxcEndSpec{productUID: "port-b", orderedVLAN: int64p(100)}),
		nil,
	), map[string]tftypes.Value{"a_end_partner_config": b.vrouterPartner(t, "10.0.0.1/30")})

	resp := fwresource.CreateResponse{State: tfsdk.State{Schema: b.schema}}
	ps.resource(t).Create(ctx, fwresource.CreateRequest{
		Config: tfsdk.Config{Schema: b.schema, Raw: plan},
		Plan:   tfsdk.Plan{Schema: b.schema, Raw: plan},
	}, &resp)

	// Order validation falls to the fake's default 500, and Create stops there.
	errs := resp.Diagnostics.Errors()
	if len(errs) != 1 || errs[0].Summary() != "Validation error while attempting to create VXC" {
		t.Fatalf("expected Create to fail only at order validation, got %v", errs)
	}
	if !slices.Contains(ps.productTypes, "interconnect-a") {
		t.Fatalf("expected an A-End product type lookup, got %v", ps.productTypes)
	}
}

func TestVXCUpdate_AWSInterconnectEndPartnerChange(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	ps := newPreflightServer(t, nil)
	ps.productTypeOf = map[string]string{"interconnect-a": awsInterconnectProductType}
	b := newVXCValueBuilder(t)

	vxc := func(ipAddress string) tftypes.Value {
		return b.with(t, b.vxc(
			b.end(vxcEndSpec{productUID: "interconnect-a"}),
			b.end(vxcEndSpec{productUID: "port-b", orderedVLAN: int64p(100), vlan: int64p(100)}),
			nil,
		), map[string]tftypes.Value{
			"product_uid":          tftypes.NewValue(tftypes.String, "vxc-1"),
			"a_end_partner_config": b.vrouterPartner(t, ipAddress),
		})
	}
	state, plan := vxc("10.0.0.1/30"), vxc("10.0.0.5/30")

	resp := fwresource.UpdateResponse{State: tfsdk.State{Schema: b.schema, Raw: state}}
	ps.resource(t).Update(ctx, fwresource.UpdateRequest{
		Config: tfsdk.Config{Schema: b.schema, Raw: plan},
		Plan:   tfsdk.Plan{Schema: b.schema, Raw: plan},
		State:  tfsdk.State{Schema: b.schema, Raw: state},
	}, &resp)

	if len(ps.updateBodies) != 1 {
		t.Fatalf("expected one VXC update request, got %d: %v", len(ps.updateBodies), resp.Diagnostics.Errors())
	}
	if _, ok := ps.updateBodies[0]["aEndConfig"]; !ok {
		t.Fatalf("expected the update to carry the A-End partner config, got %v", ps.updateBodies[0])
	}
}
