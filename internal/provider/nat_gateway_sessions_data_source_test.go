package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccMegaportNATGatewaySessionsDataSource_Basic(t *testing.T) {
	t.Parallel()
	defer acquireAccTestSlot(t)()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
data "megaport_nat_gateway_sessions" "this" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrWith(
						"data.megaport_nat_gateway_sessions.this",
						"sessions.#",
						func(v string) error {
							if v == "0" {
								return fmt.Errorf("expected non-empty sessions matrix, got 0 entries")
							}
							return nil
						},
					),
				),
			},
		},
	})
}
