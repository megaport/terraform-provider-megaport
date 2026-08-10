package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccMegaportVXCCSPConnection_Basic provisions a VXC with an AWS CSP
// connection and reads it back through the megaport_vxc_csp_connection data
// source, asserting the csp_connections list populates.
func TestAccMegaportVXCCSPConnection_Basic(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()
	locs := findVXCPortTestLocationsWithPartner(t, 1, "AWSHC")
	portName := RandomTestName()
	vxcName := RandomTestName()

	config := providerConfig + fmt.Sprintf(`
		data "megaport_location" "loc" {
			id = %d
		}

		data "megaport_partner" "aws_port" {
			connect_type = "AWSHC"
			location_id  = data.megaport_location.loc.id
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
				requested_product_uid = megaport_port.port.product_uid
				ordered_vlan          = 200
			}

			b_end = {
				requested_product_uid = data.megaport_partner.aws_port.product_uid
			}

			b_end_partner_config = {
				partner = "aws"
				aws_config = {
					name          = "%s"
					type          = "private"
					connect_type  = "AWSHC"
					owner_account = "123456789012"
				}
			}
		}

		data "megaport_vxc_csp_connection" "csp" {
			vxc_uid = megaport_vxc.vxc.product_uid
		}
	`, locs[0], portName, vxcName, vxcName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.megaport_vxc_csp_connection.csp", "vxc_uid", "megaport_vxc.vxc", "product_uid"),
					resource.TestCheckResourceAttrSet("data.megaport_vxc_csp_connection.csp", "csp_connections.#"),
					resource.TestCheckResourceAttrSet("data.megaport_vxc_csp_connection.csp", "csp_connections.0.connect_type"),
					resource.TestCheckResourceAttrSet("data.megaport_vxc_csp_connection.csp", "csp_connections.0.resource_type"),
				),
			},
		},
	})
}
