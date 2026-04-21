package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccMegaportIX_Basic tests the basic lifecycle of an IX resource
func TestAccMegaportIX_Basic(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locationID, _ := findPortTestLocation(t, 1000)
	ixName := RandomTestName()
	portName := RandomTestName()
	ixNameUpdated := ixName + "-updated"
	ixRateLimit := 500
	ixRateLimitUpdated := 750
	ixVLAN := 2001
	ixVLANUpdated := 2002

	configInitial := fmt.Sprintf(`
resource "megaport_port" "test_port" {
    product_name           = "%s"
    location_id            = %d
    port_speed             = 1000
    marketplace_visibility = false
    contract_term_months   = 1
}

resource "megaport_ix" "test_ix" {
    product_name        = "%s"
    requested_product_uid = megaport_port.test_port.product_uid
    network_service_type = "Sydney IX"
    asn                 = 12345
    mac_address         = "00:CA:FE:BA:BE:01"
    rate_limit          = %d
    vlan                = %d
    shutdown            = false
}
`, portName, locationID, ixName, ixRateLimit, ixVLAN)

	// Updated Terraform config
	configUpdated := fmt.Sprintf(`
resource "megaport_port" "test_port" {
    product_name           = "%s"
    location_id            = %d
    port_speed             = 1000
    marketplace_visibility = false
    contract_term_months   = 1
}

resource "megaport_ix" "test_ix" {
    product_name        = "%s"
    requested_product_uid = megaport_port.test_port.product_uid
    network_service_type = "Sydney IX"
    asn                 = 12345
    mac_address         = "00:CA:FE:BA:BE:01"
    rate_limit          = %d
    vlan                = %d
    shutdown            = false
}
`, portName, locationID, ixNameUpdated, ixRateLimitUpdated, ixVLANUpdated)

	resourceName := "megaport_ix.test_ix"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: configInitial,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "product_name", ixName),
					resource.TestCheckResourceAttr(resourceName, "rate_limit", fmt.Sprintf("%d", ixRateLimit)),
					resource.TestCheckResourceAttr(resourceName, "vlan", fmt.Sprintf("%d", ixVLAN)),
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
				ImportStateVerifyIgnore: []string{"last_updated", "requested_product_uid", "shutdown", "mac_address", "contract_start_date", "contract_end_date", "live_date", "resources", "provisioning_status"},
			},
			{
				Config: configUpdated,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "product_name", ixNameUpdated),
					resource.TestCheckResourceAttr(resourceName, "rate_limit", fmt.Sprintf("%d", ixRateLimitUpdated)),
					resource.TestCheckResourceAttr(resourceName, "vlan", fmt.Sprintf("%d", ixVLANUpdated)),
				),
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
	const initialPromo = "tf-acc-test-promo-initial"
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
    mac_address           = "00:CA:FE:BA:BE:01"
    rate_limit            = 500
    vlan                  = 2001
    shutdown              = false
    %s
}
`, portName, locationID, ixName, promoLine)
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
