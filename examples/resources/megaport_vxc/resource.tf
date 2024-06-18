resource "megaport_vxc" "vxc" {
  product_name         = "Megaport VXC"
  rate_limit           = 1000
  contract_term_months = 1

  a_end = {
    requested_product_uid = "614451ba-9869-4a20-8eda-626a5f0d18b4"
  }

  b_end = {
    requested_product_uid = "38526d04-7f22-40c8-88a7-be7dbe0df18e"
  }
}