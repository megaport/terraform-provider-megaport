provider "megaport" {
  environment            = "staging"
  access_key             = "access_key"
  secret_key             = "secret_Key"
  accept_purchase_terms  = true
}

data "megaport_location" "bne_nxt1" {
  name = "NextDC B1"
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

resource "megaport_mcr" "mcr" {
  product_name             = "Megaport Example MCR A-End"
  location_id              = data.megaport_location.bne_nxt1.id
  marketplace_visibility   = false
  market                   = "AU"
  contract_term_months     = 1
  port_speed               = 5000
  asn                      = 64555
}

resource "megaport_vxc" "aws_vxc" {
  product_name            = "Megaport VXC Example - AWS"
  rate_limit              = 1000
  port_uid                = megaport_mcr.mcr.product_uid
  contract_term_months    = 1

  a_end = {
    ordered_vlan = 2191
  }

  b_end = {
    ordered_product_uid = data.megaport_partner.aws_port.product_uid
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
  product_name            = "Megaport VXC Example - Google"
  rate_limit              = 1000
  contract_term_months    = 1
  port_uid                = megaport_mcr.mcr.product_uid

  a_end = {
    ordered_vlan = 182
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
  product_name            = "Megaport VXC Example - Azure"
  rate_limit              = 200
  contract_term_months    = 1
  port_uid                = megaport_mcr.mcr.product_uid

  a_end = {
    ordered_vlan = 0
  }

  b_end = {}

  b_end_partner_config = {
    partner = "azure"
    azure_config = {
      service_key = "1b2329a5-56dc-45d0-8a0d-87b706297777"
    }
  }
}
