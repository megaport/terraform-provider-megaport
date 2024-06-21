resource "megaport_port" "port" {
  product_name           = "Megaport Port Example"
  port_speed             = 1000
  location_id            = 6
  contract_term_months   = 1
  marketplace_visibility = false
  cost_centre            = "Megaport Single Port Example"
}