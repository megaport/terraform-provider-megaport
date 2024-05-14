provider "megaport" {
  environment            = "staging"
  access_key             = "access_key"
  secret_key             = "secret_Key"
  accept_purchase_terms  = true
}

data "megaport_location" "loc" {
  name = "NextDC B1"
}

resource "megaport_port" "port_1" {
  product_name            = "Megaport Port A-End"
  port_speed              = 1000
  location_id             = data.megaport_location.loc.id
  contract_term_months    = 1
  market                  = "AU"
  marketplace_visibility  = false
}

resource "megaport_port" "port_2" {
  product_name            = "Megaport Port B-End"
  port_speed              = 1000
  location_id             = data.megaport_location.loc.id
  contract_term_months    = 1
  market                  = "AU"
  marketplace_visibility  = false
}

resource "megaport_vxc" "vxc" {
  product_name           = "Megaport VXC"
  rate_limit             = 1000
  contract_term_months   = 1
  port_uid               = megaport_port.port_1.product_uid

  a_end = {}

  b_end = {
    ordered_product_uid = megaport_port.port_2.product_uid
  }
}
