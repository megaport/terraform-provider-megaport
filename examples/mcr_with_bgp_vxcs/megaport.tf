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

data "megaport_location" "bne_nxt1" {
  name    = "NextDC B1"
  has_mcr = true
}

data "megaport_location" "syd_sy3" {
  name = "Equinix SY3"
}

data "megaport_partner_port" "aws_port" {
  connect_type = "AWSHC"
  company_name = "AWS"
  product_name = "Asia Pacific (Sydney) (ap-southeast-2) [DZ-BLUE]"
  location_id  = data.megaport_location.syd_sy3.id
}

resource "megaport_mcr" "mcr" {
  mcr_name    = "Terraform Example - MCR"
  location_id = data.megaport_location.bne_nxt1.id

  router {
    port_speed    = 2500
    requested_asn = 64555
  }

  prefix_filter_list {
    name           = "Prefix filter list 1"
    address_family = "IPv4"

    entry {
      action    = "permit"
      prefix    = "10.0.1.0/24"
      range_min = 24
      range_max = 24
    }
    entry {
      action    = "deny"
      prefix    = "10.0.2.0/24"
      range_min = 24
      range_max = 24
    }
  }
}

resource "megaport_azure_connection" "azure_vxc" {
  vxc_name   = "Terraform Example - Azure VXC"
  rate_limit = 200

  a_end {
    port_id        = megaport_mcr.mcr.id
    requested_vlan = 0
  }

  csp_settings {
    service_key = "197d927b-90bc-4b1b-bffd-fca17a7ec735"

    private_peering {
      peer_asn         = "64555"
      primary_subnet   = "10.0.1.0/30"
      secondary_subnet = "10.0.2.0/30"
      shared_key       = "SharedKey1"
      requested_vlan   = 100
    }
  }
}

resource "megaport_aws_connection" "aws_vxc" {
  vxc_name   = "Terraform Example - AWSHC VXC"
  rate_limit = 1000

  a_end {
    port_id        = megaport_mcr.mcr.id
    requested_vlan = 0
  }

  a_end_mcr_configuration {
    ip_addresses     = ["10.0.0.1/30"]
    nat_ip_addresses = ["10.0.0.1"]

    bfd_configuration {
      tx_interval = 500
      rx_interval = 400
      multiplier  = 5
    }

    bgp_connection {
      peer_asn           = 64512
      local_ip_address   = "10.0.0.1"
      peer_ip_address    = "10.0.0.2"
      password           = "notARealPassword"
      shutdown           = false
      med_in             = 100
      med_out            = 100
      bfd_enabled        = true
      export_policy      = "deny"
      permit_export_to   = ["10.0.1.2"]
      import_permit_list = "Prefix filter list 1"
    }
  }

  csp_settings {
    requested_product_id = data.megaport_partner_port.aws_port.id
    requested_asn        = 64550
    amazon_asn           = 64551
    amazon_account       = "684021030471"
    hosted_connection    = true
  }
}
