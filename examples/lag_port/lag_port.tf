provider "megaport" {
  environment            = "staging"
  access_key             = "access_key"
  secret_key             = "secret_Key"
  accept_purchase_terms  = true
}

data "megaport_location" "bne_nxt1" {
  name = "NextDC B1"
}

resource "megaport_lag_port" "lag_port" {
  product_name             = "%s"
  port_speed               = 10000
  location_id              = data.megaport_location.bne_nxt1.id
  contract_term_months     = 1
  market                   = "AU"
  marketplace_visibility   = false
  lag_count                = 3
}
