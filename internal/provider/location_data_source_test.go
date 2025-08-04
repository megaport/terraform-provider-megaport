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
				),
			},
			// Search by Lag Port Name
			{
				Config: providerConfig + fmt.Sprintf(`data "megaport_location" "test_location" {
					name = "%s"
				}`, LagPortTestLocationName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "name", LagPortTestLocationName),
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
				),
			},
			// Search by MCR Name
			{
				Config: providerConfig + fmt.Sprintf(`data "megaport_location" "test_location" {
					name = "%s"
				}`, MCRTestLocationName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "name", MCRTestLocationName),
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
				),
			},
			// Search by MVE Name
			{
				Config: providerConfig + fmt.Sprintf(`data "megaport_location" "test_location" {
					name = "%s"
				}`, MVETestLocationName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "name", MVETestLocationName),
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
				),
			},
			// Search by Single Port Name
			{
				Config: providerConfig + fmt.Sprintf(`data "megaport_location" "test_location" {
					name = "%s"
				}`, SinglePortTestLocationName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "name", SinglePortTestLocationName),
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
				),
			},
			// Search by VXC Name One
			{
				Config: providerConfig + fmt.Sprintf(`data "megaport_location" "test_location" {
					name = "%s"
				}`, VXCLocationNameOne),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "name", VXCLocationNameOne),
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
				),
			},
			// Search by VXC Name Two
			{
				Config: providerConfig + fmt.Sprintf(`data "megaport_location" "test_location" {
					name = "%s"
				}`, VXCLocationNameTwo),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "name", VXCLocationNameTwo),
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
				),
			},
			// Search by VXC Name Three
			{
				Config: providerConfig + fmt.Sprintf(`data "megaport_location" "test_location" {
					name = "%s"
				}`, VXCLocationNameThree),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "name", VXCLocationNameThree),
				),
			},
		},
	})
}
