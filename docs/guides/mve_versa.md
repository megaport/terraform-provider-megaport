---
page_title: "Versa Megaport Virtual Edge (MVE)"
description: |-
  Versa Megaport Virtual Edge (MVE) Deployment
---

# Versa Megaport Virtual Edge (MVE)

This guide provides an example configuration for deploying a Versa Megaport Virtual Edge (MVE).

## Example Configuration

This example configuration creates a Versa MVE.

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

data "megaport_mve_images" "versa" {
  vendor_filter = "Versa"
  id_filter     = 20
}

resource "megaport_mve" "mve" {
  product_name         = "Megaport Versa MVE Example"
  location_id          = data.megaport_location.bne_nxt1.id
  contract_term_months = 1

  vendor_config = {
    vendor             = "versa"
    product_size       = "LARGE"
    image_id           = data.megaport_mve_images.versa.mve_images.0.id
    director_address   = "director1.versa.com"
    controller_address = "controller1.versa.com"
    local_auth         = "SDWAN-Branch@Versa.com"
    remote_auth        = "Controller-1-staging@Versa.com"
    serial_number      = "Megaport-Hub1"
  }
}
```

## MVE Documentation

For more information on creating and using an Aruba Megaport Virtual Edge, additional documentation is available [here](https://docs.megaport.com/mve/).