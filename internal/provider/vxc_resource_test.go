package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccMegaportVXC_Basic(t *testing.T) {
	portName1 := RandomTestName()
	portName2 := RandomTestName()
	vxcName := RandomTestName()
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "loc" {
					name = "NextDC B1"
				}
					resource "megaport_port" "port_1" {
                    product_name  = "%s"
                    port_speed  = 1000
                    location_id = data.megaport_location.loc.id
                    contract_term_months        = 1
					market = "AU"
					marketplace_visibility = false
                  }
                  resource "megaport_port" "port_2" {
                    product_name  = "%s"
                    port_speed  = 1000
                    location_id = data.megaport_location.loc.id
                    contract_term_months        = 1
					market = "AU"
					marketplace_visibility = false
                  }
                  resource "megaport_vxc" "vxc" {
                    product_name   = "%s"
                    rate_limit = 1000
                    contract_term_months = 1
                    port_uid = megaport_port.port_1.product_uid

                    a_end = {
                    }

                    b_end = {
                        ordered_product_uid = megaport_port.port_2.product_uid
                    }
                  }
                  `, portName1, portName2, vxcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_port.port_1", "product_name", portName1),
					resource.TestCheckResourceAttr("megaport_port.port_1", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_port.port_1", "contract_term_months", "1"),
					resource.TestCheckResourceAttr("megaport_port.port_1", "market", "AU"),
					resource.TestCheckResourceAttr("megaport_port.port_1", "marketplace_visibility", "false"),
					resource.TestCheckResourceAttrSet("megaport_port.port_1", "product_uid"),
					resource.TestCheckResourceAttr("megaport_port.port_2", "product_name", portName2),
					resource.TestCheckResourceAttr("megaport_port.port_2", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_port.port_2", "contract_term_months", "1"),
					resource.TestCheckResourceAttr("megaport_port.port_2", "market", "AU"),
					resource.TestCheckResourceAttr("megaport_port.port_2", "marketplace_visibility", "false"),
					resource.TestCheckResourceAttrSet("megaport_port.port_2", "product_uid"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "product_name", vxcName),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "rate_limit", "1000"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "contract_term_months", "1"),
					resource.TestCheckResourceAttrSet("megaport_vxc.vxc", "product_uid"),
				),
			},
			// ImportState testing
			{
				ResourceName:                         "megaport_vxc.vxc",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "product_uid",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resourceName := "megaport_vxc.vxc"
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
				ImportStateVerifyIgnore: []string{"last_updated", "port_uid", "a_end_partner_config", "b_end_partner_config"},
			},
		},
	})
}

func TestAccMegaportMCRVXC_Basic(t *testing.T) {
	mcrName := RandomTestName()
	vxcName1 := RandomTestName()
	vxcName2 := RandomTestName()
	vxcName3 := RandomTestName()
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
                      ordered_vlan = 2191
                    }

                    b_end = {
                        ordered_product_uid = data.megaport_partner.aws_port.product_uid
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

                  resource "megaport_vxc" "gcp_vxc" {
                    product_name   = "%s"
                    rate_limit = 1000
                    contract_term_months = 1
                    port_uid        = megaport_mcr.mcr.product_uid

                    a_end = {
                      ordered_vlan = 182
                    }

                    b_end = {}

                    b_end_partner_config = {
                        partner = "google"
                        google_config = {
                            pairing_key = "7e51371e-72a3-40b5-b844-2e3efefaee59/australia-southeast1/2"
                        }
                    }
                  }

                  resource "megaport_vxc" "azure_vxc" {
                    product_name   = "%s"
                    rate_limit = 200
                    contract_term_months = 1
                    port_uid        = megaport_mcr.mcr.product_uid

                    a_end = {
                      ordered_vlan = 0
                    }

                    b_end = {}

                    b_end_partner_config = {
                        partner = "azure"
                        azure_config = {
                            service_key = "1b2329a5-56dc-45d0-8a0d-87b706297777"
                        }
                    }
                  }
                  `, mcrName, vxcName1, vxcName1, vxcName2, vxcName3),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("megaport_vxc.aws_vxc", "product_uid"),
					resource.TestCheckResourceAttrSet("megaport_vxc.azure_vxc", "product_uid"),
				),
			},
		},
	})
}
