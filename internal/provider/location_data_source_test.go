package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestLagPortLocation(t *testing.T) {
	t.Parallel()
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Search by Lag Port Name
			{
				Config: providerConfig + fmt.Sprintf(`data "megaport_location" "test_location" {
					name = "%s"
				}`, LagPortTestLocation),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "name", LagPortTestLocation),
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "site_code", LagPortTestSiteCode),
				),
			},
			// Search by Lag Port Site Code
			{
				Config: providerConfig + fmt.Sprintf(`data "megaport_location" "test_location" {
					site_code = "%s"
				}`, LagPortTestSiteCode),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "name", LagPortTestLocation),
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "site_code", LagPortTestSiteCode),
				),
			},
		},
	})
}

func TestMCRLocation(t *testing.T) {
	t.Parallel()
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Search by MCR Name
			{
				Config: providerConfig + fmt.Sprintf(`data "megaport_location" "test_location" {
					name = "%s"
				}`, MCRTestLocation),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "name", MCRTestLocation),
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "site_code", MCRTestSiteCode),
				),
			},
			// Search by MCR Site Code
			{
				Config: providerConfig + fmt.Sprintf(`data "megaport_location" "test_location" {
					site_code = "%s"
				}`, MCRTestSiteCode),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "name", MCRTestLocation),
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "site_code", MCRTestSiteCode),
				),
			},
		},
	})
}

func TestMVELocation(t *testing.T) {
	t.Parallel()
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Search by MVE Name
			{
				Config: providerConfig + fmt.Sprintf(`data "megaport_location" "test_location" {
					name = "%s"
				}`, MVETestLocation),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "name", MVETestLocation),
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "site_code", MVETestSiteCode),
				),
			},
			// Search by MVE Site Code
			{
				Config: providerConfig + fmt.Sprintf(`data "megaport_location" "test_location" {
					site_code = "%s"
				}`, MVETestSiteCode),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "name", MVETestLocation),
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "site_code", MVETestSiteCode),
				),
			},
		},
	})
}

func TestSinglePortLocation(t *testing.T) {
	t.Parallel()
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Search by Single Port Name
			{
				Config: providerConfig + fmt.Sprintf(`data "megaport_location" "test_location" {
					name = "%s"
				}`, SinglePortTestLocation),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "name", SinglePortTestLocation),
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "site_code", SinglePortTestSiteCode),
				),
			},
			// Search by Single Port Site Code
			{
				Config: providerConfig + fmt.Sprintf(`data "megaport_location" "test_location" {
					site_code = "%s"
				}`, SinglePortTestSiteCode),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "name", SinglePortTestLocation),
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "site_code", SinglePortTestSiteCode),
				),
			},
		},
	})
}

func TestVXCLocationOne(t *testing.T) {
	t.Parallel()
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Search by VXC Name One
			{
				Config: providerConfig + fmt.Sprintf(`data "megaport_location" "test_location" {
					name = "%s"
				}`, VXCLocationOne),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "name", VXCLocationOne),
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "site_code", VXCLocationOneSiteCode),
				),
			},
			// Search by VXC Site Code One
			{
				Config: providerConfig + fmt.Sprintf(`data "megaport_location" "test_location" {
					site_code = "%s"
				}`, VXCLocationOneSiteCode),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "name", VXCLocationOne),
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "site_code", VXCLocationOneSiteCode),
				),
			},
		},
	})
}

func TestVXCLocationTwo(t *testing.T) {
	t.Parallel()
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Search by VXC Name Two
			{
				Config: providerConfig + fmt.Sprintf(`data "megaport_location" "test_location" {
					name = "%s"
				}`, VXCLocationTwo),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "name", VXCLocationTwo),
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "site_code", VXCLocationTwoSiteCode),
				),
			},
			// Search by VXC Site Code Two
			{
				Config: providerConfig + fmt.Sprintf(`data "megaport_location" "test_location" {
					site_code = "%s"
				}`, VXCLocationTwoSiteCode),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "name", VXCLocationTwo),
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "site_code", VXCLocationTwoSiteCode),
				),
			},
		},
	})
}

func TestVXCLocationThree(t *testing.T) {
	t.Parallel()
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Search by VXC Name Three
			{
				Config: providerConfig + fmt.Sprintf(`data "megaport_location" "test_location" {
					name = "%s"
				}`, VXCLocationThree),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "name", VXCLocationThree),
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "site_code", VXCLocationThreeSiteCode),
				),
			},
			// Search by VXC Site Code Three
			{
				Config: providerConfig + fmt.Sprintf(`data "megaport_location" "test_location" {
					site_code = "%s"
				}`, VXCLocationThreeSiteCode),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "name", VXCLocationThree),
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "site_code", VXCLocationThreeSiteCode),
				),
			},
		},
	})
}
