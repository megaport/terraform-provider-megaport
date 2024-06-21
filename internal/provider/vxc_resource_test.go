package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/suite"
)

type VXCBasicProviderTestSuite ProviderTestSuite
type VXCWithCSPsProviderTestSuite ProviderTestSuite
type VXCWithMVEProviderTestSuite ProviderTestSuite

func TestVXCBasicProviderTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(VXCBasicProviderTestSuite))
}

func (suite *VXCBasicProviderTestSuite) TestAccMegaportVXC_Basic() {
	portName1 := RandomTestName()
	portName2 := RandomTestName()
	portName3 := RandomTestName()
	portName4 := RandomTestName()
	vxcName := RandomTestName()
	vxcNameNew := RandomTestName()
	costCentreName := RandomTestName()
	costCentreNew := RandomTestName()

	resource.Test(suite.T(), resource.TestCase{
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
                    contract_term_months        = 12
					marketplace_visibility = false
                  }
                  resource "megaport_port" "port_2" {
                    product_name  = "%s"
                    port_speed  = 1000
                    location_id = data.megaport_location.loc.id
                    contract_term_months        = 12
					marketplace_visibility = false
                  }
				  resource "megaport_port" "port_3" {
                    product_name  = "%s"
                    port_speed  = 1000
                    location_id = data.megaport_location.loc.id
                    contract_term_months        = 12
					marketplace_visibility = false
                  }
                  resource "megaport_port" "port_4" {
                    product_name  = "%s"
                    port_speed  = 1000
                    location_id = data.megaport_location.loc.id
                    contract_term_months        = 12
					marketplace_visibility = false
                  }
                  resource "megaport_vxc" "vxc" {
                    product_name   = "%s"
                    rate_limit = 1000
                    contract_term_months = 12
					cost_centre = "%s"

                    a_end = {
                        requested_product_uid = megaport_port.port_1.product_uid
                    }

                    b_end = {
                        requested_product_uid = megaport_port.port_2.product_uid
                    }
                  }
                  `, portName1, portName2, portName3, portName4, vxcName, costCentreName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_port.port_1", "product_name", portName1),
					resource.TestCheckResourceAttr("megaport_port.port_1", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_port.port_1", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_port.port_1", "marketplace_visibility", "false"),
					resource.TestCheckResourceAttrSet("megaport_port.port_1", "product_uid"),
					resource.TestCheckResourceAttr("megaport_port.port_2", "product_name", portName2),
					resource.TestCheckResourceAttr("megaport_port.port_2", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_port.port_2", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_port.port_2", "marketplace_visibility", "false"),
					resource.TestCheckResourceAttrSet("megaport_port.port_2", "product_uid"),
					resource.TestCheckResourceAttr("megaport_port.port_3", "product_name", portName3),
					resource.TestCheckResourceAttr("megaport_port.port_3", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_port.port_3", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_port.port_3", "marketplace_visibility", "false"),
					resource.TestCheckResourceAttrSet("megaport_port.port_3", "product_uid"),
					resource.TestCheckResourceAttr("megaport_port.port_4", "product_name", portName4),
					resource.TestCheckResourceAttr("megaport_port.port_4", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_port.port_4", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_port.port_4", "marketplace_visibility", "false"),
					resource.TestCheckResourceAttrSet("megaport_port.port_4", "product_uid"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "product_name", vxcName),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "rate_limit", "1000"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "contract_term_months", "12"),
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
				ImportStateVerifyIgnore: []string{"last_updated", "a_end_partner_config", "b_end_partner_config", "a_end", "b_end", "contract_start_date", "contract_end_date", "live_date", "resources", "provisioning_status"},
			},
			// Update Tests
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "loc" {
					name = "NextDC B1"
				}
					resource "megaport_port" "port_1" {
			        product_name  = "%s"
			        port_speed  = 1000
			        location_id = data.megaport_location.loc.id
			        contract_term_months        = 12
					cost_centre = "test"
					marketplace_visibility = false
			      }
			      resource "megaport_port" "port_2" {
			        product_name  = "%s"
			        port_speed  = 1000
			        location_id = data.megaport_location.loc.id
			        contract_term_months        = 12
					cost_centre = "test"
					marketplace_visibility = false
			      }
				  resource "megaport_port" "port_3" {
                    product_name  = "%s"
                    port_speed  = 1000
                    location_id = data.megaport_location.loc.id
                    contract_term_months        = 12
					marketplace_visibility = false
                  }
                  resource "megaport_port" "port_4" {
                    product_name  = "%s"
                    port_speed  = 1000
                    location_id = data.megaport_location.loc.id
                    contract_term_months        = 12
					marketplace_visibility = false
                  }
			      resource "megaport_vxc" "vxc" {
			        product_name   = "%s"
			        rate_limit = 1000
					contract_term_months = 12

			        a_end = {
			            requested_product_uid = megaport_port.port_3.product_uid
			        }

			        b_end = {
			            requested_product_uid = megaport_port.port_4.product_uid
			        }
			      }
			      `, portName1, portName2, portName3, portName4, vxcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_port.port_1", "product_name", portName1),
					resource.TestCheckResourceAttr("megaport_port.port_1", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_port.port_1", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_port.port_1", "marketplace_visibility", "false"),
					resource.TestCheckResourceAttrSet("megaport_port.port_1", "product_uid"),
					resource.TestCheckResourceAttr("megaport_port.port_2", "product_name", portName2),
					resource.TestCheckResourceAttr("megaport_port.port_2", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_port.port_2", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_port.port_2", "marketplace_visibility", "false"),
					resource.TestCheckResourceAttrSet("megaport_port.port_2", "product_uid"),
					resource.TestCheckResourceAttr("megaport_port.port_3", "product_name", portName3),
					resource.TestCheckResourceAttr("megaport_port.port_3", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_port.port_3", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_port.port_3", "marketplace_visibility", "false"),
					resource.TestCheckResourceAttrSet("megaport_port.port_3", "product_uid"),
					resource.TestCheckResourceAttr("megaport_port.port_4", "product_name", portName4),
					resource.TestCheckResourceAttr("megaport_port.port_4", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_port.port_4", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_port.port_4", "marketplace_visibility", "false"),
					resource.TestCheckResourceAttrSet("megaport_port.port_4", "product_uid"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "product_name", vxcName),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "rate_limit", "1000"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "contract_term_months", "12"),
					resource.TestCheckResourceAttrSet("megaport_vxc.vxc", "product_uid"),
				),
			},
			// Update Tests
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "loc" {
					name = "NextDC B1"
				}
					resource "megaport_port" "port_1" {
			        product_name  = "%s"
			        port_speed  = 1000
			        location_id = data.megaport_location.loc.id
			        contract_term_months        = 12
					cost_centre = "test"
					marketplace_visibility = false
			      }
			      resource "megaport_port" "port_2" {
			        product_name  = "%s"
			        port_speed  = 1000
			        location_id = data.megaport_location.loc.id
			        contract_term_months        = 12
					cost_centre = "test"
					marketplace_visibility = false
			      }
				  resource "megaport_port" "port_3" {
                    product_name  = "%s"
                    port_speed  = 1000
                    location_id = data.megaport_location.loc.id
                    contract_term_months        = 12
					marketplace_visibility = false
                  }
                  resource "megaport_port" "port_4" {
                    product_name  = "%s"
                    port_speed  = 1000
                    location_id = data.megaport_location.loc.id
                    contract_term_months        = 12
					marketplace_visibility = false
                  }
			      resource "megaport_vxc" "vxc" {
			        product_name   = "%s"
			        rate_limit = 500
					contract_term_months = 12
					cost_centre = "%s"

			        a_end = {
			            requested_product_uid = megaport_port.port_3.product_uid
			        }

			        b_end = {
			            requested_product_uid = megaport_port.port_4.product_uid
			        }
			      }
			      `, portName1, portName2, portName3, portName4, vxcNameNew, costCentreNew),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_port.port_1", "product_name", portName1),
					resource.TestCheckResourceAttr("megaport_port.port_1", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_port.port_1", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_port.port_1", "marketplace_visibility", "false"),
					resource.TestCheckResourceAttrSet("megaport_port.port_1", "product_uid"),
					resource.TestCheckResourceAttr("megaport_port.port_2", "product_name", portName2),
					resource.TestCheckResourceAttr("megaport_port.port_2", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_port.port_2", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_port.port_2", "marketplace_visibility", "false"),
					resource.TestCheckResourceAttrSet("megaport_port.port_2", "product_uid"),
					resource.TestCheckResourceAttr("megaport_port.port_3", "product_name", portName3),
					resource.TestCheckResourceAttr("megaport_port.port_3", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_port.port_3", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_port.port_3", "marketplace_visibility", "false"),
					resource.TestCheckResourceAttrSet("megaport_port.port_3", "product_uid"),
					resource.TestCheckResourceAttr("megaport_port.port_4", "product_name", portName4),
					resource.TestCheckResourceAttr("megaport_port.port_4", "port_speed", "1000"),
					resource.TestCheckResourceAttr("megaport_port.port_4", "contract_term_months", "12"),
					resource.TestCheckResourceAttr("megaport_port.port_4", "marketplace_visibility", "false"),
					resource.TestCheckResourceAttrSet("megaport_port.port_4", "product_uid"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "product_name", vxcNameNew),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "cost_centre", costCentreNew),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "rate_limit", "500"),
					resource.TestCheckResourceAttr("megaport_vxc.vxc", "contract_term_months", "12"),
					resource.TestCheckResourceAttrSet("megaport_vxc.vxc", "product_uid"),
				),
			},
		},
	})
}

func TestVXCWithCSPsProviderTestSuiteProviderTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(VXCWithCSPsProviderTestSuite))
}

func (suite *VXCWithCSPsProviderTestSuite) TestAccMegaportMCRVXCWithCSPs_Basic() {
	mcrName := RandomTestName()
	vxcName1 := RandomTestName()
	vxcName2 := RandomTestName()
	vxcName3 := RandomTestName()
	resource.Test(suite.T(), resource.TestCase{
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
                    contract_term_months = 1
                    port_speed = 5000
                    asn = 64555
                  }

                  resource "megaport_vxc" "aws_vxc" {
                    product_name   = "%s"
                    rate_limit = 1000
                    contract_term_months = 1

                    a_end = {
                      requested_product_uid = megaport_mcr.mcr.product_uid
                      ordered_vlan = 2191
                    }

                    b_end = {
                        requested_product_uid = data.megaport_partner.aws_port.product_uid
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

                    a_end = {
                      requested_product_uid = megaport_mcr.mcr.product_uid
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

                    a_end = {
                      requested_product_uid = megaport_mcr.mcr.product_uid
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

func (suite *VXCWithCSPsProviderTestSuite) TestAccMegaportMCRVXCWithBGP_Basic() {
	mcrName := RandomTestName()
	vxcName1 := RandomTestName()
	prefixFilterListName := RandomTestName()
	resource.Test(suite.T(), resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "bne_nxt1" {
					name = "NextDC B1"
				  }

				  data "megaport_location" "syd_ndc" {
					name = "NextDC C1"
				  }

				  data "megaport_partner" "aws_port" {
					connect_type = "AWS"
					company_name = "AWS"
					product_name = "Asia Pacific (Sydney) (ap-southeast-2)"
					location_id  = data.megaport_location.syd_ndc.id
				  }

				  resource "megaport_mcr" "mcr" {
					product_name            = "%s"
					location_id             = data.megaport_location.bne_nxt1.id
					contract_term_months    = 1
					port_speed              = 5000
					asn                     = 64555

					prefix_filter_lists = [{
					  description     = "%s"
					  address_family  = "IPv4"
					  entries = [
						{
						  action  = "permit"
						  prefix  = "10.0.1.0/24"
						  ge      = 24
						  le      = 24
						},
						{
						  action  = "deny"
						  prefix  = "10.0.2.0/24"
						  ge      = 24
						  le      = 24
						}
					  ]
					}]
				  }

				  resource "megaport_vxc" "aws_vxc" {
					product_name           = "%s"
					rate_limit             = 1000
					contract_term_months   = 1

					a_end = {
                      requested_product_uid = megaport_mcr.mcr.product_uid
					  ordered_vlan = 0
					}

					a_end_partner_config = {
					  partner = "a-end"
					  partner_a_end_config = {
						interfaces = [{
							ip_addresses     = ["10.0.0.1/30"]
							nat_ip_addresses = ["10.0.0.1"]
						  bfd = {
							tx_interval   = 500
							rx_interval   = 400
							multiplier    = 5
						  }
						  bgp_connections = [
							{
							  peer_asn          = 64512
							  local_ip_address  = "10.0.0.1"
							  peer_ip_address   = "10.0.0.2"
							  password          = "notARealPassword"
							  shutdown          = false
							  description       = "BGP Connection 1"
							  med_in            = 100
							  med_out           = 100
							  bfd_enabled       = true
							  export_policy     = "deny"
							  permit_export_to = ["10.0.1.2"]
							  import_white_list = "%s"
							}
						  ]
						}]
					  }
					}

					b_end = {
					  requested_product_uid = data.megaport_partner.aws_port.product_uid
					}

					b_end_partner_config = {
					  partner = "aws"
					  aws_config = {
						name            = "%s"
						asn             = 64550
						type            = "private"
						connect_type    = "AWSHC"
						amazon_asn      = 64551
						owner_account   = "684021030471"
					  }
					}
				  }
                  `, mcrName, prefixFilterListName, vxcName1, prefixFilterListName, vxcName1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("megaport_vxc.aws_vxc", "product_uid"),
					resource.TestCheckResourceAttr("megaport_vxc.aws_vxc", "b_end_partner_config.aws_config.name", vxcName1),
				),
			},
		},
	})
}

func TestVXCWithMVEProviderTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(VXCWithMVEProviderTestSuite))
}

func (suite *VXCWithMVEProviderTestSuite) TestMVE_TransitVXC() {
	portName := RandomTestName()
	costCentreName := RandomTestName()
	mveName := RandomTestName()
	transitVXCName := RandomTestName()

	resource.Test(suite.T(), resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "bne_nxt1" {
					name = "NextDC B1"
				  }

				  data "megaport_location" "syd_gs" {
					name = "Global Switch Sydney West"
				  }

				  resource "megaport_port" "port" {
					product_name           = "%s"
					port_speed             = 1000
					location_id            = data.megaport_location.bne_nxt1.id
					contract_term_months   = 12
					marketplace_visibility = true
					cost_centre            = "%s"
				  }

				  data "megaport_partner" "internet_port" {
					connect_type  = "TRANSIT"
					company_name  = "Networks"
					product_name  = "Megaport Internet"
					location_id   = data.megaport_location.syd_gs.id
				  }

				  resource "megaport_mve" "mve" {
					product_name           = "%s"
					location_id            = data.megaport_location.bne_nxt1.id
					contract_term_months   = 1

					vnics = [
					  {
						description = "Data Plane"
					  },
					  {
						description = "Management Plane"
					  },
					  {
						description = "Control Plane"
					  }
					]

					vendor_config = {
					  vendor        = "aruba"
					  product_size  = "MEDIUM"
					  image_id      = 23
					  account_name  = "%s"
					  account_key   = "%s"
					  system_tag    = "Preconfiguration-aruba-test-1"
					}
				  }

				  resource "megaport_vxc" "transit_vxc" {
					product_name         = "%s"
					rate_limit           = 100
					contract_term_months = 1

					a_end = {
					  requested_product_uid = megaport_mve.mve.product_uid
					  vnic_index            = 2
					}

					b_end = {
					  requested_product_uid = data.megaport_partner.internet_port.product_uid
					}

					b_end_partner_config = {
					  partner = "transit"
					}
				  }
                  `, portName, costCentreName, mveName, mveName, mveName, transitVXCName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("megaport_vxc.transit_vxc", "product_uid"),
				),
			},
		},
	})
}

func (suite *VXCWithMVEProviderTestSuite) TestMVE_AWS_VXC() {
	portName := RandomTestName()
	costCentreName := RandomTestName()
	mveName := RandomTestName()
	awsVXCName := RandomTestName()

	resource.Test(suite.T(), resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "bne_nxt1" {
					name = "NextDC B1"
				  }

				  data "megaport_location" "bne_nxt2" {
					name = "NextDC B2"
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

				  resource "megaport_port" "port" {
					product_name            = "%s"
					port_speed              = 1000
					location_id             = data.megaport_location.bne_nxt1.id
					contract_term_months    = 12
					marketplace_visibility  = true
					cost_centre = "%s"
				  }

				resource "megaport_mve" "mve" {
                    product_name  = "%s"
                    location_id = data.megaport_location.bne_nxt1.id
                    contract_term_months        = 1

					vnics = [
						{
							description = "to_aws"
						},
						{
								description = "to_port"
						},
					]

                    vendor_config = {
                        vendor = "aruba"
                        product_size = "MEDIUM"
                        image_id = 23
						account_name = "%s"
						account_key = "%s"
						system_tag = "Preconfiguration-aruba-test-1"
                    }
                  }

				  resource "megaport_vxc" "aws_vxc" {
					product_name            = "%s"
					rate_limit              = 100
					contract_term_months    = 1

					a_end = {
						requested_product_uid = megaport_mve.mve.product_uid
						inner_vlan            = 100
						vnic_index            = 0
					}

					b_end = {
						requested_product_uid = data.megaport_partner.aws_port.product_uid
					}

					b_end_partner_config = {
					  partner = "aws"
					  aws_config = {
						name          = "%s"
						asn           = 65121
						type          = "private"
						connect_type  = "AWSHC"
						amazon_asn    = 64512
						owner_account = "123456789012"
					  }
					}
				  }

                  `, portName, costCentreName, mveName, mveName, mveName, awsVXCName, awsVXCName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("megaport_vxc.aws_vxc", "product_uid"),
					resource.TestCheckResourceAttr("megaport_vxc.aws_vxc", "b_end_partner_config.aws_config.name", awsVXCName),
				),
			},
		},
	})
}

