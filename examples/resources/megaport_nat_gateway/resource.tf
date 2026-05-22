data "megaport_location" "syd" {
  id = 3
}

resource "megaport_nat_gateway" "example" {
  product_name         = "Megaport NAT Gateway Example"
  location_id          = data.megaport_location.syd.id
  speed                = 1000
  session_count        = 32768
  contract_term_months = 1
  diversity_zone       = "red"
  asn                  = 64512

  resource_tags = {
    environment = "production"
  }
}
