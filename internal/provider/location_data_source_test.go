package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestDynamicLocation(t *testing.T) {
	t.Parallel()
	locID := findAnyActiveLocationID(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`data "megaport_location" "test_location" {
					id = "%d"
				}`, locID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.megaport_location.test_location", "id", fmt.Sprintf("%d", locID)),
				),
			},
		},
	})
}
