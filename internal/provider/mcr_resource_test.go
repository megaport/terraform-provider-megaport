package provider

// func TestAccMegaportMCR_Basic(t *testing.T) {
// 	mcrName := RandomTestName()
// 	resource.Test(t, resource.TestCase{
// 		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
// 		Steps: []resource.TestStep{
// 			{
// 				Config: providerConfig + fmt.Sprintf(`
// 				data "megaport_location" "bne_nxt1" {
// 					name = "NextDC B1"
// 				}
// 				resource "megaport_mcr" "mcr" {
//                     product_name  = "%s"
//                     port_speed  = 1000
//                     location_id = data.megaport_location.bne_nxt1.id
//                     contract_term_months        = 1
// 					market = "AU"
// 					marketplace_visibility = false
//                   }`, mcrName),
// 				Check: resource.ComposeAggregateTestCheckFunc(
// 					resource.TestCheckResourceAttr("megaport_mcr.mcr", "product_name", mcrName),
// 					resource.TestCheckResourceAttr("megaport_mcr.mcr", "port_speed", "1000"),
// 					resource.TestCheckResourceAttr("megaport_mcr.mcr", "contract_term_months", "1"),
// 					resource.TestCheckResourceAttr("megaport_mcr.mcr", "market", "AU"),
// 					resource.TestCheckResourceAttr("megaport_mcr.mcr", "marketplace_visibility", "false"),
// 					resource.TestCheckResourceAttrSet("megaport_mcr.mcr", "product_uid"),
// 				),
// 			},
// 			// ImportState testing
// 			{
// 				ResourceName:                         "megaport_mcr.mcr",
// 				ImportState:                          true,
// 				ImportStateVerify:                    true,
// 				ImportStateVerifyIdentifierAttribute: "product_uid",
// 				ImportStateIdFunc: func(state *terraform.State) (string, error) {
// 					resourceName := "megaport_mcr.mcr"
// 					var rawState map[string]string
// 					for _, m := range state.Modules {
// 						if len(m.Resources) > 0 {
// 							if v, ok := m.Resources[resourceName]; ok {
// 								rawState = v.Primary.Attributes
// 							}
// 						}
// 					}
// 					return rawState["product_uid"], nil
// 				},
// 				ImportStateVerifyIgnore: []string{"last_updated"},
// 			},
// 		},
// 	})
// }
