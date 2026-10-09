package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// vxcPlanFixture holds the megaport_vxc schema type and the nested types the
// plan tests build values for.
type vxcPlanFixture struct {
	vxc, end, partner, aws tftypes.Object
}

func newVXCPlanFixture(t *testing.T) vxcPlanFixture {
	t.Helper()
	ctx := context.Background()

	schemaResp := fwresource.SchemaResponse{}
	(&vxcResource{}).Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)

	vxc, ok := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	if !ok {
		t.Fatal("schema type is not tftypes.Object")
	}
	end, ok := vxc.AttributeTypes["a_end"].(tftypes.Object)
	if !ok {
		t.Fatal("a_end type is not tftypes.Object")
	}
	partner, ok := vxc.AttributeTypes["b_end_partner_config"].(tftypes.Object)
	if !ok {
		t.Fatal("b_end_partner_config type is not tftypes.Object")
	}
	aws, ok := partner.AttributeTypes["aws_config"].(tftypes.Object)
	if !ok {
		t.Fatal("aws_config type is not tftypes.Object")
	}
	return vxcPlanFixture{vxc: vxc, end: end, partner: partner, aws: aws}
}

// objectWith builds an object of objType with the given attributes set and
// every other attribute null.
func objectWith(objType tftypes.Object, set map[string]tftypes.Value) tftypes.Value {
	attrs := nullValueMap(objType)
	for name, value := range set {
		attrs[name] = value
	}
	return tftypes.NewValue(objType, attrs)
}

// awsHostedConnection returns an AWS Hosted Connection B-End partner config.
func (fx vxcPlanFixture) awsHostedConnection() tftypes.Value {
	return objectWith(fx.partner, map[string]tftypes.Value{
		"partner": tftypes.NewValue(tftypes.String, "aws"),
		"aws_config": objectWith(fx.aws, map[string]tftypes.Value{
			"connect_type":  tftypes.NewValue(tftypes.String, "AWSHC"),
			"type":          tftypes.NewValue(tftypes.String, "private"),
			"owner_account": tftypes.NewValue(tftypes.String, "123456789012"),
		}),
	})
}

// priorEnd returns an end as state records it after a read.
func (fx vxcPlanFixture) priorEnd(uid, name, location string, locationID int) tftypes.Value {
	return objectWith(fx.end, map[string]tftypes.Value{
		"owner_uid":             tftypes.NewValue(tftypes.String, "owner-uid-1"),
		"requested_product_uid": tftypes.NewValue(tftypes.String, uid),
		"current_product_uid":   tftypes.NewValue(tftypes.String, uid),
		"product_name":          tftypes.NewValue(tftypes.String, name),
		"location":              tftypes.NewValue(tftypes.String, location),
		"location_id":           tftypes.NewValue(tftypes.Number, locationID),
		"ordered_vlan":          tftypes.NewValue(tftypes.Number, 200),
		"vlan":                  tftypes.NewValue(tftypes.Number, 200),
		"inner_vlan":            tftypes.NewValue(tftypes.Number, 0),
		"vnic_index":            tftypes.NewValue(tftypes.Number, 0),
		"secondary_name":        tftypes.NewValue(tftypes.String, "secondary"),
	})
}

// configEnd returns an end as a configuration writes it.
func (fx vxcPlanFixture) configEnd(uid string) tftypes.Value {
	return objectWith(fx.end, map[string]tftypes.Value{
		"requested_product_uid": tftypes.NewValue(tftypes.String, uid),
		"ordered_vlan":          tftypes.NewValue(tftypes.Number, 200),
	})
}

// priorVXC returns a live VXC in state. bEndPartnerConfig is null for a
// VXC between two ports.
func (fx vxcPlanFixture) priorVXC(bEnd, bEndPartnerConfig tftypes.Value) tftypes.Value {
	return objectWith(fx.vxc, map[string]tftypes.Value{
		"product_uid":          tftypes.NewValue(tftypes.String, "vxc-uid-1"),
		"product_name":         tftypes.NewValue(tftypes.String, "vxc-one"),
		"rate_limit":           tftypes.NewValue(tftypes.Number, 200),
		"contract_term_months": tftypes.NewValue(tftypes.Number, 12),
		"provisioning_status":  tftypes.NewValue(tftypes.String, "LIVE"),
		"last_updated":         tftypes.NewValue(tftypes.String, "Friday, 02-Oct-26 18:42:17 UTC"),
		"contract_end_date":    tftypes.NewValue(tftypes.String, "2027-05-24"),
		"contract_start_date":  tftypes.NewValue(tftypes.String, "2026-05-24"),
		"create_date":          tftypes.NewValue(tftypes.String, "2026-05-22"),
		"live_date":            tftypes.NewValue(tftypes.String, "2026-05-24"),
		"a_end":                fx.priorEnd("port-uid-1", "port-one", "Location One", 530),
		"b_end":                bEnd,
		"b_end_partner_config": bEndPartnerConfig,
	})
}

