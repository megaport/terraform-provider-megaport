package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func singleNestedAttr(t *testing.T, attrs map[string]schema.Attribute, name string) schema.SingleNestedAttribute {
	t.Helper()
	attr, ok := attrs[name]
	if !ok {
		t.Fatalf("expected a %q attribute", name)
	}
	nested, ok := attr.(schema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("expected %q to be a SingleNestedAttribute, got %T", name, attr)
	}
	return nested
}

func listNestedAttr(t *testing.T, attrs map[string]schema.Attribute, name string) schema.ListNestedAttribute {
	t.Helper()
	attr, ok := attrs[name]
	if !ok {
		t.Fatalf("expected a %q attribute", name)
	}
	nested, ok := attr.(schema.ListNestedAttribute)
	if !ok {
		t.Fatalf("expected %q to be a ListNestedAttribute, got %T", name, attr)
	}
	return nested
}

// Every deprecated bfd attribute has to warn, and none of them may start
// rejecting a config that already sets it.
func assertBFDDeprecated(t *testing.T, name string, attr schema.Attribute) {
	t.Helper()
	if attr.GetDeprecationMessage() == "" {
		t.Errorf("expected %s to carry a DeprecationMessage", name)
	}
	for _, want := range []string{"bgp_connections[].bfd_enabled", "300 ms", "multiplier of 3"} {
		if !strings.Contains(attr.GetDeprecationMessage(), want) {
			t.Errorf("expected %s DeprecationMessage to mention %q, got %q", name, want, attr.GetDeprecationMessage())
		}
		if !strings.Contains(attr.GetDescription(), want) {
			t.Errorf("expected %s Description to mention %q, got %q", name, want, attr.GetDescription())
		}
	}
	if !attr.IsOptional() {
		t.Errorf("expected %s to stay optional", name)
	}
	if attr.IsRequired() {
		t.Errorf("expected %s to stay not-required", name)
	}
}

// The bfd timers are inert. Both partner config shapes, on both ends, must
// warn and point at the switch that does work.
func TestVXCSchema_InterfaceBFDIsDeprecated(t *testing.T) {
	resp := &resource.SchemaResponse{}
	(&vxcResource{}).Schema(context.Background(), resource.SchemaRequest{}, resp)

	for _, end := range []string{"a_end_partner_config", "b_end_partner_config"} {
		for _, partnerConfig := range []string{"vrouter_config", "partner_a_end_config"} {
			t.Run(end+"."+partnerConfig, func(t *testing.T) {
				config := singleNestedAttr(t, singleNestedAttr(t, resp.Schema.Attributes, end).Attributes, partnerConfig)
				interfaces := listNestedAttr(t, config.Attributes, "interfaces")
				bfd := singleNestedAttr(t, interfaces.NestedObject.Attributes, "bfd")

				assertBFDDeprecated(t, "bfd", bfd)

				// The timers carry the warning too. A reader who deep-links to
				// the nested docs anchor never sees the parent block.
				for _, timer := range []string{"tx_interval", "rx_interval", "multiplier"} {
					child, ok := bfd.Attributes[timer]
					if !ok {
						t.Fatalf("expected a %q attribute inside bfd", timer)
					}
					assertBFDDeprecated(t, "bfd."+timer, child)
				}
			})
		}
	}
}

// Terraform prints a key in plan and show output unless the schema marks it Sensitive.
func TestVXCSchema_CSPKeysSensitive(t *testing.T) {
	resp := &resource.SchemaResponse{}
	(&vxcResource{}).Schema(context.Background(), resource.SchemaRequest{}, resp)

	for _, end := range []string{"a_end_partner_config", "b_end_partner_config"} {
		google := singleNestedAttr(t, singleNestedAttr(t, resp.Schema.Attributes, end).Attributes, "google_config")
		pairingKey, ok := google.Attributes["pairing_key"]
		if !ok {
			t.Fatalf("expected a pairing_key attribute inside %s.google_config", end)
		}
		if !pairingKey.IsSensitive() {
			t.Errorf("expected %s.google_config.pairing_key to be sensitive", end)
		}
	}

	conns := listNestedAttr(t, resp.Schema.Attributes, "csp_connections")
	for _, name := range []string{"auth_key", "pairing_key"} {
		attr, ok := conns.NestedObject.Attributes[name]
		if !ok {
			t.Fatalf("expected a %q attribute inside csp_connections", name)
		}
		if !attr.IsSensitive() {
			t.Errorf("expected csp_connections.%s to be sensitive", name)
		}
	}
}
