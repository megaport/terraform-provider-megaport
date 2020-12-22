---
page_title: "Full Ecosystem"
subcategory: "Examples"
---

# Full Ecosystem

Provision one of everything!

Replace the `username`, `password` and optional `mfa_otp_key` with your own credentials.

This configuration will deploy on the staging environment. To use this on production, valid CSP attributes are required:
+ `megaport_aws_connection.amazon_account`
+ `megaport_gcp_connection.pairing_key`
+ `megaport_azure_connection.service_key`

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
  has_mcr = true
}

data megaport_location glb_switch_sydney {
  name = "Global Switch Sydney West"
}

data megaport_partner_port aws_test_sydney_1 {
  connect_type = "AWS"
  company_name = "AWS"
  product_name = "Asia Pacific (Sydney) (ap-southeast-2)"
  location_id  = data.megaport_location.glb_switch_sydney.id
}

data megaport_mcr tf_test {
  mcr_id = megaport_mcr.test.id
}

data megaport_port tf_test {
  port_id = megaport_port.tf_test.id
}

resource megaport_mcr test {
  mcr_name    = "Terraform Test - MCR"
  location_id = data.megaport_location.ndc_b1.id

  router {
    port_speed    = 5000
    requested_asn = 64555
  }
}

resource megaport_port tf_test {
  port_name   = "Test Port"
  port_speed  = 1000
  location_id = data.megaport_location.ndc_b1.id
  term        = 12
}

resource megaport_aws_connection test {
  vxc_name   = "Terraform Test - AWS VIF"
  rate_limit = 1000

  a_end {
    requested_vlan = 191
  }

  csp_settings {
    attached_to          = megaport_mcr.test.id
    requested_product_id = data.megaport_partner_port.aws_test_sydney_1.id
    requested_asn        = 64550
    amazon_asn           = 64551
    amazon_account       = "123456789012"
  }
}

resource "megaport_gcp_connection" "test" {
  vxc_name   = "Terraform Test - GCP"
  rate_limit = 1000

  a_end {
    requested_vlan = 182
  }

  csp_settings {
    attached_to = megaport_mcr.test.id
    pairing_key = "7e51371e-72a3-40b5-b844-2e3efefaee59/australia-southeast1/2"
  }
}

data megaport_location nextdc_brisbane_2 {
  name = "NextDC B2"
}

resource megaport_port port_2 {
  port_name      = "Port 2"
  port_speed     = 10000
  location_id    = data.megaport_location.nextdc_brisbane_2.id
  lag            = true
  lag_port_count = 5

}

resource megaport_vxc vxc {
  vxc_name   = "VXC Port->Port"
  rate_limit = 1000

  a_end {
    port_id = megaport_port.tf_test.id
    requested_vlan = 180
  }

  b_end {
    port_id = megaport_port.port_2.id
    requested_vlan = 180
  }
}

resource megaport_vxc mcr_vxc {
  vxc_name   = "Terraform Test VXC Port->MCR"
  rate_limit = 1000

  a_end {
    port_id = megaport_port.tf_test.id
    requested_vlan = 181
  }

  b_end {
    port_id = megaport_mcr.test.id
    requested_vlan = 181
  }
}
```
