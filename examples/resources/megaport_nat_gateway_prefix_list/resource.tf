resource "megaport_nat_gateway" "example" {
  product_name         = "Megaport NAT Gateway Example"
  location_id          = 5
  speed                = 1000
  session_count        = 250000
  contract_term_months = 1
  diversity_zone       = "red"
  asn                  = 64512
}

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
