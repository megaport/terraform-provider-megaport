package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccMegaportMCRPrefixFilterList_Basic(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locationID, _ := findMCRTestLocation(t, 1000)
	mcrName := RandomTestName()
	prefixFilterName := RandomTestName()
	prefixFilterName2 := RandomTestName()
	prefixFilterNameNew := RandomTestName()
	prefixFilterNameNew2 := RandomTestName()
	prefixFilterNameNew3 := RandomTestName()
	prefixFilterNameNew4 := RandomTestName()
	costCentreName := RandomTestName()
	mcrNameNew := RandomTestName()
	costCentreNameNew := RandomTestName()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create MCR and prefix filter lists
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}

				resource "megaport_mcr" "mcr" {
					product_name         = "%s"
					port_speed          = 1000
					location_id         = data.megaport_location.test_location.id
					contract_term_months = 12
					cost_centre         = "%s"

					resource_tags = {
						"key1" = "value1"
						"key2" = "value2"
					}

				}

				resource "megaport_mcr_prefix_filter_list" "prefix_list_1" {
					mcr_id         = megaport_mcr.mcr.product_uid
					description    = "%s"
					address_family = "IPv4"
					entries = [
						{
							action = "permit"
							prefix = "10.0.1.0/24"
							ge     = 25
							le     = 32
						},
						{
							action = "deny"
							prefix = "10.0.2.0/24"
							ge     = 25
							le     = 27
						}
					]
				}

				resource "megaport_mcr_prefix_filter_list" "prefix_list_2" {
					mcr_id         = megaport_mcr.mcr.product_uid
					description    = "%s"
					address_family = "IPv4"
					entries = [
						{
							action = "permit"
							prefix = "10.0.1.0/24"
							ge     = 26
							le     = 32
						},
						{
							action = "deny"
							prefix = "10.0.2.0/24"
							ge     = 24
							le     = 25
						}
					]
				}
				`, locationID, mcrName, costCentreName, prefixFilterName, prefixFilterName2),
				Check: resource.ComposeAggregateTestCheckFunc(
					// MCR checks
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "product_name", mcrName),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "cost_centre", costCentreName),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "resource_tags.key1", "value1"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "resource_tags.key2", "value2"),
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "product_uid"),

					// Prefix filter list 1 checks
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_1", "description", prefixFilterName),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_1", "address_family", "IPv4"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_1", "entries.#", "2"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_1", "entries.0.action", "permit"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_1", "entries.0.prefix", "10.0.1.0/24"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_1", "entries.0.ge", "25"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_1", "entries.0.le", "32"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_1", "entries.1.action", "deny"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_1", "entries.1.prefix", "10.0.2.0/24"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_1", "entries.1.ge", "25"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_1", "entries.1.le", "27"),
					resource.TestCheckResourceAttrSet("megaport_mcr_prefix_filter_list.prefix_list_1", "id"),

					// Prefix filter list 2 checks
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_2", "description", prefixFilterName2),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_2", "address_family", "IPv4"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_2", "entries.#", "2"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_2", "entries.0.action", "permit"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_2", "entries.0.prefix", "10.0.1.0/24"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_2", "entries.0.ge", "26"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_2", "entries.0.le", "32"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_2", "entries.1.action", "deny"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_2", "entries.1.prefix", "10.0.2.0/24"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_2", "entries.1.ge", "24"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_2", "entries.1.le", "25"),
					resource.TestCheckResourceAttrSet("megaport_mcr_prefix_filter_list.prefix_list_2", "id"),
				),
			},
			// Test ImportState for prefix filter list 1
			{
				ResourceName:      "megaport_mcr_prefix_filter_list.prefix_list_1",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resourceName1 := "megaport_mcr.mcr"
					resourceName2 := "megaport_mcr_prefix_filter_list.prefix_list_1"
					var mcrUID, prefixListID string

					for _, m := range state.Modules {
						if len(m.Resources) > 0 {
							if v, ok := m.Resources[resourceName1]; ok {
								mcrUID = v.Primary.Attributes["product_uid"]
							}
							if v, ok := m.Resources[resourceName2]; ok {
								prefixListID = v.Primary.Attributes["id"]
							}
						}
					}
					return fmt.Sprintf("%s:%s", mcrUID, prefixListID), nil
				},
				ImportStateVerifyIgnore: []string{},
			},
			// Update Test 1: Modify existing prefix filter lists and add a new one
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}
				
				resource "megaport_mcr" "mcr" {
					product_name         = "%s"
					port_speed          = 1000
					location_id         = data.megaport_location.test_location.id
					contract_term_months = 12
					cost_centre         = "%s"

					resource_tags = {
						"key1updated" = "value1updated"
						"key2updated" = "value2updated"
					}

				}

				resource "megaport_mcr_prefix_filter_list" "prefix_list_1" {
					mcr_id         = megaport_mcr.mcr.product_uid
					description    = "%s"
					address_family = "IPv4"
					entries = [
						{
							action = "permit"
							prefix = "10.0.1.0/24"
							ge     = 24
							le     = 32
						},
						{
							action = "deny"
							prefix = "10.0.2.0/24"
							ge     = 25
							le     = 29
						}
					]
				}

				resource "megaport_mcr_prefix_filter_list" "prefix_list_2" {
					mcr_id         = megaport_mcr.mcr.product_uid
					description    = "%s"
					address_family = "IPv4"
					entries = [
						{
							action = "permit"
							prefix = "10.0.1.0/24"
							ge     = 25
							le     = 32
						},
						{
							action = "deny"
							prefix = "10.0.2.0/24"
							ge     = 24
							le     = 26
						}
					]
				}

				resource "megaport_mcr_prefix_filter_list" "prefix_list_3" {
					mcr_id         = megaport_mcr.mcr.product_uid
					description    = "%s"
					address_family = "IPv4"
					entries = [
						{
							action = "permit"
							prefix = "10.0.1.0/24"
							ge     = 24
							le     = 30
						},
						{
							action = "deny"
							prefix = "10.0.2.0/24"
							ge     = 27
							le     = 32
						}
					]
				}
				`, locationID, mcrName, costCentreName, prefixFilterNameNew, prefixFilterNameNew2, prefixFilterNameNew3),
				Check: resource.ComposeAggregateTestCheckFunc(
					// MCR checks
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "product_name", mcrName),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "resource_tags.key1updated", "value1updated"),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "resource_tags.key2updated", "value2updated"),

					// Updated prefix filter list 1 checks
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_1", "description", prefixFilterNameNew),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_1", "entries.0.ge", "24"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_1", "entries.0.le", "32"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_1", "entries.1.ge", "25"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_1", "entries.1.le", "29"),

					// Updated prefix filter list 2 checks
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_2", "description", prefixFilterNameNew2),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_2", "entries.0.ge", "25"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_2", "entries.1.ge", "24"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_2", "entries.1.le", "26"),

					// New prefix filter list 3 checks
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_3", "description", prefixFilterNameNew3),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_3", "entries.0.ge", "24"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_3", "entries.0.le", "30"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_3", "entries.1.ge", "27"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_3", "entries.1.le", "32"),
					resource.TestCheckResourceAttrSet("megaport_mcr_prefix_filter_list.prefix_list_3", "id"),
				),
			},
			// Update Test 2: Reduce to single prefix filter list
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}
				
				resource "megaport_mcr" "mcr" {
					product_name         = "%s"
					port_speed          = 1000
					location_id         = data.megaport_location.test_location.id
					contract_term_months = 12
					cost_centre         = "%s"

					resource_tags = {
						"key1updated" = "value1updated"
						"key2updated" = "value2updated"
					}

				}

				resource "megaport_mcr_prefix_filter_list" "prefix_list_single" {
					mcr_id         = megaport_mcr.mcr.product_uid
					description    = "%s"
					address_family = "IPv4"
					entries = [
						{
							action = "permit"
							prefix = "10.0.1.0/24"
							ge     = 28
							le     = 32
						}
					]
				}
				`, locationID, mcrNameNew, costCentreNameNew, prefixFilterNameNew4),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "product_name", mcrNameNew),
					resource.TestCheckResourceAttr("megaport_mcr.mcr", "cost_centre", costCentreNameNew),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_single", "description", prefixFilterNameNew4),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_single", "entries.#", "1"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_single", "entries.0.action", "permit"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_single", "entries.0.prefix", "10.0.1.0/24"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_single", "entries.0.ge", "28"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.prefix_list_single", "entries.0.le", "32"),
				),
			},
		},
	})
}

func TestAccMegaportMCRPrefixFilterList_IPv6(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locationID, _ := findMCRTestLocation(t, 1000)
	mcrName := RandomTestName()
	prefixFilterName := RandomTestName()
	costCentreName := RandomTestName()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}
				
				resource "megaport_mcr" "mcr" {
					product_name         = "%s"
					port_speed          = 1000
					location_id         = data.megaport_location.test_location.id
					contract_term_months = 12
					cost_centre         = "%s"

				}

				resource "megaport_mcr_prefix_filter_list" "ipv6_list" {
					mcr_id         = megaport_mcr.mcr.product_uid
					description    = "%s"
					address_family = "IPv6"
					entries = [
						{
							action = "permit"
							prefix = "2001:db8::/32"
							ge     = 48
							le     = 64
						},
						{
							action = "deny"
							prefix = "2001:db8:1::/48"
							ge     = 56
							le     = 128
						}
					]
				}
				`, locationID, mcrName, costCentreName, prefixFilterName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.ipv6_list", "description", prefixFilterName),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.ipv6_list", "address_family", "IPv6"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.ipv6_list", "entries.#", "2"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.ipv6_list", "entries.0.action", "permit"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.ipv6_list", "entries.0.prefix", "2001:db8::/32"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.ipv6_list", "entries.0.ge", "48"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.ipv6_list", "entries.0.le", "64"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.ipv6_list", "entries.1.action", "deny"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.ipv6_list", "entries.1.prefix", "2001:db8:1::/48"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.ipv6_list", "entries.1.ge", "56"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.ipv6_list", "entries.1.le", "128"),
					resource.TestCheckResourceAttrSet("megaport_mcr_prefix_filter_list.ipv6_list", "id"),
				),
			},
		},
	})
}

