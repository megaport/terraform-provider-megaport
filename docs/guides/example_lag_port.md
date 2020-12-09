---
page_title: "Lag Port"
subcategory: "Examples"
---

# Lag Port
This will provision a Megaport with 5 Lagged ports

```
terraform {
  required_providers {
    megaport = {
      source = "megaport/megaport"
      version = "0.1.1"
    }
  }
}

provider "megaport" {
    username                = "my.test.user@example.org"
    password                = "n0t@re4lPassw0rd"
    mfa_otp_key             = "ABCDEFGHIJK01234"
    accept_purchase_terms   = true
    delete_ports            = true
    environment             = "staging"
}

data megaport_location nextdc_brisbane_2 {
  name = "NextDC B2"
}

resource megaport_port lag_port {
  port_name      = "Test Lag Port"
  port_speed     = 10000
  location_id    = data.megaport_location.nextdc_brisbane_2.id
  lag            = true
  lag_port_count = 5
}
```
