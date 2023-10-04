---
page_title: "Lag Port"
subcategory: "Examples"
---

# Lag Port
This will provision a Megaport with 3 Lagged ports.

Replace the `access_key` and `secret_key` with your own credentials.

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

data "megaport_location" "bne_nxt2" {
  name = "NextDC B2"
}

resource "megaport_port" "lag_port" {
  port_name      = "Terraform Example - LAG Port"
  port_speed     = 10000
  location_id    = data.megaport_location.bne_nxt2.id
  lag            = true
  lag_port_count = 3
}
```
