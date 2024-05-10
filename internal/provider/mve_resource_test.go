package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccMegaportMVE_Basic(t *testing.T) {
	mveName := RandomTestName()
	mveKey := RandomTestName()
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
				data "megaport_location" "bne_nxt1" {
					name = "NextDC B1"
				}
				resource "megaport_mve" "mve" {
                    product_name  = "%s"
                    location_id = data.megaport_location.bne_nxt1.id
                    contract_term_months        = 1

                    vendor_config = {
                        vendor = "aruba"
                        product_size = "MEDIUM"
                        image_id = 23
						account_name = "%s"
						account_key = "%s"
						system_tag = "Preconfiguration-aruba-test-1"
                    }
                  }`, mveName, mveName, mveKey),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("megaport_mve.mve", "product_name", mveName),
					resource.TestCheckResourceAttrSet("megaport_mve.mve", "product_uid"),
				),
			},
		},
	})
}
