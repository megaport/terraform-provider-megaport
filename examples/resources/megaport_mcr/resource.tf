resource "megaport_mcr" "mcr" {
  product_name         = "Megaport MCR Example"
  port_speed           = 1000
  location_id          = 6
  contract_term_months = 1

  prefix_filter_lists = [{
    description    = "Megaport Example Prefix Filter List"
    address_family = "IPv4"
    entries = [
      {
        action = "permit"
        prefix = "10.0.1.0/24"
        ge     = 24
        le     = 24
      },
      {
        action = "deny"
        prefix = "10.0.2.0/24"
        ge     = 24
        le     = 24
      }
    ]
  }]
}