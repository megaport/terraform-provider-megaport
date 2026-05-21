package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccMegaportNATGatewayPacketFilter_Basic exercises the create → read →
// import → update lifecycle of megaport_nat_gateway_packet_filter against a
// freshly-provisioned NAT Gateway in staging.
func TestAccMegaportNATGatewayPacketFilter_Basic(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()

	speed, sessionCount, err := getNATGatewayTestConfig()
	if err != nil {
		t.Skipf("Skipping NAT Gateway packet filter test: %v", err)
	}
	locationID, _ := findNATGatewayTestLocation(t, speed)
	natGWName := RandomTestName()
	pfDescription := RandomTestName()
	pfDescriptionUpdated := RandomTestName()

	natResourceName := "megaport_nat_gateway.test"
	pfResourceName := "megaport_nat_gateway_packet_filter.test"

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
}

resource "megaport_nat_gateway_packet_filter" "test" {
    nat_gateway_product_uid = megaport_nat_gateway.test.product_uid
    description             = "%s"
    entries = [
        {
            action              = "permit"
            description         = "allow https"
            source_address      = "0.0.0.0/0"
            destination_address = "10.0.0.0/24"
            destination_ports   = "443"
            ip_protocol         = 6
        },
        {
            action              = "deny"
            source_address      = "0.0.0.0/0"
            destination_address = "0.0.0.0/0"
        }
    ]
}
`, locationID, natGWName, speed, sessionCount, pfDescription)

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
}

resource "megaport_nat_gateway_packet_filter" "test" {
    nat_gateway_product_uid = megaport_nat_gateway.test.product_uid
    description             = "%s"
    entries = [
        {
            action              = "permit"
            description         = "allow https"
            source_address      = "0.0.0.0/0"
            destination_address = "10.0.0.0/24"
            destination_ports   = "443"
            ip_protocol         = 6
        },
        {
            action              = "permit"
            description         = "allow dns udp"
            source_address      = "0.0.0.0/0"
            destination_address = "10.0.0.0/24"
            destination_ports   = "53"
            ip_protocol         = 17
        },
        {
            action              = "deny"
            source_address      = "0.0.0.0/0"
            destination_address = "0.0.0.0/0"
        }
    ]
}
`, locationID, natGWName, speed, sessionCount, pfDescriptionUpdated)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create + verify
			{
				Config: configInitial,
				Check: resource.ComposeAggregateTestCheckFunc(
					checkNATGatewayProvisioned(natResourceName),
					resource.TestCheckResourceAttr(pfResourceName, "description", pfDescription),
					resource.TestCheckResourceAttr(pfResourceName, "entries.#", "2"),
					resource.TestCheckResourceAttr(pfResourceName, "entries.0.action", "permit"),
					resource.TestCheckResourceAttr(pfResourceName, "entries.0.description", "allow https"),
					resource.TestCheckResourceAttr(pfResourceName, "entries.0.destination_ports", "443"),
					resource.TestCheckResourceAttr(pfResourceName, "entries.0.ip_protocol", "6"),
					resource.TestCheckResourceAttr(pfResourceName, "entries.1.action", "deny"),
					resource.TestCheckResourceAttrSet(pfResourceName, "id"),
					resource.TestCheckResourceAttrPair(
						pfResourceName, "nat_gateway_product_uid",
						natResourceName, "product_uid",
					),
				),
			},
			// ImportState
			{
				ResourceName:                         pfResourceName,
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "id",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					rs, ok := state.RootModule().Resources[pfResourceName]
					if !ok {
						return "", fmt.Errorf("resource %s not found", pfResourceName)
					}
					return fmt.Sprintf("%s:%s",
						rs.Primary.Attributes["nat_gateway_product_uid"],
						rs.Primary.Attributes["id"],
					), nil
				},
			},
			// Update
			{
				Config: configUpdated,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(pfResourceName, "description", pfDescriptionUpdated),
					resource.TestCheckResourceAttr(pfResourceName, "entries.#", "3"),
					resource.TestCheckResourceAttr(pfResourceName, "entries.1.description", "allow dns udp"),
					resource.TestCheckResourceAttr(pfResourceName, "entries.1.ip_protocol", "17"),
					resource.TestCheckResourceAttr(pfResourceName, "entries.2.action", "deny"),
				),
			},
		},
	})
}
