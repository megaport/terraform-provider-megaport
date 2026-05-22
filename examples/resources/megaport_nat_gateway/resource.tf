resource "megaport_nat_gateway" "example" {
  product_name         = "Megaport NAT Gateway Example"
  location_id          = 5
  speed                = 1000
  session_count        = 250000
  contract_term_months = 1
  diversity_zone       = "red"

  resource_tags = {
    environment = "production"
  }
}
