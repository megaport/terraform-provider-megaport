resource "megaport_nat_gateway_prefix_list" "example" {
  nat_gateway_product_uid = megaport_nat_gateway.example.product_uid
  description             = "Customer routes"
  address_family          = "IPv4"

  entries = [
    {
      action = "permit"
      prefix = "10.0.0.0/8"
      ge     = 24
      le     = 32
    },
    {
      action = "permit"
      prefix = "172.16.0.0/12"
    },
    {
      action = "deny"
      prefix = "192.168.0.0/16"
    },
  ]
}
