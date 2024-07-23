provider "megaport" {
  environment           = "staging"
  access_key            = "access_key"
  secret_key            = "secret_Key"
  accept_purchase_terms = true
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
  product_name         = "Megaport Example MCR A-End"
  location_id          = data.megaport_location.bne_nxt1.id
  contract_term_months = 1
  port_speed           = 5000
  asn                  = 64555
  cost_centre          = "MCR Example"
}

resource "megaport_mcr" "mcr_2" {
  product_name         = "Megaport Example MCR A-End 2"
  location_id          = data.megaport_location.bne_nxt1.id
  contract_term_months = 1
  port_speed           = 10000
  asn                  = 64555
  cost_centre          = "MCR Example 2"
}

resource "megaport_vxc" "aws_vxc" {
  product_name         = "Megaport VXC Example - AWS"
  rate_limit           = 1000
  contract_term_months = 1

  a_end = {
    requested_product_uid = megaport.mcr.mcr.product_uid
    ordered_vlan          = 2191
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