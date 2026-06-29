package provider

import (
	"context"
	"fmt"
	"regexp"
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

// findInvalidNATGatewaySpeed derives a speed guaranteed not to be in the
// live NAT Gateway availability matrix by returning max(SpeedMbps)+1. It
// also returns any positive session count from the matrix so the HCL
// parses and the validator's session-count pre-check passes (the test
// targets the speed sentinel, not the session-count sentinel). The two
// selections are independent: if the highest-speed entry has no
// SessionCount values, we still pick a session count from another entry.
// Deriving both values dynamically keeps the test stable as the matrix
// evolves; a hard-coded "500 Mbps is invalid" would silently break the
// day 500 Mbps is added.
func findInvalidNATGatewaySpeed() (invalidSpeed, sessionCount int, err error) {
	client, err := getTestClient()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to create test client: %w", err)
	}
	sessions, err := client.NATGatewayService.ListNATGatewaySessions(context.Background())
	if err != nil {
		return 0, 0, fmt.Errorf("failed to list NAT Gateway sessions: %w", err)
	}
	maxSpeed := 0
	for _, s := range sessions {
		if s == nil {
			continue
		}
		if s.SpeedMbps > maxSpeed {
			maxSpeed = s.SpeedMbps
		}
		if sessionCount == 0 {
			for _, c := range s.SessionCount {
				if c > 0 {
					sessionCount = c
					break
				}
			}
		}
	}
	if maxSpeed <= 0 {
		return 0, 0, fmt.Errorf("no positive NAT Gateway speeds advertised")
	}
	if sessionCount <= 0 {
		return 0, 0, fmt.Errorf("no positive session count advertised in NAT Gateway matrix")
	}
	return maxSpeed + 1, sessionCount, nil
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

// TestAccMegaportNATGateway_PlanTimeRejectsInvalidSpeed asserts that the
// resource's ModifyPlan hook rejects an invalid speed during terraform
// plan, before any provisioning occurs. The invalid speed is derived from
// the live availability matrix (max advertised speed + 1 Mbps) so the
// test stays accurate as the matrix evolves.
func TestAccMegaportNATGateway_PlanTimeRejectsInvalidSpeed(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()

	invalidSpeed, sessionCount, err := findInvalidNATGatewaySpeed()
	if err != nil {
		t.Skipf("Skipping NAT Gateway plan-time validation test: %v", err)
	}

	// Preflight the same matrix lookup the provider's ModifyPlan will use, so
	// we only assert on the validation outcome when the lookup is actually
	// working. The provider downgrades operational lookup failures (transport,
	// auth, 5xx) to a warning, which would leave the PlanOnly step with no
	// error to match. Skip in that case instead of flaking.
	client, err := getTestClient()
	if err != nil {
		t.Skipf("Skipping NAT Gateway plan-time validation test: %v", err)
	}
	matrix, err := client.NATGatewayService.ListNATGatewaySessions(context.Background())
	if err != nil {
		t.Skipf("Skipping NAT Gateway plan-time validation test: matrix lookup failed operationally: %v", err)
	}
	if _, _, ok := natGatewaySpeedSessionSupported(matrix, invalidSpeed, sessionCount); ok {
		t.Skipf("Skipping NAT Gateway plan-time validation test: derived invalid speed %d Mbps is supported by the matrix (matrix may have changed)", invalidSpeed)
	}
	// Find a location that supports the highest advertised speed (one less
	// than invalidSpeed). The location is needed for the HCL to parse; the
	// plan-only step never reaches provisioning.
	locationID, _ := findNATGatewayTestLocation(t, invalidSpeed-1)
	natGWName := RandomTestName()

	config := providerConfig + fmt.Sprintf(`
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
}
`, locationID, natGWName, invalidSpeed, sessionCount)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      config,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`Invalid NAT Gateway speed / session count combination`),
			},
		},
	})
}
