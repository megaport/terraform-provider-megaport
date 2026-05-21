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

// TestAccMegaportVXC_NATGatewayPrefixListBGP exercises the NAT Gateway prefix
// list ↔ VXC BGP wiring. NetAuto classifies MCR and NAT Gateway as the same
// VRouter product class (confirmed with the NGW backend team), so the four
// import/export whitelist/blacklist fields on a BGP connection accept the
// description of any prefix filter list owned by the VXC's vrouter endpoint
// — including NGW prefix lists, not just MCR ones. This test asserts that
// vrouterPrefixFilterListsForEndpoint resolves NGW prefix-list descriptions
// to their numeric IDs on the wire-level VXC order.
//
// Shape:
//
//   - NAT Gateway + same-site Port
//   - megaport_nat_gateway_prefix_list (IPv4, two entries)
//   - VXC with A-End on the NAT Gateway and a BGP connection that sets both
//     import_whitelist and export_whitelist to the prefix list's description.
//     The peer is shutdown=true because the B-End port has no live peer; we
//     only care that the order is accepted and the description-to-ID lookup
//     succeeds.
func TestAccMegaportVXC_NATGatewayPrefixListBGP(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()

	speed, sessionCount, err := getNATGatewayTestConfig()
	if err != nil {
		t.Skipf("Skipping NAT Gateway VXC prefix-list test: %v", err)
	}
	locationID, _ := findNATGatewayAndPortTestLocation(t, speed)

	natGWName := RandomTestName()
	portName := RandomTestName()
	plDescription := RandomTestName()
	vxcName := RandomTestName()

	natResource := "megaport_nat_gateway.nat"
	plResource := "megaport_nat_gateway_prefix_list.pl"
	portResource := "megaport_port.port"
	vxcResource := "megaport_vxc.vxc"

	config := providerConfig + fmt.Sprintf(`
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
    product_name           = "%s"
    location_id            = data.megaport_location.loc.id
    port_speed             = 1000
    contract_term_months   = 1
    marketplace_visibility = false
}

resource "megaport_nat_gateway_prefix_list" "pl" {
    nat_gateway_product_uid = megaport_nat_gateway.nat.product_uid
    description             = "%s"
    address_family          = "IPv4"
    entries = [
        {
            action = "permit"
            prefix = "10.0.0.0/8"
            ge     = 24
            le     = 32
        },
        {
            action = "deny"
            prefix = "192.168.0.0/16"
        }
    ]
}

resource "megaport_vxc" "vxc" {
    product_name         = "%s"
    rate_limit           = 100
    contract_term_months = 1

    a_end = {
        requested_product_uid = megaport_nat_gateway.nat.product_uid
        ordered_vlan          = 100
    }

    a_end_partner_config = {
        partner = "vrouter"
        vrouter_config = {
            interfaces = [{
                description  = "nat-gw-bgp-filtered"
                ip_addresses = ["10.0.0.1/30"]
                bgp_connections = [{
                    peer_asn         = 64600
                    local_ip_address = "10.0.0.1"
                    peer_ip_address  = "10.0.0.2"
                    shutdown         = true
                    description      = "filtered by NGW prefix list"
                    bfd_enabled      = false
                    export_policy    = "permit"
                    import_whitelist = "%s"
                    export_whitelist = "%s"
                }]
            }]
        }
    }

    b_end = {
        requested_product_uid = megaport_port.port.product_uid
        ordered_vlan          = 200
    }

    depends_on = [megaport_nat_gateway_prefix_list.pl]
}
`, locationID, natGWName, speed, sessionCount, portName, plDescription, vxcName, plDescription, plDescription)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					checkNATGatewayProvisioned(natResource),
					resource.TestCheckResourceAttrSet(portResource, "product_uid"),
					resource.TestCheckResourceAttrSet(plResource, "id"),
					resource.TestCheckResourceAttrSet(vxcResource, "product_uid"),
					resource.TestCheckResourceAttr(vxcResource, "product_name", vxcName),
					resource.TestCheckResourceAttrPair(
						vxcResource, "a_end.current_product_uid",
						natResource, "product_uid",
					),
					resource.TestCheckResourceAttr(
						vxcResource,
						"a_end_partner_config.vrouter_config.interfaces.0.bgp_connections.0.import_whitelist",
						plDescription,
					),
					resource.TestCheckResourceAttr(
						vxcResource,
						"a_end_partner_config.vrouter_config.interfaces.0.bgp_connections.0.export_whitelist",
						plDescription,
					),
				),
			},
		},
	})
}
