package provider

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

// TestAccMegaportIX_Basic tests the basic lifecycle of an IX resource
func TestAccMegaportIX_Basic(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locationID, _ := findPortTestLocation(t, 1000)
	ixName := RandomTestName()
	portName := RandomTestName()
	// The import verify compares against the API, which lowercases the MAC and adds a trailing dot to reverse DNS.
	ixMAC := strings.ToLower(RandomMACAddress())
	ixMACUpdated := strings.ToLower(RandomMACAddress())
	ixNameUpdated := ixName + "-updated"
	ixRateLimit := 500
	ixRateLimitUpdated := 750
	ixVLAN := 2001
	ixVLANUpdated := 2002

	baseConfig := fmt.Sprintf(`
resource "megaport_port" "test_port" {
    product_name           = "%s"
    location_id            = %d
    port_speed             = 1000
    marketplace_visibility = false
    contract_term_months   = 1
}
`, portName, locationID)

	configInitial := baseConfig + fmt.Sprintf(`
resource "megaport_ix" "test_ix" {
    product_name        = "%s"
    requested_product_uid = megaport_port.test_port.product_uid
    network_service_type = "Sydney IX"
    asn                 = 12345
    mac_address         = "%s"
    rate_limit          = %d
    vlan                = %d
    shutdown            = false
    cost_centre         = "tf-acc-cc-1"
    public_graph        = true
    reverse_dns         = "ix1.tf-acc.example.com."
}
`, ixName, ixMAC, ixRateLimit, ixVLAN)

	// Updated Terraform config
	configUpdated := baseConfig + fmt.Sprintf(`
resource "megaport_ix" "test_ix" {
    product_name        = "%s"
    requested_product_uid = megaport_port.test_port.product_uid
    network_service_type = "Sydney IX"
    asn                 = 12345
    mac_address         = "%s"
    rate_limit          = %d
    vlan                = %d
    shutdown            = false
    cost_centre         = "tf-acc-cc-2"
    public_graph        = false
    reverse_dns         = "ix2.tf-acc.example.com."
}
`, ixNameUpdated, ixMACUpdated, ixRateLimitUpdated, ixVLANUpdated)

	// The forget step drops the IX from state and leaves the live service
	// alone, so the import after it is a real one.
	forgetConfig := baseConfig + `
removed {
    from = megaport_ix.test_ix
    lifecycle {
        destroy = false
    }
}
`

	resourceName := "megaport_ix.test_ix"

	// Holds the UID across the forget step.
	var ixUID string
	captureUID := func(state *terraform.State) error {
		rs, ok := state.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("%s not found in state", resourceName)
		}
		ixUID = rs.Primary.Attributes["product_uid"]
		if ixUID == "" {
			return fmt.Errorf("%s has no product_uid", resourceName)
		}
		return nil
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: configInitial,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "product_name", ixName),
					resource.TestCheckResourceAttr(resourceName, "rate_limit", fmt.Sprintf("%d", ixRateLimit)),
					resource.TestCheckResourceAttr(resourceName, "vlan", fmt.Sprintf("%d", ixVLAN)),
					resource.TestCheckResourceAttr(resourceName, "mac_address", ixMAC),
					resource.TestCheckResourceAttr(resourceName, "cost_centre", "tf-acc-cc-1"),
					resource.TestCheckResourceAttr(resourceName, "public_graph", "true"),
					resource.TestCheckResourceAttr(resourceName, "reverse_dns", "ix1.tf-acc.example.com."),
					captureUID,
				),
			},
			// ImportState testing
			{
				ResourceName:                         "megaport_ix.test_ix",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "product_uid",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resourceName := "megaport_ix.test_ix"
					var rawState map[string]string
					for _, m := range state.Modules {
						if len(m.Resources) > 0 {
							if v, ok := m.Resources[resourceName]; ok {
								rawState = v.Primary.Attributes
							}
						}
					}
					return rawState["product_uid"], nil
				},
				ImportStateVerifyIgnore: []string{"last_updated", "requested_product_uid", "contract_start_date", "contract_end_date", "live_date", "resources", "provisioning_status"},
			},
			{
				Config: configUpdated,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "product_name", ixNameUpdated),
					resource.TestCheckResourceAttr(resourceName, "rate_limit", fmt.Sprintf("%d", ixRateLimitUpdated)),
					resource.TestCheckResourceAttr(resourceName, "vlan", fmt.Sprintf("%d", ixVLANUpdated)),
					resource.TestCheckResourceAttr(resourceName, "mac_address", ixMACUpdated),
					resource.TestCheckResourceAttr(resourceName, "cost_centre", "tf-acc-cc-2"),
					resource.TestCheckResourceAttr(resourceName, "public_graph", "false"),
					resource.TestCheckResourceAttr(resourceName, "reverse_dns", "ix2.tf-acc.example.com."),
				),
			},
			{
				Config: forgetConfig,
			},
			{
				Config:                               configUpdated,
				ResourceName:                         resourceName,
				ImportState:                          true,
				ImportStatePersist:                   true,
				ImportStateVerifyIdentifierAttribute: "product_uid",
				ImportStateIdFunc: func(_ *terraform.State) (string, error) {
					if ixUID == "" {
						return "", fmt.Errorf("no IX UID captured")
					}
					return ixUID, nil
				},
			},
			// Apply after the import. The import leaves requested_product_uid
			// null, and the apply has to record it without a replace.
			{
				Config: configUpdated,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(resourceName, "requested_product_uid", "megaport_port.test_port", "product_uid"),
					resource.TestCheckResourceAttr(resourceName, "mac_address", ixMACUpdated),
					resource.TestCheckResourceAttrWith(resourceName, "product_uid", func(v string) error {
						if v != ixUID {
							return fmt.Errorf("product_uid = %q, want %q", v, ixUID)
						}
						return nil
					}),
				),
			},
		},
	})
}

