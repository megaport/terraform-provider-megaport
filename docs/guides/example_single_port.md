---
page_title: "Single Port"
subcategory: "Examples"
---

# Single Port
This will provision a single Megaport. It is useful for confirming the Megaport Terraform provider is installed and configured correctly.  

Replace the `username`, `password` and optional `mfa_otp_key` with your own credentials.

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

data megaport_location ndc_b1 {
  name    = "NextDC B1"
  has_mcr = false
}

resource megaport_port tf_test {
  port_name   = "Test Port"
  port_speed  = 1000
  location_id = data.megaport_location.ndc_b1.id
  term        = 1
}
```
