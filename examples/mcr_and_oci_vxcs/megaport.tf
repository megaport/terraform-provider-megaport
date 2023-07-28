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
      version = ">=0.2.10"
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

data "megaport_location" "syd_sy1" {
  name    = "Equinix SY1"
  has_mcr = true
}

data "megaport_partner_port" "primary_oci_port" {
  product_name   = "OCI (ap-sydney-1) (BMC)"
  diversity_zone = "blue"
  location_id    = data.megaport_location.syd_sy1.id
}

resource "megaport_mcr" "mcr" {
  mcr_name    = "Terraform Example - MCR"
  location_id = data.megaport_location.syd_sy1.id

  router {
    port_speed = 2500
  }
}

resource "megaport_oci_connection" "oci_vxc" {
  vxc_name   = "Terraform Example - OCI VXC"
  rate_limit = 1000

  a_end {
    port_id = megaport_mcr.mcr.id
  }

  a_end_mcr_configuration {
    ip_addresses = [var.customer_bgp_peering_ip]

    bgp_connection {
      peer_asn         = 31898
      local_ip_address = var.customer_bgp_peering_ip
      peer_ip_address  = var.oracle_bgp_peering_ip
    }
  }

  csp_settings {
    virtual_circut_id    = oci_core_virtual_circuit.generated_oci_core_virtual_circuit.id
    requested_product_id = data.megaport_partner_port.primary_oci_port.id
  }
}
