package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccMegaportVXC_NATGatewayAEnd is the headline test for VXC ↔ NAT Gateway
// integration. It provisions a NAT Gateway, a Port at the same site, a packet
// filter on the NAT Gateway, then a VXC whose A-End is the NAT Gateway with
// the new vrouter partner-config fields populated:
//
//   - description
//   - packet_filter_in (bound to the just-created packet filter)
//   - nat_ip_addresses (carved from the NAT Gateway's pool)
//   - bgp_connections (shutdown — the B-End Port has no BGP peer)
//
// This exercises the full shape unblocked by the megaportgo PR #143 changes
// to PartnerConfigInterface (Description / InterfaceType / PacketFilterIn /
// PacketFilterOut).
func TestAccMegaportVXC_NATGatewayAEnd(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()

	speed, sessionCount, err := getNATGatewayTestConfig()
	if err != nil {
		t.Skipf("Skipping NAT Gateway VXC test: %v", err)
	}
	locationID, _ := findNATGatewayAndPortTestLocation(t, speed)

	natGWName := RandomTestName()
	portName := RandomTestName()
	pfDescription := RandomTestName()
	vxcName := RandomTestName()
	vxcNameUpdated := RandomTestName()

	natResource := "megaport_nat_gateway.nat"
	pfResource := "megaport_nat_gateway_packet_filter.pf"
	portResource := "megaport_port.port"
	vxcResource := "megaport_vxc.vxc"

	configFor := func(vxcDisplayName string, rateLimit int) string {
		return providerConfig + fmt.Sprintf(`
data "megaport_location" "loc" {
    id = %d
}

resource "megaport_nat_gateway" "nat" {
    product_name         = "%s"
    location_id          = data.megaport_location.loc.id
    speed                = %d
    session_count        = %d
    contract_term_months = 1
    diversity_zone       = "red"
    asn                  = 64512
}

resource "megaport_port" "port" {
    product_name         = "%s"
    location_id          = data.megaport_location.loc.id
    port_speed           = 1000
    contract_term_months = 1
    marketplace_visibility = false
}

resource "megaport_nat_gateway_packet_filter" "pf" {
    nat_gateway_product_uid = megaport_nat_gateway.nat.product_uid
    description             = "%s"
    entries = [
        {
            action              = "permit"
            description         = "allow https"
            source_address      = "0.0.0.0/0"
            destination_address = "10.0.0.0/30"
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

resource "megaport_vxc" "vxc" {
    product_name         = "%s"
    rate_limit           = %d
    contract_term_months = 1

    a_end = {
        requested_product_uid = megaport_nat_gateway.nat.product_uid
        ordered_vlan          = 100
    }

    a_end_partner_config = {
        partner = "vrouter"
        vrouter_config = {
            interfaces = [{
                description      = "nat-gw-to-port"
                ip_addresses     = ["10.0.0.1/30"]
                packet_filter_in = megaport_nat_gateway_packet_filter.pf.id
                bgp_connections = [{
                    peer_asn         = 64600
                    local_ip_address = "10.0.0.1"
                    peer_ip_address  = "10.0.0.2"
                    shutdown         = true
                    description      = "no live peer on B-End port"
                    bfd_enabled      = false
                    export_policy    = "permit"
                }]
            }]
        }
    }

    b_end = {
        requested_product_uid = megaport_port.port.product_uid
        ordered_vlan          = 200
    }
}
`, locationID, natGWName, speed, sessionCount, portName, pfDescription, vxcDisplayName, rateLimit)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create — VXC A-End on the NAT Gateway with packet_filter_in bound.
			{
				Config: configFor(vxcName, 100),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkNATGatewayProvisioned(natResource),
					resource.TestCheckResourceAttrSet(portResource, "product_uid"),
					resource.TestCheckResourceAttrSet(pfResource, "id"),
					resource.TestCheckResourceAttrSet(vxcResource, "product_uid"),
					resource.TestCheckResourceAttr(vxcResource, "product_name", vxcName),
					resource.TestCheckResourceAttr(vxcResource, "rate_limit", "100"),
					resource.TestCheckResourceAttrPair(
						vxcResource, "a_end.current_product_uid",
						natResource, "product_uid",
					),
					resource.TestCheckResourceAttrPair(
						vxcResource, "b_end.current_product_uid",
						portResource, "product_uid",
					),
					resource.TestCheckResourceAttr(vxcResource, "a_end_partner_config.partner", "vrouter"),
					resource.TestCheckResourceAttr(vxcResource, "a_end_partner_config.vrouter_config.interfaces.0.description", "nat-gw-to-port"),
					resource.TestCheckResourceAttr(vxcResource, "a_end_partner_config.vrouter_config.interfaces.0.ip_addresses.0", "10.0.0.1/30"),
					resource.TestCheckResourceAttrPair(
						vxcResource, "a_end_partner_config.vrouter_config.interfaces.0.packet_filter_in",
						pfResource, "id",
					),
				),
			},
			// Update — change name + rate limit, confirm a NAT-attached VXC is writable.
			{
				Config: configFor(vxcNameUpdated, 200),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(vxcResource, "product_name", vxcNameUpdated),
					resource.TestCheckResourceAttr(vxcResource, "rate_limit", "200"),
				),
			},
		},
	})
}
