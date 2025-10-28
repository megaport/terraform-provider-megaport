---
page_title: "Megaport Virtual Cross Connect (VXC)"
description: |-
  How to create a Megaport Port to Port Virtual Cross Connect (VXC)
---

# Megaport Virtual Cross Connect (VXC)

This guide provides an example configuration for deploying a Megaport Port to Port Virtual Cross Connect (VXC).

## Example Configuration

This example configuration creates two Megaport Ports and a Virtual Cross Connect (VXC).

```terraform
provider "megaport" {
  environment           = "staging"
  access_key            = "access_key"
  secret_key            = "secret_Key"
  accept_purchase_terms = true
}

data "megaport_location" "loc" {
  id = 5 # NextDC Brisbane B1
}

resource "megaport_port" "port_1" {
  product_name           = "Megaport Port A-End"
  port_speed             = 1000
  location_id            = data.megaport_location.loc.id
  contract_term_months   = 1
  marketplace_visibility = false
  cost_centre            = "Megaport Port A-End"
}

resource "megaport_port" "port_2" {
  product_name           = "Megaport Port B-End"
  port_speed             = 1000
  location_id            = data.megaport_location.loc.id
  contract_term_months   = 1
  marketplace_visibility = false
  cost_centre            = "Megaport Port B-End"
}

resource "megaport_vxc" "vxc" {
  product_name         = "Megaport VXC"
  rate_limit           = 1000
  contract_term_months = 1

  a_end = {
    requested_product_uid = megaport_port.port_1.product_uid
  }

  b_end = {
    requested_product_uid = megaport_port.port_2.product_uid
  }
}
```

## VXC Documentation

For additional documentation on VXCs, please visit [this page](https://docs.megaport.com/connections/).