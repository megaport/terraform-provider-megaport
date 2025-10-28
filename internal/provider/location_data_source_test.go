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
				),
			},
		},
	})
}
