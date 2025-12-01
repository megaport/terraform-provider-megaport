provider "megaport" {
  environment           = "staging"
  access_key            = "access_key"
  secret_key            = "secret_Key"
  accept_purchase_terms = true
}

data "megaport_location" "bne_nxt1" {
  id = 5 # NextDC Brisbane B1
}

data "megaport_location" "syd_gs" {
  id = 3 # Global Switch Sydney West
}

data "megaport_partner" "aws_port" {
  connect_type = "AWSHC"
  company_name = "AWS"
  product_name = "Asia Pacific (Sydney) (ap-southeast-2)"
  location_id  = data.megaport_location.syd_gs.id
}

resource "megaport_mcr" "mcr" {
  product_name         = "Megaport Example MCR"
  location_id          = data.megaport_location.bne_nxt1.id
  contract_term_months = 1
  port_speed           = 5000
  asn                  = 64555
  cost_centre          = "MCR Example"
}

resource "megaport_mcr_prefix_filter_list" "example" {
  mcr_id         = megaport_mcr.mcr.product_uid
  description    = "Megaport Example Prefix Filter List"
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

resource "megaport_vxc" "aws_vxc" {
  product_name         = "Megaport Example VXC - AWS"
  rate_limit           = 1000
  contract_term_months = 1

  a_end = {
    requested_product_uid = megaport_mcr.mcr.product_uid
    ordered_vlan          = 0
  }

  a_end_partner_config = {
    partner = "vrouter"
    vrouter_config = {
      interfaces = [
        {
          ip_addresses     = ["10.0.0.1/30"]
          nat_ip_addresses = ["10.0.0.1"]
          bfd = {
            tx_interval = 500
            rx_interval = 400
            multiplier  = 5
          }
          bgp_connections = [
            {
              peer_asn         = 64512
              local_ip_address = "10.0.0.1"
              peer_ip_address  = "10.0.0.2"
              password         = "notARealPassword"
              shutdown         = false
              description      = "BGP Connection 1"
              med_in           = 100
              med_out          = 100
              bfd_enabled      = true
              export_policy    = "deny"
              permit_export_to = ["10.0.1.2"]
              import_whitelist = megaport_mcr_prefix_filter_list.example.description
            }
          ]
        }
      ]
    }
  }

  b_end = {
    requested_product_uid = data.megaport_partner.aws_port.product_uid
  }

  b_end_partner_config = {
    partner = "aws"
    aws_config = {
      name          = "Megaport Example VXC - AWS"
      asn           = 64550
      type          = "private"
      connect_type  = "AWSHC"
      amazon_asn    = 64551
      owner_account = "684021030471"
    }
  }
}
