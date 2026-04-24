package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccMegaportMCRIpsecAddon_Basic(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locationID, _ := findMCRTestLocation(t, 1000)
	mcrName := RandomTestName()
	costCentreName := RandomTestName()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create MCR and IPSec add-on with 10 tunnels
			{
				Config: providerConfig + testAccMCRIpsecAddonConfig(locationID, mcrName, costCentreName, 10),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr_ipsec_addon.test", "tunnel_count", "10"),
					resource.TestCheckResourceAttrSet("megaport_mcr_ipsec_addon.test", "add_on_uid"),
					resource.TestCheckResourceAttrSet("megaport_mcr_ipsec_addon.test", "mcr_id"),
				),
			},
			// Plan-only check — no drift
			{
				Config:             providerConfig + testAccMCRIpsecAddonConfig(locationID, mcrName, costCentreName, 10),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// Update to 20 tunnels
			{
				Config: providerConfig + testAccMCRIpsecAddonConfig(locationID, mcrName, costCentreName, 20),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr_ipsec_addon.test", "tunnel_count", "20"),
					resource.TestCheckResourceAttrSet("megaport_mcr_ipsec_addon.test", "add_on_uid"),
				),
			},
			// Import
			{
				ResourceName:                         "megaport_mcr_ipsec_addon.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "add_on_uid",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["megaport_mcr_ipsec_addon.test"]
					if !ok {
						return "", fmt.Errorf("resource not found: megaport_mcr_ipsec_addon.test")
					}
					return rs.Primary.Attributes["mcr_id"] + ":" + rs.Primary.Attributes["add_on_uid"], nil
				},
			},
		},
	})
}

func TestParseImportIDStrings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		wantLeft  string
		wantRight string
		wantErr   bool
	}{
		{
			name:      "valid",
			input:     "mcr-uid:addon-uid",
			wantLeft:  "mcr-uid",
			wantRight: "addon-uid",
		},
		{
			name:    "missing colon",
			input:   "no-colon-here",
			wantErr: true,
		},
		{
			name:    "empty left",
			input:   ":addon-uid",
			wantErr: true,
		},
		{
			name:    "empty right",
			input:   "mcr-uid:",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "multiple colons",
			input:   "a:b:c",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			left, right, err := parseImportIDStrings(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for input %q, got nil", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for input %q: %v", tc.input, err)
			}
			if left != tc.wantLeft || right != tc.wantRight {
				t.Fatalf("parseImportIDStrings(%q) = (%q, %q), want (%q, %q)", tc.input, left, right, tc.wantLeft, tc.wantRight)
			}
		})
	}
}

func testAccMCRIpsecAddonConfig(locationID int, mcrName, costCentreName string, tunnelCount int) string {
	return fmt.Sprintf(`
data "megaport_location" "test_location" {
	id = %d
}

resource "megaport_mcr" "mcr" {
	product_name         = "%s"
	port_speed           = 1000
	location_id          = data.megaport_location.test_location.id
	contract_term_months = 1
	cost_centre          = "%s"

	prefix_filter_lists = []

	lifecycle {
		ignore_changes = [prefix_filter_lists]
	}
}

resource "megaport_mcr_ipsec_addon" "test" {
	mcr_id       = megaport_mcr.mcr.product_uid
	tunnel_count = %d
}
`, locationID, mcrName, costCentreName, tunnelCount)
}