// TestAccMegaportMCRPrefixFilterList_ExactMatch tests the exact match prefix filter entries
// This specifically tests the normalization fix for when the Megaport API returns le=32 (IPv4)
// or le=128 (IPv6) instead of the exact match value configured by the user.
// See PR #308 for details on the bug fix.
func TestAccMegaportMCRPrefixFilterList_ExactMatch(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locationID, _ := findMCRTestLocation(t, 1000)
	mcrName := RandomTestName()
	prefixFilterNameIPv4 := RandomTestName()
	prefixFilterNameIPv6 := RandomTestName()
	costCentreName := RandomTestName()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create prefix filter lists with exact match entries
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}
				
				resource "megaport_mcr" "mcr" {
					product_name         = "%s"
					port_speed          = 1000
					location_id         = data.megaport_location.test_location.id
					contract_term_months = 12
					cost_centre         = "%s"

				}

				# IPv4 Exact Match Test - ge=le should not cause drift
				resource "megaport_mcr_prefix_filter_list" "ipv4_exact" {
					mcr_id         = megaport_mcr.mcr.product_uid
					description    = "%s"
					address_family = "IPv4"
					entries = [
						{
							action = "permit"
							prefix = "10.0.0.0/24"
							ge     = 24
							le     = 24  # Exact match - should remain 24, not change to 32
						},
						{
							action = "deny"
							prefix = "192.168.0.0/16"
							ge     = 16
							le     = 16  # Exact match - should remain 16, not change to 32
						},
						{
							action = "permit"
							prefix = "172.16.0.0/12"
							ge     = 20
							le     = 20  # Exact match for /20 within /12 - should remain 20
						}
					]
				}

				# IPv6 Exact Match Test - ge=le should not cause drift
				resource "megaport_mcr_prefix_filter_list" "ipv6_exact" {
					mcr_id         = megaport_mcr.mcr.product_uid
					description    = "%s"
					address_family = "IPv6"
					entries = [
						{
							action = "permit"
							prefix = "2001:db8::/32"
							ge     = 48
							le     = 48  # Exact match - should remain 48, not change to 128
						},
						{
							action = "deny"
							prefix = "2001:db8:1::/48"
							ge     = 64
							le     = 64  # Exact match - should remain 64, not change to 128
						}
					]
				}
				`, locationID, mcrName, costCentreName, prefixFilterNameIPv4, prefixFilterNameIPv6),
				Check: resource.ComposeAggregateTestCheckFunc(
					// IPv4 Exact Match Checks - verify ge=le is preserved
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.ipv4_exact", "description", prefixFilterNameIPv4),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.ipv4_exact", "address_family", "IPv4"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.ipv4_exact", "entries.#", "3"),

					// Entry 0: /24 exact match
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.ipv4_exact", "entries.0.action", "permit"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.ipv4_exact", "entries.0.prefix", "10.0.0.0/24"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.ipv4_exact", "entries.0.ge", "24"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.ipv4_exact", "entries.0.le", "24"), // Must stay 24, not become 32

					// Entry 1: /16 exact match
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.ipv4_exact", "entries.1.action", "deny"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.ipv4_exact", "entries.1.prefix", "192.168.0.0/16"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.ipv4_exact", "entries.1.ge", "16"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.ipv4_exact", "entries.1.le", "16"), // Must stay 16, not become 32

					// Entry 2: /20 exact match within /12
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.ipv4_exact", "entries.2.action", "permit"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.ipv4_exact", "entries.2.prefix", "172.16.0.0/12"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.ipv4_exact", "entries.2.ge", "20"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.ipv4_exact", "entries.2.le", "20"), // Must stay 20, not become 32

					// IPv6 Exact Match Checks - verify ge=le is preserved
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.ipv6_exact", "description", prefixFilterNameIPv6),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.ipv6_exact", "address_family", "IPv6"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.ipv6_exact", "entries.#", "2"),

					// Entry 0: /48 exact match
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.ipv6_exact", "entries.0.action", "permit"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.ipv6_exact", "entries.0.prefix", "2001:db8::/32"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.ipv6_exact", "entries.0.ge", "48"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.ipv6_exact", "entries.0.le", "48"), // Must stay 48, not become 128

					// Entry 1: /64 exact match
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.ipv6_exact", "entries.1.action", "deny"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.ipv6_exact", "entries.1.prefix", "2001:db8:1::/48"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.ipv6_exact", "entries.1.ge", "64"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.ipv6_exact", "entries.1.le", "64"), // Must stay 64, not become 128
				),
			},
			// Step 2: Run plan again to ensure no drift is detected (idempotency check)
			// This is the critical test - if normalization doesn't work, this step will fail
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}
				
				resource "megaport_mcr" "mcr" {
					product_name         = "%s"
					port_speed          = 1000
					location_id         = data.megaport_location.test_location.id
					contract_term_months = 12
					cost_centre         = "%s"

				}

				# IPv4 Exact Match Test - ge=le should not cause drift
				resource "megaport_mcr_prefix_filter_list" "ipv4_exact" {
					mcr_id         = megaport_mcr.mcr.product_uid
					description    = "%s"
					address_family = "IPv4"
					entries = [
						{
							action = "permit"
							prefix = "10.0.0.0/24"
							ge     = 24
							le     = 24  # Exact match - should remain 24, not change to 32
						},
						{
							action = "deny"
							prefix = "192.168.0.0/16"
							ge     = 16
							le     = 16  # Exact match - should remain 16, not change to 32
						},
						{
							action = "permit"
							prefix = "172.16.0.0/12"
							ge     = 20
							le     = 20  # Exact match for /20 within /12 - should remain 20
						}
					]
				}

				# IPv6 Exact Match Test - ge=le should not cause drift
				resource "megaport_mcr_prefix_filter_list" "ipv6_exact" {
					mcr_id         = megaport_mcr.mcr.product_uid
					description    = "%s"
					address_family = "IPv6"
					entries = [
						{
							action = "permit"
							prefix = "2001:db8::/32"
							ge     = 48
							le     = 48  # Exact match - should remain 48, not change to 128
						},
						{
							action = "deny"
							prefix = "2001:db8:1::/48"
							ge     = 64
							le     = 64  # Exact match - should remain 64, not change to 128
						}
					]
				}
				`, locationID, mcrName, costCentreName, prefixFilterNameIPv4, prefixFilterNameIPv6),
				// PlanOnly checks that no changes are needed - validates idempotency
				PlanOnly: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Re-verify the values are still correct
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.ipv4_exact", "entries.0.le", "24"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.ipv4_exact", "entries.1.le", "16"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.ipv4_exact", "entries.2.le", "20"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.ipv6_exact", "entries.0.le", "48"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.ipv6_exact", "entries.1.le", "64"),
				),
			},
			// Step 3: Test import of exact match prefix filter lists
			// Note: During import, we return raw API values (le=32 for IPv4).
			// This is intentional - import shows actual API state, and users can
			// adjust their HCL to match their desired configuration (exact match or range).
			// After the first apply with user's config, normalization works correctly.
			{
				ResourceName:      "megaport_mcr_prefix_filter_list.ipv4_exact",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resourceName1 := "megaport_mcr.mcr"
					resourceName2 := "megaport_mcr_prefix_filter_list.ipv4_exact"
					var mcrUID, prefixListID string

					for _, m := range state.Modules {
						if len(m.Resources) > 0 {
							if v, ok := m.Resources[resourceName1]; ok {
								mcrUID = v.Primary.Attributes["product_uid"]
							}
							if v, ok := m.Resources[resourceName2]; ok {
								prefixListID = v.Primary.Attributes["id"]
							}
						}
					}
					return fmt.Sprintf("%s:%s", mcrUID, prefixListID), nil
				},
				// Ignore 'le' fields during import verify because the API returns le=32 (max)
				// for exact match entries. During normal operation, we normalize this back to
				// the user's configured value (ge=le). But during import, we can't know the
				// user's intention, so we return raw API values.
				ImportStateVerifyIgnore: []string{"entries.0.le", "entries.1.le", "entries.2.le"},
			},
		},
	})
}

