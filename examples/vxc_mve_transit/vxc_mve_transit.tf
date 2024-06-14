provider "megaport" {
  environment           = "staging"
  access_key            = "access_key"
  secret_key            = "secret_key"
  accept_purchase_terms = true
}

data "megaport_location" "bne_nxt1" {
  name = "NextDC B1"
}

data "megaport_location" "syd_gs" {
  name = "Global Switch Sydney West"
}

data "megaport_partner" "internet_port" {
  connect_type = "TRANSIT"
  company_name = "Networks"
  product_name = "Megaport Internet"
  location_id  = data.megaport_location.syd_gs.id
}

resource "megaport_port" "port" {
  product_name           = "Megaport Example Port"
  port_speed             = 1000
  location_id            = data.megaport_location.bne_nxt1.id
  contract_term_months   = 12
  marketplace_visibility = true
  cost_centre            = "Megaport Example Port"
}

resource "megaport_mve" "mve" {
  product_name         = "Megaport Aruba MVE"
  location_id          = data.megaport_location.bne_nxt1.id
  contract_term_months = 1

  vnics = [
    {
      description = "Data Plane"
    },
    {
      description = "Management Plane"
    },
    {
      description = "Control Plane"
    }
  ]

  vendor_config = {
    vendor       = "aruba"
    product_size = "MEDIUM"
    image_id     = 23
    account_name = "Megaport Aruba MVE"
    account_key  = "Megaport Aruba MVE"
    system_tag   = "Preconfiguration-aruba-test-1"
  }
}

resource "megaport_vxc" "transit_vxc" {
  product_name         = "Transit VXC Example"
  rate_limit           = 100
  contract_term_months = 1

  a_end = {
    requested_product_uid = megaport_mve.mve.product_uid
    vnic_index            = 2
  }

  b_end = {
    requested_product_uid = data.megaport_partner.internet_port.product_uid
  }

  b_end_partner_config = {
    partner = "transit"
  }
}
