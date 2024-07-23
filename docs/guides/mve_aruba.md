---
page_title: "Aruba Megaport Virtual Edge (MVE)"
description: |-
  Aruba Megaport Virtual Edge (MVE) Deployment
---

# Aruba Megaport Virtual Edge (MVE)

This guide provides an example configuration for deploying an Aruba Megaport Virtual Edge (MVE).

## Example Configuration

This example configuration creates an Aruba MVE.

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

resource "megaport_mve" "mve" {
  product_name         = "Megaport MVE Example"
  location_id          = data.megaport_location.bne_nxt1.id
  contract_term_months = 1

  vnics = [
    {
      description = "Data Plane"
    },
    {
      description = "Control Plane"
    },
    {
      description = "Management Plane"
    }
  ]

  vendor_config = {
    vendor       = "aruba"
    product_size = "MEDIUM"
    image_id     = 23
    account_name = "Aruba Test Account"
    account_key  = "12345678"
    system_tag   = "Preconfiguration-aruba-test-1"
  }
}
```

## MVE Documentation

For more information on creating and using an Aruba Megaport Virtual Edge, additional documentation is available [here](https://docs.megaport.com/mve/).