package provider

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	megaport "github.com/megaport/megaportgo"
)

// getNATGatewayTestConfig queries the staging NAT Gateway sessions API to get
// a valid speed/session-count pair for acceptance testing. The validate
// endpoint rejects arbitrary combinations, so the test must use a pair the
// API advertises.
func getNATGatewayTestConfig() (speed, sessionCount int, err error) {
	client, err := getTestClient()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to create test client: %w", err)
	}

	sessions, err := client.NATGatewayService.ListNATGatewaySessions(context.Background())
	if err != nil {
		return 0, 0, fmt.Errorf("failed to list NAT Gateway sessions: %w", err)
	}
	for _, s := range sessions {
		if s == nil || len(s.SessionCount) == 0 {
			continue
		}
		return s.SpeedMbps, s.SessionCount[0], nil
	}
	return 0, 0, fmt.Errorf("no NAT Gateway session/speed pairs available")
}

// checkNATGatewayProvisioned asserts the resource's provisioning_status is
// one of the ready states (CONFIGURED or LIVE), confirming the validate/buy
// flow ran and the service was actually purchased.
func checkNATGatewayProvisioned(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found", resourceName)
		}
		status := rs.Primary.Attributes["provisioning_status"]
		if !slices.Contains(megaport.SERVICE_STATE_READY, status) {
			return fmt.Errorf("expected provisioning_status in %v, got %q", megaport.SERVICE_STATE_READY, status)
		}
		return nil
	}
}

// TestAccMegaportNATGateway_Basic tests the full lifecycle of a NAT Gateway resource
func TestAccMegaportNATGateway_Basic(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	natGWName := RandomTestName()
	natGWNameUpdated := RandomTestName()
	resourceName := "megaport_nat_gateway.test"

	speed, sessionCount, err := getNATGatewayTestConfig()
	if err != nil {
		t.Skipf("Skipping NAT Gateway test: %v", err)
	}
	locationID, _ := findNATGatewayTestLocation(t, speed)

	configInitial := providerConfig + fmt.Sprintf(`
data "megaport_location" "test_location" {
    id = %d
}

resource "megaport_nat_gateway" "test" {
    product_name         = "%s"
    location_id          = data.megaport_location.test_location.id
    speed                = %d
    session_count        = %d
    contract_term_months = 1
    diversity_zone       = "red"
    asn                  = 64512

    resource_tags = {
        "key1" = "value1"
        "key2" = "value2"
    }
}
`, locationID, natGWName, speed, sessionCount)

	configUpdated := providerConfig + fmt.Sprintf(`
data "megaport_location" "test_location" {
    id = %d
}

resource "megaport_nat_gateway" "test" {
    product_name         = "%s"
    location_id          = data.megaport_location.test_location.id
    speed                = %d
    session_count        = %d
    contract_term_months = 1
    diversity_zone       = "red"
    asn                  = 64512

    resource_tags = {
        "key1" = "value1-updated"
        "key3" = "value3"
    }
}
`, locationID, natGWNameUpdated, speed, sessionCount)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and verify
			{
				Config: configInitial,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "product_name", natGWName),
					resource.TestCheckResourceAttr(resourceName, "speed", fmt.Sprintf("%d", speed)),
					resource.TestCheckResourceAttr(resourceName, "contract_term_months", "1"),
					resource.TestCheckResourceAttr(resourceName, "diversity_zone", "red"),
					resource.TestCheckResourceAttr(resourceName, "resource_tags.key1", "value1"),
					resource.TestCheckResourceAttr(resourceName, "resource_tags.key2", "value2"),
					resource.TestCheckResourceAttrSet(resourceName, "product_uid"),
					resource.TestCheckResourceAttrSet(resourceName, "create_date"),
					resource.TestCheckResourceAttrSet(resourceName, "created_by"),
					resource.TestCheckResourceAttrSet(resourceName, "location_id"),
					resource.TestCheckResourceAttrSet(resourceName, "asn"),
					resource.TestCheckResourceAttrSet(resourceName, "provisioning_status"),
					checkNATGatewayProvisioned(resourceName),
				),
			},
			// ImportState testing
			{
				ResourceName:                         resourceName,
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "product_uid",
				// provisioning_status is a runtime field that transitions
				// CONFIGURED -> LIVE after Create returns on the earliest
				// ready state. Skip the equality check for it on import.
				ImportStateVerifyIgnore: []string{"provisioning_status"},
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
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
			},
			// Update and verify
			{
				Config: configUpdated,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "product_name", natGWNameUpdated),
					resource.TestCheckResourceAttr(resourceName, "resource_tags.key1", "value1-updated"),
					resource.TestCheckResourceAttr(resourceName, "resource_tags.key3", "value3"),
					resource.TestCheckNoResourceAttr(resourceName, "resource_tags.key2"),
					checkNATGatewayProvisioned(resourceName),
				),
			},
		},
	})
}

// TestAccMegaportNATGateway_PromoCode exercises promo_code on megaport_nat_gateway.
// The API does not echo back the promo code, so state must track the config-supplied
// value; this test verifies: create with promo → change promo → remove promo.
func TestAccMegaportNATGateway_PromoCode(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()

	speed, sessionCount, err := getNATGatewayTestConfig()
	if err != nil {
		t.Skipf("Skipping NAT Gateway promo code test: %v", err)
	}
	locationID, _ := findNATGatewayTestLocation(t, speed)
	natGWName := RandomTestName()
	resourceName := "megaport_nat_gateway.test"
	const initialPromo = "tf-acc-test-promo-initial"
	const otherPromo = "tf-acc-test-promo-other"

	configFor := func(promoLine string) string {
		return providerConfig + fmt.Sprintf(`
data "megaport_location" "test_location" {
    id = %d
}

resource "megaport_nat_gateway" "test" {
    product_name         = "%s"
    location_id          = data.megaport_location.test_location.id
    speed                = %d
    session_count        = %d
    contract_term_months = 1
    diversity_zone       = "red"
    asn                  = 64512
    %s
}
`, locationID, natGWName, speed, sessionCount, promoLine)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: configFor(fmt.Sprintf(`promo_code = "%s"`, initialPromo)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "promo_code", initialPromo),
					resource.TestCheckResourceAttrSet(resourceName, "product_uid"),
					checkNATGatewayProvisioned(resourceName),
				),
			},
			{
				Config: configFor(fmt.Sprintf(`promo_code = "%s"`, otherPromo)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "promo_code", otherPromo),
				),
			},
			{
				Config: configFor(""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr(resourceName, "promo_code"),
				),
			},
		},
	})
}
