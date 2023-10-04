---
page_title: "Single Port"
subcategory: "Examples"
---

# Single Port
This will provision a single Megaport. It is useful for confirming the Megaport Terraform provider is installed and configured correctly.  

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

data "megaport_location" "bne_nxt1" {
  name    = "NextDC B1"
  has_mcr = false
}

resource "megaport_port" "port" {
  port_name   = "Terraform Example - Port"
  port_speed  = 1000
  location_id = data.megaport_location.bne_nxt1.id
  term        = 1
}
```