// TestAccMegaportMCRPrefixFilterList_CIDRValidation tests that prefixes with host bits set
// are rejected with a descriptive error, and that canonical prefixes work correctly.
// This is the end-to-end test for the fix in issue #317.
func TestAccMegaportMCRPrefixFilterList_CIDRValidation(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locationID, _ := findMCRTestLocation(t, 1000)
	mcrName := RandomTestName()
	prefixFilterName := RandomTestName()
	costCentreName := RandomTestName()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Non-canonical CIDR prefix should be rejected with a helpful error
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}

				resource "megaport_mcr" "mcr" {
					product_name         = "%s"
					port_speed          = 1000
					location_id         = data.megaport_location.test_location.id
					contract_term_months = 12
					cost_centre         = "%s"

				}

				resource "megaport_mcr_prefix_filter_list" "cidr_test" {
					mcr_id         = megaport_mcr.mcr.product_uid
					description    = "%s"
					address_family = "IPv4"
					entries = [
						{
							action = "permit"
							prefix = "192.168.1.100/24"
							ge     = 24
							le     = 24
						}
					]
				}
				`, locationID, mcrName, costCentreName, prefixFilterName),
				ExpectError: regexp.MustCompile(`(?s)host bits set.*Use the network address.*192\.168\.1\.0/24`),
			},
		},
	})
}

// TestAccMegaportMCRPrefixFilterList_MixedExactAndRange tests a combination of exact match
// and range-based prefix filter entries to ensure both are handled correctly.
func TestAccMegaportMCRPrefixFilterList_MixedExactAndRange(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locationID, _ := findMCRTestLocation(t, 1000)
	mcrName := RandomTestName()
	prefixFilterName := RandomTestName()
	costCentreName := RandomTestName()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "test_location" {
					id = %d
				}
				
				resource "megaport_mcr" "mcr" {
					product_name         = "%s"
					port_speed          = 1000
					location_id         = data.megaport_location.test_location.id
					contract_term_months = 12
					cost_centre         = "%s"

				}

				resource "megaport_mcr_prefix_filter_list" "mixed" {
					mcr_id         = megaport_mcr.mcr.product_uid
					description    = "%s"
					address_family = "IPv4"
					entries = [
						{
							# Exact match entry - le should stay 24
							action = "permit"
							prefix = "10.0.0.0/24"
							ge     = 24
							le     = 24
						},
						{
							# Range entry - le should stay 28 (not max)
							action = "deny"
							prefix = "192.168.0.0/16"
							ge     = 24
							le     = 28
						},
						{
							# Range to max - le=32 is intentional here (full range)
							action = "permit"
							prefix = "172.16.0.0/12"
							ge     = 16
							le     = 32
						},
						{
							# Another exact match
							action = "deny"
							prefix = "10.10.0.0/16"
							ge     = 20
							le     = 20
						}
					]
				}
				`, locationID, mcrName, costCentreName, prefixFilterName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.mixed", "description", prefixFilterName),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.mixed", "entries.#", "4"),

					// Entry 0: Exact match - must stay 24
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.mixed", "entries.0.ge", "24"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.mixed", "entries.0.le", "24"),

					// Entry 1: Range - must stay 28
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.mixed", "entries.1.ge", "24"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.mixed", "entries.1.le", "28"),

					// Entry 2: Full range to max - user explicitly configured le=32
					// With the fix, this should NOT be normalized since the plan has le=32
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.mixed", "entries.2.ge", "16"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.mixed", "entries.2.le", "32"),

					// Entry 3: Exact match - must stay 20
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.mixed", "entries.3.ge", "20"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.mixed", "entries.3.le", "20"),
				),
			},
		},
	})
}

