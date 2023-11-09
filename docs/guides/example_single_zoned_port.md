---
page_title: "Single Zoned Port"
subcategory: "Examples"
---

# Single Port
This will provision a single Megaport in the Blue Diversity Zone. 

Replace the `username`, `password` and optional `mfa_otp_key` with your own credentials.

```
terraform {
  required_providers {
    megaport = {
      source  = "megaport/megaport"
      version = ">=0.2.11"
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

data "megaport_location" "bne_nxt1" {
  name    = "NextDC B1"
  has_mcr = false
}

resource "megaport_port" "port" {
  port_name   = "Terraform Example - Port"
  port_speed  = 1000
  location_id = data.megaport_location.bne_nxt1.id
  term        = 1
  diversity_zone = "blue"
}
```
