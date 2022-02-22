---
page_title: "MCR with BGP VXCs"
subcategory: "Examples"
---

# MCR with BGP VXC's
This will provision an MCR (Megaport Cloud Router) connected to AWS over a Hosted Connection with BGP and BFD configuration.

Replace the `username`, `password` and optional `mfa_otp_key` with your own credentials.

This configuration will deploy on the staging environment. To use this on production, valid CSP attributes are required:
+ `megaport_aws_connection.amazon_account`

```
terraform {
  required_providers {
    megaport = {
      source  = "megaport/megaport"
      version = ">=0.1.4"
    }
  }
}

provider "megaport" {
  username              = "my.test.user@example.org"
  password              = "n0t@re4lPassw0rd"
  mfa_otp_key           = "ABCDEFGHIJK01234"
  accept_purchase_terms = true
  delete_ports          = true
  environment           = "staging"
}

data "megaport_location" "bne_nxt1" {
  name    = "NextDC B1"
  has_mcr = true
}

data "megaport_location" "syd_sy3" {
  name = "Equinix SY3"
}

data "megaport_partner_port" "aws_port" {
  connect_type = "AWSHC"
  company_name = "AWS"
  product_name = "Asia Pacific (Sydney) (ap-southeast-2) [DZ-BLUE]"
  location_id  = data.megaport_location.syd_sy3.id
}

resource "megaport_mcr" "mcr" {
  mcr_name    = "Terraform Example - MCR"
  location_id = data.megaport_location.bne_nxt1.id

  router {
    port_speed    = 2500
    requested_asn = 64555
  }
}

resource "megaport_aws_connection" "aws_vxc" {
  vxc_name   = "Terraform Example - AWSHC VXC"
  rate_limit = 1000

  a_end {
    port_id        = megaport_mcr.mcr.id
    requested_vlan = 0
  }

  a_end_mcr_configuration {
    ip_addresses = ["10.0.0.1/30"]
    nat_ip_addresses = ["10.0.0.1"]
    
    bfd_configuration {
      tx_interval = 500
      rx_interval = 400
      multiplier  = 5
    }
    
    bgp_connection {
      peer_asn         = 64512
      local_ip_address = "10.0.0.1"
      peer_ip_address  = "10.0.0.2"
      password         = "notARealPassword"
      shutdown         = false
      med_in           = 100
      med_out          = 100
      bfd_enabled      = true
    }
  }

  csp_settings {
    requested_product_id = data.megaport_partner_port.aws_port.id
    requested_asn        = 64550
    amazon_asn           = 64551
    amazon_account       = "684021030471"
    hosted_connection    = true
  }
}
```
