---
page_title: "Megaport Single Port"
description: |-
  Megaport Single Port Deployment
---

# Megaport Single Port

This guide provides an example configuration for deploying a Megaport Single Port.

## Example Configuration

This example configuration creates a Megaport Single Port.

```terraform
terraform {
  required_providers {
    megaport = {
      source  = "megaport/megaport"
      version = "1.0.0-beta1"
    }
  }
}
provider "megaport" {
  environment           = "staging"
  access_key            = "access_key"
  secret_key            = "secret_Key"
  accept_purchase_terms = true
}

data "megaport_location" "bne_nxt1" {
  name = "NextDC B1"
}

resource "megaport_port" "port" {
  product_name           = "Megaport Port Example"
  port_speed             = 1000
  location_id            = data.megaport_location.bne_nxt1.id
  contract_term_months   = 1
  marketplace_visibility = false
  cost_centre            = "Megaport Single Port Example"
}
```

## Additional documentation on Ports

For additional documentation on creating a port, please visit [this page](https://docs.megaport.com/connections/creating-port/).