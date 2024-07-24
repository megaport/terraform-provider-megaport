provider "megaport" {
  environment           = "staging"
  access_key            = "access_key"
  secret_key            = "secret_Key"
  accept_purchase_terms = true
}

data "megaport_location" "loc" {
  name = "NextDC B1"
}

resource "megaport_port" "port_1" {
  product_name           = "Megaport Port A-End"
  port_speed             = 1000
  location_id            = data.megaport_location.loc.id
  contract_term_months   = 1
  marketplace_visibility = false
  cost_centre            = "Megaport Port A-End"
}

resource "megaport_port" "port_2" {
  product_name           = "Megaport Port B-End"
  port_speed             = 1000
  location_id            = data.megaport_location.loc.id
  contract_term_months   = 1
  marketplace_visibility = false
  cost_centre            = "Megaport Port B-End"
}

resource "megaport_vxc" "vxc" {
  product_name         = "Megaport VXC"
  rate_limit           = 1000
  contract_term_months = 1

  a_end = {
    requested_product_uid = megaport_port.port_1.product_uid
    ordered_vlan          = 200
  }

  b_end = {
    requested_product_uid = megaport_port.port_2.product_uid
    ordered_vlan          = 201
  }
}
