---
page_title: "Changing VLAN for a Virtual Cross Connect End Configuration"
description: |-
    How to change the VLAN for a Virtual Cross Connect (VXC) end configuration
---

# Changing VLAN for a Virtual Cross Connect End Configuration

This guide provides an example configuration for changing the VLAN for a Virtual Cross Connect (VXC) end configuration.

## Example Configuration

This serves as an example of how to change the VLANs on Virtual Cross Connect (VXC) end configurations.  In the first example, we will provide an `ordered_vlan` of 100 and 101 in the respective `a_end` and `b_end` configurations.

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
    ordered_vlan          = 100
  }

  b_end = {
    requested_product_uid = megaport_port.port_2.product_uid
    ordered_vlan          = 101
  }
}
```

To change the VLAN for each end configuration, simply specify a different value for the `ordered_vlan` attribute.  In the following example, we will change the `ordered_vlan` to 200 and 201 in the respective `a_end` and `b_end` configurations.

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
    ordered_vlan          = 200
  }

  b_end = {
    requested_product_uid = megaport_port.port_2.product_uid
    ordered_vlan          = 201
  }
}
```