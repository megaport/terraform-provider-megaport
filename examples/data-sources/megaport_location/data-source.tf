data "megaport_location" "my_location_1" {
  name = "NextDC B1"
}

data "megaport_location" "my_location_2" {
  site_code = "bne_nxt1"
}

data "megaport_location" "my_location_3" {
  id = 5
}