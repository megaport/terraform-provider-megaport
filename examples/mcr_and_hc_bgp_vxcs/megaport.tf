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


data megaport_location eq_sy3 {
  name = "Equinix SY3"
}

data megaport_location ndc_b1 {
  name    = "NextDC B1"
  has_mcr = true
}

data megaport_partner_port test {
  connect_type = "AWSHC"
  company_name = "AWS"
  product_name = "Asia Pacific (Sydney) (ap-southeast-2) [DZ-BLUE]"
  location_id  = data.megaport_location.eq_sy3.id
}


resource megaport_mcr test {
  mcr_name    = "Terraform Test - MCR"
  location_id = data.megaport_location.ndc_b1.id

  router {
    port_speed    = 2500
    requested_asn = 64555
  }

}

resource megaport_aws_connection test {

  vxc_name   = "Terraform Test - AWS VXC Hosted Connection"
  rate_limit = 1000

  a_end {

    requested_vlan = 0
    
    partner_configuration {

      ip_addresses = [ "11.192.0.25/29" ,"12.192.0.25/29" ]

      bfd_configuration {
        tx_internal = 500
        rx_internal = 400
        multiplier = 5
      }

      bgp_connection {
        peer_asn = 62512
        local_ip_address = "12.192.0.25"
        peer_ip_address = "12.192.0.26"
        password = "notARealPassword"
        shutdown = true
        med_in = 100
        med_out = 100
        bfd_enabled = true
      }

    }

  }

  csp_settings {
    attached_to          = megaport_mcr.test.id
    requested_product_id = data.megaport_partner_port.test.id
    requested_asn        = 64550
    amazon_asn           = 64551
    amazon_account       = "684021030471"
    hosted_connection    = true
  }

}