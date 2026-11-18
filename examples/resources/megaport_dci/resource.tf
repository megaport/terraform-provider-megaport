resource "megaport_dci" "dci" {
  name                 = "Megaport DCI Example"
  a_end_location_id    = 6
  b_end_location_id    = 3
  speed                = 10000
  contract_term_months = 12
  auto_renew           = true
  cost_centre          = "Megaport DCI Example"
}
