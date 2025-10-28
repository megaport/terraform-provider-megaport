---
page_title: "LAG Port"
description: |-
  Lag Port Deployment
---

# LAG Port

This guide provides an example configuration for deploying a LAG Port.

## Example Configuration

This example configuration creates a LAG Port.

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

resource "megaport_lag_port" "lag_port" {
  product_name           = "Megaport Lag Port Example"
  port_speed             = 10000
  location_id            = data.megaport_location.bne_nxt1.id
  contract_term_months   = 1
  marketplace_visibility = false
  lag_count              = 1
  cost_centre            = "Lag Port Example"
}
```

## LAG Port Documentation

For more information on creating and using LAG ports with Megaport, please see [Creating a Link Aggregation Group (LAG) Port](https://docs.megaport.com/connections/lag/) and [Adding a Port to a LAG](https://docs.megaport.com/connections/lag-adding/).