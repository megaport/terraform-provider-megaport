---
page_title: "Two Ports and VXC"
subcategory: "Examples"
---
# Two Ports and VXC
This will provision two Megaports in different locations linked by a VXC (Virtual Cross Connect).

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
  has_mcr = true
}

data "megaport_location" "bne_nxt2" {
  name = "NextDC B2"
}

resource "megaport_port" "port_1" {
  port_name   = "Terraform Example - Port 1"
  port_speed  = 1000
  location_id = data.megaport_location.bne_nxt1.id
}

resource "megaport_port" "port_2" {
  port_name   = "Terraform Example - Port 2"
  port_speed  = 1000
  location_id = data.megaport_location.bne_nxt2.id
}

resource "megaport_vxc" "vxc" {
  vxc_name   = "Terraform Example - VXC"
  rate_limit = 1000

  a_end {
    port_id        = megaport_port.port_1.id
    requested_vlan = 180
  }

  b_end {
    port_id        = megaport_port.port_2.id
    requested_vlan = 180
  }
}
```
