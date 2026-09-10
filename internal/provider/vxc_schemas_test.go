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

				if bfd.DeprecationMessage == "" {
					t.Error("expected bfd to carry a DeprecationMessage")
				}
				for _, want := range []string{"bgp_connections[].bfd_enabled", "300 ms", "multiplier of 3"} {
					if !strings.Contains(bfd.DeprecationMessage, want) {
						t.Errorf("expected DeprecationMessage to mention %q, got %q", want, bfd.DeprecationMessage)
					}
					if !strings.Contains(bfd.Description, want) {
						t.Errorf("expected Description to mention %q, got %q", want, bfd.Description)
					}
				}

				// Deprecating warns; it must not start rejecting configs that
				// already set the block.
				if !bfd.Optional {
					t.Error("expected bfd to stay optional")
				}
				if bfd.Required {
					t.Error("expected bfd to stay not-required")
				}
			})
		}
	}
}
