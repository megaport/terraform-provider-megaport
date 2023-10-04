---
page_title: "MCR with BGP VXCs"
subcategory: "Examples"
---

# MCR with BGP VXC's
This will provision an MCR (Megaport Cloud Router) connected to AWS over a Hosted Connection with BGP and BFD configuration.

Replace the `access_key` and `secret_key` with your own credentials.

This configuration will deploy on the staging environment. To use this on production, valid CSP attributes are required:
+ `megaport_aws_connection.amazon_account`

```
terraform {
  required_providers {
    megaport = {
      source  = "megaport/megaport"
      version = ">=0.3.0"
    }
  }
}

provider "megaport" {
  access_key            = "my-access-key"
  secret_key            = "my-secret-key"
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
  connect_type   = "AWSHC"
  company_name   = "AWS"
  product_name   = "Asia Pacific (Sydney) (ap-southeast-2) [DZ-BLUE]"
  diversity_zone = "blue"
  location_id    = data.megaport_location.syd_sy3.id
}

resource "megaport_mcr" "mcr" {
  mcr_name    = "Terraform Example - MCR"
  location_id = data.megaport_location.bne_nxt1.id

  router {
    port_speed    = 2500
    requested_asn = 64555
  }

  prefix_filter_list {
    name           = "Prefix filter list 1"
    address_family = "IPv4"

    entry {
      action    = "permit"
      prefix    = "10.0.1.0/24"
      range_min = 24
      range_max = 24
    }
    entry {
      action    = "deny"
      prefix    = "10.0.2.0/24"
      range_min = 24
      range_max = 24
    }
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
    service_key = "197d927b-90bc-4b1b-bffd-fca17a7ec735"

    private_peering {
      peer_asn         = "64555"
      primary_subnet   = "10.0.1.0/30"
      secondary_subnet = "10.0.2.0/30"
      shared_key       = "SharedKey1"
      requested_vlan   = 100
    }
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
    ip_addresses     = ["10.0.0.1/30"]
    nat_ip_addresses = ["10.0.0.1"]

    bfd_configuration {
      tx_interval = 500
      rx_interval = 400
      multiplier  = 5
    }

    bgp_connection {
      peer_asn           = 64512
      local_ip_address   = "10.0.0.1"
      peer_ip_address    = "10.0.0.2"
      password           = "notARealPassword"
      shutdown           = false
      med_in             = 100
      med_out            = 100
      bfd_enabled        = true
      export_policy      = "deny"
      permit_export_to   = ["10.0.1.2"]
      import_permit_list = "Prefix filter list 1"
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
