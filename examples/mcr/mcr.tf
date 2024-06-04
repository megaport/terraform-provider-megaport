provider "megaport" {
  environment           = "staging"
  access_key            = "access_key"
  secret_key            = "secret_Key"
  accept_purchase_terms = true
}

data "megaport_location" "bne_nxt1" {
  name = "NextDC B1"
}

resource "megaport_mcr" "mcr" {
  product_name         = "Megaport MCR Example"
  port_speed           = 1000
  location_id          = data.megaport_location.bne_nxt1.id
  contract_term_months = 1
}
