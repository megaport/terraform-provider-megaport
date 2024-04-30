package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccMegaportMCRVXC_Basic(t *testing.T) {
	mcrName := RandomTestName()
	vxcName1 := RandomTestName()
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "bne_nxt1" {
                    name    = "NextDC B1"
                  }
                  
                  data "megaport_location" "syd_gs" {
                    name = "Global Switch Sydney West"
                  }

                  data "megaport_partner" "aws_port" {
                    connect_type = "AWS"
                    company_name = "AWS"
                    product_name = "Asia Pacific (Sydney) (ap-southeast-2)"
                    location_id  = data.megaport_location.syd_gs.id
                  }

                  resource "megaport_mcr" "mcr" {
                    product_name    = "%s"
                    location_id = data.megaport_location.bne_nxt1.id
                    marketplace_visibility = false
                    market = "AU"
                    contract_term_months = 1
                    port_speed = 5000
                    asn = 64555
                  }

                  resource "megaport_vxc" "aws_vxc" {
                    product_name   = "%s"
                    rate_limit = 1000
                    port_uid = megaport_mcr.mcr.product_uid
                    contract_term_months = 1
                  
                    a_end = {
                      vlan = 2191
                    }

                    b_end = {
                        product_uid = data.megaport_partner.aws_port.product_uid
                    }

                    b_end_partner_config = {
                        partner = "aws"
                        aws_config = {
                            name = "%s"
                            asn = 64550
                            type = "private"
                            connect_type = "AWS"
                            amazon_asn = 64551
                            owner_account = "123456789012"
                        }
                    }
                  }
                  `, mcrName, vxcName1, vxcName1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("megaport_vxc.aws_vxc", "product_uid"),
				),
			},
		},
	})
}
