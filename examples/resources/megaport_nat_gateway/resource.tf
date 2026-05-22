resource "megaport_nat_gateway" "example" {
  product_name         = "Megaport NAT Gateway Example"
  location_id          = 6
  speed                = 1000
  session_count        = 32768
  contract_term_months = 1
  diversity_zone       = "red"
  asn                  = 64512

  resource_tags = {
    environment = "production"
  }
}