// configVXC returns a configuration for the VXC that priorVXC records.
func (fx vxcPlanFixture) configVXC(rateLimit int, aEnd, bEnd, bEndPartnerConfig tftypes.Value) tftypes.Value {
	return objectWith(fx.vxc, map[string]tftypes.Value{
		"product_name":         tftypes.NewValue(tftypes.String, "vxc-one"),
		"rate_limit":           tftypes.NewValue(tftypes.Number, rateLimit),
		"contract_term_months": tftypes.NewValue(tftypes.Number, 12),
		"a_end":                aEnd,
		"b_end":                bEnd,
		"b_end_partner_config": bEndPartnerConfig,
	})
}

// proposedNewState builds what Terraform core proposes for a plan: the config
// where it sets a value and prior state where it does not. The fixtures hold
// prior values only in Computed attributes, where core does the same.
func proposedNewState(prior, config tftypes.Value) tftypes.Value {
	if config.IsNull() {
		return prior
	}
	objType, ok := config.Type().(tftypes.Object)
	if !ok || prior.IsNull() {
		return config
	}
	var priorAttrs, configAttrs map[string]tftypes.Value
	_ = prior.As(&priorAttrs)
	_ = config.As(&configAttrs)
	proposed := make(map[string]tftypes.Value, len(objType.AttributeTypes))
	for name := range objType.AttributeTypes {
		proposed[name] = proposedNewState(priorAttrs[name], configAttrs[name])
	}
	return tftypes.NewValue(objType, proposed)
}

