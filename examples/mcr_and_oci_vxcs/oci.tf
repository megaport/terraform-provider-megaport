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

provider "oci" {}

resource "oci_core_virtual_circuit" "generated_oci_core_virtual_circuit" {
  bandwidth_shape_name = "1 Gbps"
  compartment_id       = "ocid1.compartment.oc1..aaaaaaaatzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz"
  cross_connect_mappings {
    customer_bgp_peering_ip = var.customer_bgp_peering_ip
    oracle_bgp_peering_ip   = var.oracle_bgp_peering_ip
  }
  customer_asn        = "133937"
  display_name        = "Primary FastConnect"
  gateway_id          = "ocid1.drg.oc1.ap-sydney-1.aaaaaaaa3yyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy"
  ip_mtu              = "MTU_1500"
  is_bfd_enabled      = "false"
  provider_service_id = "ocid1.providerservice.oc1.ap-sydney-1.aaaaaaaaewwwwwwwwwwwwwwwwwwwwwwwwwwww"
  type                = "PRIVATE"
}

output "fast_connect_ocid" {
  value = oci_core_virtual_circuit.generated_oci_core_virtual_circuit.id
}
