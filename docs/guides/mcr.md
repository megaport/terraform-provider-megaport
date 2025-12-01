---
page_title: "Megaport Cloud Router (MCR)"
description: |-
  Megaport Cloud Router (MCR) Deployment
---

# Megaport Cloud Router (MCR)

This guide provides an example configuration for deploying a Megaport Cloud Router (MCR).

## Example Configuration

This example configuration creates a Megaport Cloud Router (MCR) with a prefix filter list.

```terraform
provider "megaport" {
  environment           = "staging"
  access_key            = "access_key"
  secret_key            = "secret_Key"
  accept_purchase_terms = true
}

data "megaport_location" "bne_nxt1" {
  id = 5 # NextDC Brisbane B1
}

resource "megaport_mcr" "mcr" {
  product_name         = "Megaport MCR Example"
  port_speed           = 1000
  location_id          = data.megaport_location.bne_nxt1.id
  contract_term_months = 1
}
```

## Megaport Cloud Router Documentation

For more information on creating and using a Megaport Cloud Router, additional documentation is available [here](https://docs.megaport.com/mcr/). 