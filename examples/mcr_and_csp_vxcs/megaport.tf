/**
 * Copyright 2020 Megaport Pty Ltd
 *
 * Licensed under the Mozilla Public License, Version 2.0 (the
 * "License"); you may not use this file except in compliance with
 * the License. You may obtain a copy of the License at
 *
 *       https://mozilla.org/MPL/2.0/
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */



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

