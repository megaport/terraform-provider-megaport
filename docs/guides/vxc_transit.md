---
page_title: "Megaport Internet Virtual Cross Connect (VXC)"
description: |-
  How to create a Megaport Internet Virtual Cross Connect (VXC)
---

# Megaport Internet Virtual Cross Connect (VXC)

This guide provides an example configuration for deploying a Megaport Internet Virtual Cross Connect (VXC).

## Example Configuration

This example configuration creates a Megaport Virtual Edge (MVE) and an Internet VXC.

```terraform
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

data "megaport_mve_images" "aruba" {
  vendor_filter = "Aruba"
  id_filter     = 23
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
    image_id     = data.megaport_mve_images.aruba.mve_images.0.id
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
```

## Megaport Internet Documentation

For additional documentation on Megaport Internet, please visit [this page](https://docs.megaport.com/megaport-internet/).
