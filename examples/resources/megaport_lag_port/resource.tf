resource "megaport_lag_port" "lag_port" {
  product_name           = "Megaport Lag Port Example"
  port_speed             = 10000
  location_id            = 6
  contract_term_months   = 1
  marketplace_visibility = false
  lag_count              = 1
  cost_centre            = "Lag Port Example"
}