// TestAccMegaportIX_Upgrade creates an IX with the latest release, then plans
// the same config with this build.
func TestAccMegaportIX_Upgrade(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locationID, _ := findPortTestLocation(t, 1000)
	ixName := RandomTestName()
	portName := RandomTestName()
	ixMAC := RandomMACAddress()

	configFor := func(asn int) string {
		return providerConfig + fmt.Sprintf(`
resource "megaport_port" "test_port" {
    product_name           = "%s"
    location_id            = %d
    port_speed             = 1000
    marketplace_visibility = false
    contract_term_months   = 1
}

resource "megaport_ix" "test_ix" {
    product_name          = "%s"
    requested_product_uid = megaport_port.test_port.product_uid
    network_service_type  = "Sydney IX"
    asn                   = %d
    mac_address           = "%s"
    rate_limit            = 500
    vlan                  = 2001
}
`, portName, locationID, ixName, asn, ixMAC)
	}

	resource.Test(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"megaport": {Source: "megaport/megaport"},
				},
				Config: configFor(12345),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   configFor(12345),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			// The API rebuilds the default reverse DNS from the new ASN.
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   configFor(12346),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectUnknownValue("megaport_ix.test_ix", tfjsonpath.New("reverse_dns"))},
				},
				Check: resource.TestMatchResourceAttr("megaport_ix.test_ix", "reverse_dns", regexp.MustCompile(`^as12346\..*\.megaport\.com\.$`)),
			},
		},
	})
}

// TestAccMegaportIX_PromoCode exercises promo_code on megaport_ix against
// the v1.8.0 ordering endpoint. State tracks the config-supplied value.
func TestAccMegaportIX_PromoCode(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locationID, _ := findPortTestLocation(t, 1000)
	ixName := RandomTestName()
	portName := RandomTestName()
	ixMAC := RandomMACAddress()
	initialPromo := testPromoCode()
	const otherPromo = "tf-acc-test-promo-other"

	configFor := func(promoLine string) string {
		return fmt.Sprintf(`
resource "megaport_port" "test_port" {
    product_name           = "%s"
    location_id            = %d
    port_speed             = 1000
    marketplace_visibility = false
    contract_term_months   = 1
}

resource "megaport_ix" "test_ix" {
    product_name          = "%s"
    requested_product_uid = megaport_port.test_port.product_uid
    network_service_type  = "Sydney IX"
    asn                   = 12345
    mac_address           = "%s"
    rate_limit            = 500
    vlan                  = 2001
    %s
}
`, portName, locationID, ixName, ixMAC, promoLine)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: configFor(fmt.Sprintf(`promo_code = "%s"`, initialPromo)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_ix.test_ix", "promo_code", initialPromo),
					resource.TestCheckResourceAttrSet("megaport_ix.test_ix", "product_uid"),
				),
			},
			{
				Config: configFor(fmt.Sprintf(`promo_code = "%s"`, otherPromo)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_ix.test_ix", "promo_code", otherPromo),
				),
			},
			{
				Config: configFor(""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("megaport_ix.test_ix", "promo_code"),
				),
			},
		},
	})
}
