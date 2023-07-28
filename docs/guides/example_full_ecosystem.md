---
page_title: "Full Ecosystem"
subcategory: "Examples"
---

# Full Ecosystem

Provision one of everything!

Replace the `username`, `password` and optional `mfa_otp_key` with your own credentials.

This configuration will deploy on the staging environment. To use this on production, valid CSP attributes are required:
+ `megaport_aws_connection.amazon_account`
+ `megaport_gcp_connection.pairing_key`
+ `megaport_azure_connection.service_key`

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

data "megaport_location" "bne_nxt2" {
  name = "NextDC B2"
}

data "megaport_location" "syd_gs" {
  name = "Global Switch Sydney West"
}

data "megaport_partner_port" "aws_port" {
  connect_type = "AWS"
  company_name = "AWS"
  product_name = "Asia Pacific (Sydney) (ap-southeast-2)"
  location_id  = data.megaport_location.syd_gs.id
}

resource "megaport_port" "port" {
  port_name   = "Terraform Example - Port"
  port_speed  = 1000
  location_id = data.megaport_location.bne_nxt1.id
  term        = 12
}

resource "megaport_port" "lag_port" {
  port_name      = "Terraform Example - LAG Port"
  port_speed     = 10000
  location_id    = data.megaport_location.bne_nxt2.id
  lag            = true
  lag_port_count = 3
}

resource "megaport_mcr" "mcr" {
  mcr_name    = "Terraform Example - MCR"
  location_id = data.megaport_location.bne_nxt1.id
  term        = 12

  router {
    port_speed    = 2500
    requested_asn = 64555
  }
}

resource "megaport_vxc" "port_vxc" {
  vxc_name   = "Terraform Example - Port-to-Port VXC"
  rate_limit = 1000

  a_end {
    port_id        = megaport_port.port.id
    requested_vlan = 180
  }

  b_end {
    port_id        = megaport_port.lag_port.id
    requested_vlan = 180
  }
}

resource "megaport_vxc" "mcr_vxc" {
  vxc_name   = "Terraform Example - Port-to-MCR VXC"
  rate_limit = 1000

  a_end {
    port_id        = megaport_port.port.id
    requested_vlan = 181
  }

  b_end {
    port_id        = megaport_mcr.mcr.id
    requested_vlan = 181
  }
}

resource "megaport_aws_connection" "aws_vxc" {
  vxc_name   = "Terraform Example - AWS VXC"
  rate_limit = 1000

  a_end {
    port_id        = megaport_mcr.mcr.id
    requested_vlan = 191
  }

  csp_settings {
    requested_product_id = data.megaport_partner_port.aws_port.id
    requested_asn        = 64550
    amazon_asn           = 64551
    amazon_account       = "123456789012"
  }
}

resource "megaport_gcp_connection" "gcp_vxc" {
  vxc_name   = "Terraform Example - GCP VXC"
  rate_limit = 1000

  a_end {
    port_id        = megaport_mcr.mcr.id
    requested_vlan = 182
  }

  csp_settings {
    pairing_key          = "7e51371e-72a3-40b5-b844-2e3efefaee59/australia-southeast1/2"
    requested_product_id = "90558833-e14f-49cf-84ba-bce1c2c40f2d"
  }
}

resource "megaport_azure_connection" "azure_vxc" {
  vxc_name   = "Terraform Example - Azure VXC"
  rate_limit = 200

  a_end {
    port_id        = megaport_mcr.mcr.id
    requested_vlan = 0
  }

  csp_settings {
    service_key                   = "1b2329a5-56dc-45d0-8a0d-87b706297777"
    auto_create_private_peering   = true
    auto_create_microsoft_peering = true
  }
}
```
