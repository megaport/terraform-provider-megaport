---
page_title: "Megaport Virtual Cross Connect (VXC) with Megaport Virtual Edge (MVE) and AWS"
description: |-
  How to create a Megaport Virtual Cross Connect (VXC) with a Megaport Virtual Edge (MVE) and AWS
---

# Megaport Virtual Cross Connect (VXC) with Megaport Virtual Edge (MVE) and AWS

This guide provides an example configuration for deploying a Megaport Virtual Cross Connect (VXC) with a Megaport Virtual Edge (MVE) and AWS.

## Example Configuration

This example configuration creates a Megaport Virtual Edge (MVE) and a Virtual Cross Connect (VXC) to AWS.

```terraform
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
  connect_type = "AWSHC"
  company_name = "AWS"
  product_name = "Asia Pacific (Sydney) (ap-southeast-2)"
  location_id  = data.megaport_location.syd_gs.id
}

data "megaport_mve_images" "aruba" {
  vendor_filter = "Aruba"
  id_filter     = 23
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
      description = "to_aws"
    },
    {
      description = "to_port"
    },
  ]

  vendor_config = {
    vendor       = "aruba"
    product_size = "MEDIUM"
    image_id     = data.megaport_mve_images.aruba.mve_images.0.id
    account_name = "Megaport Aruba MVE"
    account_key  = "Megaport Aruba MVE"
    system_tag   = "Preconfiguration-aruba-test-1"
  }
}

resource "megaport_vxc" "aws_vxc" {
  product_name         = "Megaport MVE VXC AWS"
  rate_limit           = 100
  contract_term_months = 1

  a_end = {
    requested_product_uid = megaport_mve.mve.product_uid
    inner_vlan            = 100
    vnic_index            = 0
  }

  b_end = {
    requested_product_uid = data.megaport_partner.aws_port.product_uid
  }

  b_end_partner_config = {
    partner = "aws"
    aws_config = {
      name          = "Megaport MVE VXC AWS"
      asn           = 65121
      type          = "private"
      connect_type  = "AWSHC"
      amazon_asn    = 64512
      owner_account = "123456789012"
    }
  }
}
```