// TestAccMegaportMCRPrefixFilterList_ImportNoVXCDrift tests that importing standalone
// MCR prefix filter list resources does not cause VXCs (with BGP connections referencing
// those prefix filter lists) to enter an update loop.
//
// This reproduces the GoTo customer issue where:
// 1. MCR has standalone prefix filter lists managed by megaport_mcr_prefix_filter_list
// 2. VXC has BGP connections that reference prefix filter lists via import_whitelist
// 3. After importing the prefix filter list, the VXC should NOT detect changes
// 4. The MCR should NOT attempt to delete the standalone-managed prefix filter lists
func TestAccMegaportMCRPrefixFilterList_ImportNoVXCDrift(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locationID, _ := findMCRTestLocation(t, 1000)
	mcrName := RandomTestName()
	portName := RandomTestName()
	vxcName := RandomTestName()
	prefixFilterListName := RandomTestName()
	costCentreName := RandomTestName()

	sharedConfig := func() string {
		return providerConfig + fmt.Sprintf(`
			data "megaport_location" "loc" {
				id = %d
			}

			resource "megaport_mcr" "mcr" {
				product_name         = "%s"
				location_id          = data.megaport_location.loc.id
				contract_term_months = 1
				port_speed           = 1000
				asn                  = 64555
				cost_centre          = "%s"

			}

			resource "megaport_mcr_prefix_filter_list" "pfl" {
				mcr_id         = megaport_mcr.mcr.product_uid
				description    = "%s"
				address_family = "IPv4"
				entries = [
					{
						action = "permit"
						prefix = "10.0.1.0/24"
						ge     = 24
						le     = 32
					},
					{
						action = "deny"
						prefix = "10.0.2.0/24"
						ge     = 25
						le     = 28
					}
				]
			}

			resource "megaport_port" "port" {
				product_name           = "%s"
				port_speed             = 1000
				location_id            = data.megaport_location.loc.id
				contract_term_months   = 1
				marketplace_visibility = false
			}

			resource "megaport_vxc" "vxc" {
				product_name         = "%s"
				rate_limit           = 500
				contract_term_months = 1

				a_end = {
					requested_product_uid = megaport_mcr.mcr.product_uid
					ordered_vlan          = 0
				}

				a_end_partner_config = {
					partner = "vrouter"
					vrouter_config = {
						interfaces = [{
							ip_addresses = ["10.0.0.1/30"]
							bgp_connections = [{
								peer_asn         = 64512
								local_ip_address = "10.0.0.1"
								peer_ip_address  = "10.0.0.2"
								password         = "testPassword123"
								shutdown         = false
								description      = "BGP with prefix filter"
								med_in           = 100
								med_out          = 100
								bfd_enabled      = false
								export_policy    = "permit"
								import_whitelist = "%s"
							}]
						}]
					}
				}

				b_end = {
					requested_product_uid = megaport_port.port.product_uid
				}

				depends_on = [megaport_mcr_prefix_filter_list.pfl]
			}
		`, locationID, mcrName, costCentreName, prefixFilterListName, portName, vxcName, prefixFilterListName)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create MCR + standalone prefix filter list + VXC with BGP referencing it
			{
				Config: sharedConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "product_uid"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.pfl", "description", prefixFilterListName),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.pfl", "entries.#", "2"),
					resource.TestCheckResourceAttrSet("megaport_vxc.vxc", "product_uid"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "product_name", vxcName),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "a_end_partner_config.partner", "vrouter"),
				),
			},
			// Step 2: Import the prefix filter list
			{
				ResourceName:      "megaport_mcr_prefix_filter_list.pfl",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					var mcrUID, prefixListID string
					for _, m := range state.Modules {
						if len(m.Resources) > 0 {
							if v, ok := m.Resources["megaport_mcr.mcr"]; ok {
								mcrUID = v.Primary.Attributes["product_uid"]
							}
							if v, ok := m.Resources["megaport_mcr_prefix_filter_list.pfl"]; ok {
								prefixListID = v.Primary.Attributes["id"]
							}
						}
					}
					return fmt.Sprintf("%s:%s", mcrUID, prefixListID), nil
				},
				ImportStateVerifyIgnore: []string{},
			},
			// Step 3: Apply the same config after import - reconcile any import differences
			{
				Config: sharedConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.pfl", "description", prefixFilterListName),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.pfl", "entries.#", "2"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "product_name", vxcName),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "a_end_partner_config.partner", "vrouter"),
				),
			},
			// Step 4: Plan-only to verify NO drift on VXC or MCR after prefix filter list import
			// This is the critical test - if the import causes VXC/MCR updates, this step fails
			{
				Config:   sharedConfig(),
				PlanOnly: true,
			},
		},
	})
}

