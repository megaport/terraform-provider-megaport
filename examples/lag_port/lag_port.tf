provider "megaport" {
  environment           = "staging"
  access_key            = "access_key"
  secret_key            = "secret_Key"
  accept_purchase_terms = true
}

data "megaport_location" "bne_nxt1" {
  id = 5 # NextDC Brisbane B1
}

resource "megaport_lag_port" "lag_port" {
  product_name           = "Megaport Lag Port Example"
  port_speed             = 10000
  location_id            = data.megaport_location.bne_nxt1.id
  contract_term_months   = 1
  marketplace_visibility = false
  lag_count              = 1
  cost_centre            = "Lag Port Example"
}
