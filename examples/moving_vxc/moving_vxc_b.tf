provider "megaport" {
  environment           = "staging"
  access_key            = "access_key"
  secret_key            = "secret_Key"
  accept_purchase_terms = true
}

data "megaport_location" "loc" {
  id = 5 # NextDC Brisbane B1
}

resource "megaport_port" "port_1" {
  product_name           = "Port 1"
  port_speed             = 1000
  location_id            = data.megaport_location.loc.id
  contract_term_months   = 12
  marketplace_visibility = false
}

resource "megaport_port" "port_2" {
  product_name           = "Port 2"
  port_speed             = 1000
  location_id            = data.megaport_location.loc.id
  contract_term_months   = 12
  marketplace_visibility = false
}

resource "megaport_port" "port_3" {
  product_name           = "Port 3"
  port_speed             = 1000
  location_id            = data.megaport_location.loc.id
  contract_term_months   = 12
  marketplace_visibility = false
}

resource "megaport_port" "port_4" {
  product_name           = "Port 4"
  port_speed             = 1000
  location_id            = data.megaport_location.loc.id
  contract_term_months   = 12
  marketplace_visibility = false
}

resource "megaport_vxc" "vxc" {
  product_name         = "Example VXC"
  rate_limit           = 500
  contract_term_months = 12
  cost_centre          = "Example Cost Centre"

  a_end = {
    requested_product_uid = megaport_port.port_3.product_uid
  }

  b_end = {
    requested_product_uid = megaport_port.port_4.product_uid
  }
}