// TestAccMegaportMCRPrefixFilterList_ImportMultipleNoVXCDrift tests that importing multiple
// standalone prefix filter lists does not trigger updates on VXCs that reference them.
// This mirrors the GoTo customer pattern of importing multiple prefix filter lists at once.
func TestAccMegaportMCRPrefixFilterList_ImportMultipleNoVXCDrift(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locationID, _ := findMCRTestLocation(t, 1000)
	mcrName := RandomTestName()
	portName := RandomTestName()
	vxcName := RandomTestName()
	pflName1 := RandomTestName()
	pflName2 := RandomTestName()
	pflName3 := RandomTestName()
	costCentreName := RandomTestName()

	sharedConfig := func() string {
		return providerConfig + fmt.Sprintf(`
			data "megaport_location" "loc" {
				id = %d
			}

			resource "megaport_mcr" "mcr" {
				product_name         = "%s"
				location_id          = data.megaport_location.loc.id
				contract_term_months = 1
				port_speed           = 1000
				asn                  = 64555
				cost_centre          = "%s"

			}

			resource "megaport_mcr_prefix_filter_list" "pfl_whitelist" {
				mcr_id         = megaport_mcr.mcr.product_uid
				description    = "%s"
				address_family = "IPv4"
				entries = [
					{
						action = "permit"
						prefix = "10.0.1.0/24"
						ge     = 24
						le     = 32
					}
				]
			}

			resource "megaport_mcr_prefix_filter_list" "pfl_blacklist" {
				mcr_id         = megaport_mcr.mcr.product_uid
				description    = "%s"
				address_family = "IPv4"
				entries = [
					{
						action = "deny"
						prefix = "10.0.2.0/24"
						ge     = 24
						le     = 32
					}
				]
			}

			resource "megaport_mcr_prefix_filter_list" "pfl_export" {
				mcr_id         = megaport_mcr.mcr.product_uid
				description    = "%s"
				address_family = "IPv4"
				entries = [
					{
						action = "permit"
						prefix = "172.16.0.0/12"
						ge     = 16
						le     = 24
					}
				]
			}

			resource "megaport_port" "port" {
				product_name           = "%s"
				port_speed             = 1000
				location_id            = data.megaport_location.loc.id
				contract_term_months   = 1
				marketplace_visibility = false
			}

			resource "megaport_vxc" "vxc" {
				product_name         = "%s"
				rate_limit           = 500
				contract_term_months = 1

				a_end = {
					requested_product_uid = megaport_mcr.mcr.product_uid
					ordered_vlan          = 0
				}

				a_end_partner_config = {
					partner = "vrouter"
					vrouter_config = {
						interfaces = [{
							ip_addresses = ["10.0.0.1/30"]
							bgp_connections = [{
								peer_asn         = 64512
								local_ip_address = "10.0.0.1"
								peer_ip_address  = "10.0.0.2"
								password         = "testPassword123"
								shutdown         = false
								description      = "BGP with multiple prefix filters"
								med_in           = 100
								med_out          = 100
								bfd_enabled      = false
								export_policy    = "deny"
								import_whitelist = "%s"
								export_whitelist = "%s"
							}]
						}]
					}
				}

				b_end = {
					requested_product_uid = megaport_port.port.product_uid
				}

				depends_on = [
					megaport_mcr_prefix_filter_list.pfl_whitelist,
					megaport_mcr_prefix_filter_list.pfl_blacklist,
					megaport_mcr_prefix_filter_list.pfl_export,
				]
			}
		`, locationID, mcrName, costCentreName,
			pflName1, pflName2, pflName3,
			portName, vxcName,
			pflName1, pflName3)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create everything
			{
				Config: sharedConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "product_uid"),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.pfl_whitelist", "description", pflName1),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.pfl_blacklist", "description", pflName2),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.pfl_export", "description", pflName3),
					resource.TestCheckResourceAttrSet("megaport_vxc.vxc", "product_uid"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "a_end_partner_config.partner", "vrouter"),
				),
			},
			// Step 2: Import prefix filter list 1 (whitelist)
			{
				ResourceName:      "megaport_mcr_prefix_filter_list.pfl_whitelist",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					var mcrUID, pflID string
					for _, m := range state.Modules {
						if len(m.Resources) > 0 {
							if v, ok := m.Resources["megaport_mcr.mcr"]; ok {
								mcrUID = v.Primary.Attributes["product_uid"]
							}
							if v, ok := m.Resources["megaport_mcr_prefix_filter_list.pfl_whitelist"]; ok {
								pflID = v.Primary.Attributes["id"]
							}
						}
					}
					return fmt.Sprintf("%s:%s", mcrUID, pflID), nil
				},
				ImportStateVerifyIgnore: []string{},
			},
			// Step 3: Import prefix filter list 2 (blacklist)
			{
				ResourceName:      "megaport_mcr_prefix_filter_list.pfl_blacklist",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					var mcrUID, pflID string
					for _, m := range state.Modules {
						if len(m.Resources) > 0 {
							if v, ok := m.Resources["megaport_mcr.mcr"]; ok {
								mcrUID = v.Primary.Attributes["product_uid"]
							}
							if v, ok := m.Resources["megaport_mcr_prefix_filter_list.pfl_blacklist"]; ok {
								pflID = v.Primary.Attributes["id"]
							}
						}
					}
					return fmt.Sprintf("%s:%s", mcrUID, pflID), nil
				},
				ImportStateVerifyIgnore: []string{},
			},
			// Step 4: Import prefix filter list 3 (export)
			{
				ResourceName:      "megaport_mcr_prefix_filter_list.pfl_export",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					var mcrUID, pflID string
					for _, m := range state.Modules {
						if len(m.Resources) > 0 {
							if v, ok := m.Resources["megaport_mcr.mcr"]; ok {
								mcrUID = v.Primary.Attributes["product_uid"]
							}
							if v, ok := m.Resources["megaport_mcr_prefix_filter_list.pfl_export"]; ok {
								pflID = v.Primary.Attributes["id"]
							}
						}
					}
					return fmt.Sprintf("%s:%s", mcrUID, pflID), nil
				},
				ImportStateVerifyIgnore: []string{},
			},
			// Step 5: Apply same config to reconcile any import state differences
			{
				Config: sharedConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.pfl_whitelist", "description", pflName1),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.pfl_blacklist", "description", pflName2),
					resource.TestCheckResourceAttr("megaport_mcr_prefix_filter_list.pfl_export", "description", pflName3),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "product_name", vxcName),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "a_end_partner_config.partner", "vrouter"),
				),
			},
			// Step 6: Plan-only to verify NO drift - VXC and MCR must not show changes
			// This validates that importing prefix filter lists doesn't trigger an update loop
			{
				Config:   sharedConfig(),
				PlanOnly: true,
			},
		},
	})
}
