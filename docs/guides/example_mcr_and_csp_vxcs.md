---
page_title: "MCR and CSP VXCs"
subcategory: "Examples"
---

# MCR and CSP VXC's
This will provision a MCR (Megaport Cloud Router) connected to AWS, Azure and GCP using Megaport VXC's (Virtual Cross Connects).  

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

data megaport_location glb_switch_sydney {
  name = "Global Switch Sydney West"
}

data megaport_partner_port aws_test_sydney_1 {
  connect_type = "AWS"
  company_name = "AWS"
  product_name = "Asia Pacific (Sydney) (ap-southeast-2)"
  location_id  = data.megaport_location.glb_switch_sydney.id
}

data megaport_location ndc_b1 {
  name    = "NextDC B1"
  has_mcr = true
}

resource megaport_mcr test {
  mcr_name    = "Terraform Test - MCR"
  location_id = data.megaport_location.ndc_b1.id

  router {
    port_speed    = 5000
    requested_asn = 64555
  }
}

resource megaport_aws_connection test {
  vxc_name   = "Terraform Test - AWS VIF"
  rate_limit = 1000

  a_end {
    requested_vlan = 2191
  }

  csp_settings {
    attached_to          = megaport_mcr.test.id
    requested_product_id = data.megaport_partner_port.aws_test_sydney_1.id
    requested_asn        = 64550
    amazon_asn           = 64551
    amazon_account       = "123456789012"
  }
}

resource megaport_gcp_connection test {
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

resource megaport_azure_connection test {
  vxc_name    = "Terraform Test - ExpressRoute"
  rate_limit  = 1000

  a_end {
    requested_vlan = 3191
  }

  csp_settings {
    attached_to = megaport_mcr.test.id
    service_key = "12345678-b4d5-424b-976a-7b0de65a1b62"
    peerings {
	  private_peer = true
	  microsoft_peer = true
	}
  }
}
```