func (suite *VXCWithCSPsProviderTestSuite) TestFullEcosystem() {
	portName := RandomTestName()
	lagPortName := RandomTestName()
	mcrName := RandomTestName()
	portVXCName := RandomTestName()
	costCentreName := RandomTestName()
	mcrVXCName := RandomTestName()
	awsVXCName := RandomTestName()
	gcpVXCName := RandomTestName()
	azureVXCName := RandomTestName()

	resource.Test(suite.T(), resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "bne_nxt1" {
					name = "NextDC B1"
				  }

				  data "megaport_location" "bne_nxt2" {
					name = "NextDC B2"
				  }

				  data "megaport_location" "bne_pol" {
					name = "Polaris"
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

				  resource "megaport_lag_port" "lag_port" {
			        product_name  = "%s"
					cost_centre = "%s"
			        port_speed  = 10000
			        location_id = data.megaport_location.bne_nxt1.id
			        contract_term_months        = 12
					marketplace_visibility = false
                    lag_count = 1
			      }

				  resource "megaport_port" "port" {
					product_name            = "%s"
					port_speed              = 1000
					location_id             = data.megaport_location.bne_pol.id
					contract_term_months    = 12
					marketplace_visibility  = true
					cost_centre = "%s"
				  }

				  resource "megaport_mcr" "mcr" {
					product_name            = "%s"
					port_speed              = 2500
					location_id             = data.megaport_location.bne_nxt1.id
					contract_term_months    = 1
					asn                      = 64555
				  }

				  resource "megaport_vxc" "port_vxc" {
					product_name           = "%s"
					rate_limit             = 1000
					contract_term_months   = 12

					a_end = {
					  requested_product_uid = megaport_port.port.product_uid
					}

					b_end = {
					  requested_product_uid = megaport_lag_port.lag_port.product_uid
					}
				  }

				  resource "megaport_vxc" "mcr_vxc" {
					product_name           = "%s"
					rate_limit             = 1000
					contract_term_months   = 12

					a_end = {
					  requested_product_uid = megaport_port.port.product_uid
					  ordered_vlan = 181
					}

					b_end = {
					  requested_product_uid = megaport_mcr.mcr.product_uid
					  ordered_vlan = 181
					}
				  }

				  resource "megaport_vxc" "aws_vxc" {
					product_name            = "%s"
					rate_limit              = 1000
					contract_term_months    = 1

					a_end = {
					  requested_product_uid = megaport_mcr.mcr.product_uid
					  ordered_vlan = 191
					}

					b_end = {
					  requested_product_uid = data.megaport_partner.aws_port.product_uid
					}

					b_end_partner_config = {
					  partner = "aws"
					  aws_config = {
						name          = "%s"
						asn           = 64550
						type          = "private"
						connect_type  = "AWS"
						amazon_asn    = 64551
						owner_account = "123456789012"
					  }
					}
				  }

				  resource "megaport_vxc" "gcp_vxc" {
					product_name            = "%s"
					rate_limit              = 1000
					contract_term_months    = 12

					a_end = {
					  requested_product_uid = megaport_mcr.mcr.product_uid
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
					product_name            = "%s"
					rate_limit              = 200
					contract_term_months    = 12

					a_end = {
					  requested_product_uid = megaport_mcr.mcr.product_uid
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
                  `, lagPortName, costCentreName, portName, costCentreName, mcrName, portVXCName, mcrVXCName, awsVXCName, awsVXCName, gcpVXCName, azureVXCName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("megaport_vxc.aws_vxc", "product_uid"),
					resource.TestCheckResourceAttr("megaport_vxc.aws_vxc", "b_end_partner_config.aws_config.name", awsVXCName),
				),
			},
		},
	})
}
