resource "megaport_port" "test_port" {
  product_name           = "Test Port for IX"
  location_id            = 67
  port_speed             = 1000
  marketplace_visibility = false
  contract_term_months   = 1
}

resource "megaport_ix" "test_ix" {
  name                  = "Test IX Connection"
  requested_product_uid = megaport_port.test_port.product_uid
  network_service_type  = "Sydney IX"
  asn                   = 65000
  mac_address           = "00:11:22:33:44:55"
  rate_limit            = 500
  vlan                  = 2000
  shutdown              = false
}