provider "megaport" {
  environment           = "staging"
  access_key            = "access_key"
  secret_key            = "secret_Key"
  accept_purchase_terms = true
}

data "megaport_location" "bne_nxt1" {
  name = "NextDC B1"
}

resource "megaport_mve" "mve" {
  product_name         = "Megaport Versa MVE Example"
  location_id          = data.megaport_location.bne_nxt1.id
  contract_term_months = 1

  vendor_config = {
    vendor             = "versa"
    product_size       = "LARGE"
    image_id           = 20
    director_address   = "director1.versa.com"
    controller_address = "controller1.versa.com"
    local_auth         = "SDWAN-Branch@Versa.com"
    remote_auth        = "Controller-1-staging@Versa.com"
    serial_number      = "Megaport-Hub1"
  }
}