// planVXC runs a megaport_vxc plan through the provider server, which marks
// the Computed unknowns before ModifyPlan runs, the same as a terraform plan.
func planVXC(t *testing.T, fx vxcPlanFixture, prior, proposed, config tftypes.Value) tftypes.Value {
	t.Helper()

	server, err := providerserver.NewProtocol6WithError(New("test")())()
	if err != nil {
		t.Fatal(err)
	}
	dynamic := func(value tftypes.Value) *tfprotov6.DynamicValue {
		dv, err := tfprotov6.NewDynamicValue(fx.vxc, value)
		if err != nil {
			t.Fatal(err)
		}
		return &dv
	}

	resp, err := server.PlanResourceChange(context.Background(), &tfprotov6.PlanResourceChangeRequest{
		TypeName:         "megaport_vxc",
		PriorState:       dynamic(prior),
		ProposedNewState: dynamic(proposed),
		Config:           dynamic(config),
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range resp.Diagnostics {
		if d.Severity == tfprotov6.DiagnosticSeverityError {
			t.Fatalf("expected no errors, got: %s: %s", d.Summary, d.Detail)
		}
	}
	planned, err := resp.PlannedState.Unmarshal(fx.vxc)
	if err != nil {
		t.Fatal(err)
	}
	return planned
}

// plannedAt returns the planned value at a path of attribute names.
func plannedAt(t *testing.T, planned tftypes.Value, names ...string) tftypes.Value {
	t.Helper()

	path := tftypes.NewAttributePath()
	for _, name := range names {
		path = path.WithAttributeName(name)
	}
	found, _, err := tftypes.WalkAttributePath(planned, path)
	if err != nil {
		t.Fatalf("could not read %s from the plan: %v", path, err)
	}
	value, ok := found.(tftypes.Value)
	if !ok {
		t.Fatalf("%s is not a tftypes.Value", path)
	}
	return value
}

// TestVXCModifyPlan_ConvergesCloudPortDrift covers the reported bug. A config
// that names a different cloud partner port than state has to plan no changes.
// The cloud-end pin hides the port, and every unknown the framework marked for
// it has to go back to prior state.
func TestVXCModifyPlan_ConvergesCloudPortDrift(t *testing.T) {
	t.Parallel()
	fx := newVXCPlanFixture(t)
	partnerConfig := fx.awsHostedConnection()
	prior := fx.priorVXC(fx.priorEnd("aws-port-old", "US East (Ohio) (us-east-2)", "Location Two", 69), partnerConfig)

	tests := []struct {
		name    string
		bEndUID string
	}{
		{"config matches state", "aws-port-old"},
		{"config names a different AWS port", "aws-port-new"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			config := fx.configVXC(200, fx.configEnd("port-uid-1"), fx.configEnd(tc.bEndUID), partnerConfig)
			planned := planVXC(t, fx, prior, proposedNewState(prior, config), config)

			if !planned.Equal(prior) {
				diffs, _ := prior.Diff(planned)
				t.Errorf("expected the plan to converge to prior state, got %d differences: %v", len(diffs), diffs)
			}
		})
	}
}

// TestVXCModifyPlan_KeepsUnknownsOnRealChange guards the other half. Update
// writes a fresh last_updated and reads the rest back from the API, so a plan
// that carries a real change has to leave the unknowns alone.
func TestVXCModifyPlan_KeepsUnknownsOnRealChange(t *testing.T) {
	t.Parallel()
	fx := newVXCPlanFixture(t)
	partnerConfig := fx.awsHostedConnection()
	portBEnd := fx.priorEnd("port-uid-2", "port-two", "Location Two", 69)
	awsBEnd := fx.priorEnd("aws-port-old", "US East (Ohio) (us-east-2)", "Location Two", 69)
	null := tftypes.NewValue(fx.partner, nil)

	tests := []struct {
		name      string
		prior     tftypes.Value
		config    tftypes.Value
		wantKnown map[string]tftypes.Value
	}{
		{
			name:   "rate limit changed with a different AWS port",
			prior:  fx.priorVXC(awsBEnd, partnerConfig),
			config: fx.configVXC(500, fx.configEnd("port-uid-1"), fx.configEnd("aws-port-new"), partnerConfig),
			wantKnown: map[string]tftypes.Value{
				"rate_limit": tftypes.NewValue(tftypes.Number, 500),
			},
		},
		{
			name:   "A-End moved to another port",
			prior:  fx.priorVXC(portBEnd, null),
			config: fx.configVXC(200, fx.configEnd("port-uid-3"), fx.configEnd("port-uid-2"), null),
			wantKnown: map[string]tftypes.Value{
				"a_end.requested_product_uid": tftypes.NewValue(tftypes.String, "port-uid-3"),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			planned := planVXC(t, fx, tc.prior, proposedNewState(tc.prior, tc.config), tc.config)

			for _, name := range []string{"last_updated", "provisioning_status"} {
				if plannedAt(t, planned, name).IsKnown() {
					t.Errorf("expected %s to stay unknown, got: %v", name, plannedAt(t, planned, name))
				}
			}
			for name, want := range tc.wantKnown {
				got := plannedAt(t, planned, strings.Split(name, ".")...)
				if !got.Equal(want) {
					t.Errorf("expected %s to be %v, got: %v", name, want, got)
				}
			}
		})
	}
}

// TestVXCModifyPlan_PlansCreateAndDestroy checks the two walks that have no
// prior state to restore from.
func TestVXCModifyPlan_PlansCreateAndDestroy(t *testing.T) {
	t.Parallel()
	fx := newVXCPlanFixture(t)
	partnerConfig := fx.awsHostedConnection()
	config := fx.configVXC(200, fx.configEnd("port-uid-1"), fx.configEnd("aws-port-new"), partnerConfig)
	prior := fx.priorVXC(fx.priorEnd("aws-port-old", "US East (Ohio) (us-east-2)", "Location Two", 69), partnerConfig)
	nullVXC := tftypes.NewValue(fx.vxc, nil)

	t.Run("create", func(t *testing.T) {
		t.Parallel()

		planned := planVXC(t, fx, nullVXC, proposedNewState(nullVXC, config), config)

		for _, name := range []string{"product_uid", "last_updated", "provisioning_status"} {
			if plannedAt(t, planned, name).IsKnown() {
				t.Errorf("expected %s to be unknown on create, got: %v", name, plannedAt(t, planned, name))
			}
		}
	})

	t.Run("destroy", func(t *testing.T) {
		t.Parallel()

		planned := planVXC(t, fx, prior, nullVXC, nullVXC)

		if !planned.IsNull() {
			t.Errorf("expected a null plan on destroy, got: %v", planned)
		}
	})
}
