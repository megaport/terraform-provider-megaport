package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	LagPortTestLocationID    = 5
	MCRTestLocationID        = 65
	MVETestLocationID        = 65
	SinglePortTestLocationID = 5
	VXCLocationIDOne         = 4
	VXCLocationIDTwo         = 3
	VXCLocationIDThree       = 23

	LagPortTestSiteCode      = "bne-nxt1"
	MCRTestSiteCode          = "sjc-tx2"
	MVETestSiteCode          = "sjc-tx2"
	SinglePortTestSiteCode   = "bne-nxt1"
	VXCLocationOneSiteCode   = "mel-nxt1"
	VXCLocationTwoSiteCode   = "syd-gs"
	VXCLocationThreeSiteCode = "mel-mdc"

	LagPortTestLocationName    = "NextDC B1"
	MCRTestLocationName        = "Digital Realty Silicon Valley SJC34 (SCL2)"
	MVETestLocationName        = "Digital Realty Silicon Valley SJC34 (SCL2)"
	SinglePortTestLocationName = "NextDC B1"
	VXCLocationNameOne         = "NextDC M1"
	VXCLocationNameTwo         = "Global Switch Sydney West"
	VXCLocationNameThree       = "5GN Melbourne Data Centre (MDC)"
)

func TestLagPortLocation(t *testing.T) {
	t.Parallel()
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Search by Lag Port ID
			{
				Config: providerConfig + fmt.Sprintf(`data "megaport_location" "test_location" {
					id = "%d"
				}`, LagPortTestLocationID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "id", fmt.Sprintf("%d", LagPortTestLocationID)),
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "name", LagPortTestLocationName),
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "site_code", LagPortTestSiteCode),
				),
			},
			// Search by Lag Port Name
			{
				Config: providerConfig + fmt.Sprintf(`data "megaport_location" "test_location" {
					name = "%s"
				}`, LagPortTestLocationName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "name", LagPortTestLocationName),
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "site_code", LagPortTestSiteCode),
				),
			},
			// Search by Lag Port Site Code
			{
				Config: providerConfig + fmt.Sprintf(`data "megaport_location" "test_location" {
					site_code = "%s"
				}`, LagPortTestSiteCode),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "name", LagPortTestLocationName),
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
			// Search by MCR ID
			{
				Config: providerConfig + fmt.Sprintf(`data "megaport_location" "test_location" {
					id = "%d"
				}`, MCRTestLocationID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "id", fmt.Sprintf("%d", MCRTestLocationID)),
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "name", MCRTestLocationName),
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "site_code", MCRTestSiteCode),
				),
			},
			// Search by MCR Name
			{
				Config: providerConfig + fmt.Sprintf(`data "megaport_location" "test_location" {
					name = "%s"
				}`, MCRTestLocationName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "name", MCRTestLocationName),
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "site_code", MCRTestSiteCode),
				),
			},
			// Search by MCR Site Code
			{
				Config: providerConfig + fmt.Sprintf(`data "megaport_location" "test_location" {
					site_code = "%s"
				}`, MCRTestSiteCode),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "name", MCRTestLocationName),
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
			// Search by MVE ID
			{
				Config: providerConfig + fmt.Sprintf(`data "megaport_location" "test_location" {
								id = "%d"
							}`, MVETestLocationID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "id", fmt.Sprintf("%d", MVETestLocationID)),
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "name", MVETestLocationName),
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "site_code", MVETestSiteCode),
				),
			},
			// Search by MVE Name
			{
				Config: providerConfig + fmt.Sprintf(`data "megaport_location" "test_location" {
					name = "%s"
				}`, MVETestLocationName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "name", MVETestLocationName),
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "site_code", MVETestSiteCode),
				),
			},
			// Search by MVE Site Code
			{
				Config: providerConfig + fmt.Sprintf(`data "megaport_location" "test_location" {
					site_code = "%s"
				}`, MVETestSiteCode),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "name", MVETestLocationName),
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
			// Search by Single Port ID
			{
				Config: providerConfig + fmt.Sprintf(`data "megaport_location" "test_location" {
								id = "%d"
							}`, SinglePortTestLocationID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "id", fmt.Sprintf("%d", SinglePortTestLocationID)),
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "name", SinglePortTestLocationName),
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "site_code", SinglePortTestSiteCode),
				),
			},
			// Search by Single Port Name
			{
				Config: providerConfig + fmt.Sprintf(`data "megaport_location" "test_location" {
					name = "%s"
				}`, SinglePortTestLocationName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "name", SinglePortTestLocationName),
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "site_code", SinglePortTestSiteCode),
				),
			},
			// Search by Single Port Site Code
			{
				Config: providerConfig + fmt.Sprintf(`data "megaport_location" "test_location" {
					site_code = "%s"
				}`, SinglePortTestSiteCode),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "name", SinglePortTestLocationName),
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
			// Search by VXC ID
			{
				Config: providerConfig + fmt.Sprintf(`data "megaport_location" "test_location" {
								id = "%d"
							}`, VXCLocationIDOne),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "id", fmt.Sprintf("%d", VXCLocationIDOne)),
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "name", VXCLocationNameOne),
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "site_code", VXCLocationOneSiteCode),
				),
			},
			// Search by VXC Name One
			{
				Config: providerConfig + fmt.Sprintf(`data "megaport_location" "test_location" {
					name = "%s"
				}`, VXCLocationNameOne),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "name", VXCLocationNameOne),
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "site_code", VXCLocationOneSiteCode),
				),
			},
			// Search by VXC Site Code One
			{
				Config: providerConfig + fmt.Sprintf(`data "megaport_location" "test_location" {
					site_code = "%s"
				}`, VXCLocationOneSiteCode),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "name", VXCLocationNameOne),
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
			// Search by VXC ID
			{
				Config: providerConfig + fmt.Sprintf(`data "megaport_location" "test_location" {
								id = "%d"
							}`, VXCLocationIDTwo),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "id", fmt.Sprintf("%d", VXCLocationIDTwo)),
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "name", VXCLocationNameTwo),
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "site_code", VXCLocationTwoSiteCode),
				),
			},
			// Search by VXC Name Two
			{
				Config: providerConfig + fmt.Sprintf(`data "megaport_location" "test_location" {
					name = "%s"
				}`, VXCLocationNameTwo),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "name", VXCLocationNameTwo),
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "site_code", VXCLocationTwoSiteCode),
				),
			},
			// Search by VXC Site Code Two
			{
				Config: providerConfig + fmt.Sprintf(`data "megaport_location" "test_location" {
					site_code = "%s"
				}`, VXCLocationTwoSiteCode),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "name", VXCLocationNameTwo),
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
			// Search by VXC ID
			{
				Config: providerConfig + fmt.Sprintf(`data "megaport_location" "test_location" {
								id = "%d"
							}`, VXCLocationIDThree),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "id", fmt.Sprintf("%d", VXCLocationIDThree)),
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "name", VXCLocationNameThree),
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "site_code", VXCLocationThreeSiteCode),
				),
			},
			// Search by VXC Name Three
			{
				Config: providerConfig + fmt.Sprintf(`data "megaport_location" "test_location" {
					name = "%s"
				}`, VXCLocationNameThree),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "name", VXCLocationNameThree),
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "site_code", VXCLocationThreeSiteCode),
				),
			},
			// Search by VXC Site Code Three
			{
				Config: providerConfig + fmt.Sprintf(`data "megaport_location" "test_location" {
					site_code = "%s"
				}`, VXCLocationThreeSiteCode),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "name", VXCLocationNameThree),
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "site_code", VXCLocationThreeSiteCode),
				),
			},
		},
	})
}
