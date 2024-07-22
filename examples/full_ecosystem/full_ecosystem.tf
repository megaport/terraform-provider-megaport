provider "megaport" {
  environment           = "staging"
  access_key            = "access_key"
  secret_key            = "secret_Key"
  accept_purchase_terms = true
}

data "megaport_location" "bne_nxt1" {
  name = "NextDC B1"
}

data "megaport_location" "bne_nxt2" {
  name = "NextDC B2"
}

data "megaport_location" "syd_gs" {
  name = "Global Switch Sydney West"
}

data "megaport_partner" "aws_port" {
  connect_type = "AWS"
  company_name = "AWS"
  product_name = "Asia Pacific (Sydney) (ap-southeast-2)"
  location_id  = data.megaport_location.syd_gs.id
}

resource "megaport_port" "port" {
  product_name           = "Megaport Port Example"
  port_speed             = 1000
  location_id            = data.megaport_location.bne_nxt1.id
  contract_term_months   = 12
  marketplace_visibility = false
  cost_centre            = "Megaport Single Port Example"
}

resource "megaport_lag_port" "lag_port" {
  product_name           = "Megaport Lag Port Example"
  port_speed             = 10000
  location_id            = data.megaport_location.bne_nxt2.id
  contract_term_months   = 12
  marketplace_visibility = false
  lag_count              = 1
  cost_centre            = "Lag Port Example"
}

resource "megaport_mcr" "mcr" {
  product_name         = "Megaport MCR Example"
  port_speed           = 2500
  location_id          = data.megaport_location.bne_nxt1.id
  contract_term_months = 1
  asn                  = 64555
}

resource "megaport_vxc" "port_vxc" {
  product_name         = "Megaport Port-to-Port VXC"
  rate_limit           = 1000
  contract_term_months = 12

  a_end = {
    requested_product_uid = megaport_port.port.product_uid
  }

  b_end = {
    requested_product_uid = megaport_lag_port.lag_port.product_uid
  }
}

resource "megaport_vxc" "mcr_vxc" {
  product_name         = "Megaport Port-to-MCR VXC"
  rate_limit           = 1000
  contract_term_months = 12

  a_end = {
    requested_product_uid = megaport_port.port.product_uid
    ordered_vlan          = 181
  }

  b_end = {
    requested_product_uid = megaport_mcr.mcr.product_uid
    ordered_vlan          = 181
  }
}

resource "megaport_vxc" "aws_vxc" {
  product_name         = "Megaport VXC Example - AWS"
  rate_limit           = 1000
  contract_term_months = 1

  a_end = {
    requested_product_uid = megaport.mcr.mcr.product_uid
    ordered_vlan          = 191
  }

  b_end = {
    requested_product_uid = data.megaport_partner.aws_port.product_uid
  }

  b_end_partner_config = {
    partner = "aws"
    aws_config = {
      name          = "Megaport VXC Example - AWS"
      asn           = 64550
      type          = "private"
      connect_type  = "AWS"
      amazon_asn    = 64551
      owner_account = "123456789012"
    }
  }
}

resource "megaport_vxc" "gcp_vxc" {
  product_name         = "Megaport VXC Example - Google"
  rate_limit           = 1000
  contract_term_months = 12

  a_end = {
    requested_product_uid = megaport_mcr.mcr.product_uid
    ordered_vlan          = 182
  }

  b_end = {}

  b_end_partner_config = {
    partner = "google"
    google_config = {
      pairing_key = "7e51371e-72a3-40b5-b844-2e3efefaee59/australia-southeast1/2"
    }
  }
}

resource "megaport_vxc" "azure_vxc" {
  product_name         = "Megaport VXC Example - Azure"
  rate_limit           = 200
  contract_term_months = 12

  a_end = {
    requested_product_uid = megaport_mcr.mcr.product_uid
    ordered_vlan          = 0
  }

  b_end = {}

  b_end_partner_config = {
    partner = "azure"
    azure_config = {
      port_choice = "primary"
      service_key = "1b2329a5-56dc-45d0-8a0d-87b706297777"
    }
  }
}
