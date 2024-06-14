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

  prefix_filter_list = {
    description    = "Megaport Prefix Filter List"
    address_family = "IPv4"
    entries = [
      {
        action = "permit"
        prefix = "10.0.1.0/24"
        ge     = 24
        le     = 24
      },
      {
        action = "deny"
        prefix = "10.0.2.0/24"
        ge     = 24
        le     = 24
      }
    ]
  }
}
