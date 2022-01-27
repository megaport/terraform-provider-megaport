---
page_title: "Lag Port"
subcategory: "Examples"
---

# Lag Port
This will provision a Megaport with 3 Lagged ports.

Replace the `username`, `password` and optional `mfa_otp_key` with your own credentials.

```
terraform {
  required_providers {
    megaport = {
      source  = "megaport/megaport"
      version = ">=0.1.4"
    }
  }
}

provider "megaport" {
  username              = "my.test.user@example.org"
  password              = "n0t@re4lPassw0rd"
  mfa_otp_key           = "ABCDEFGHIJK01234"
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
