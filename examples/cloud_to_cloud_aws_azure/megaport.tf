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

// megaport locations used for ports and mcr
data "megaport_location" "bne_nxt1" {
  name    = "NextDC B1"
  has_mcr = true
}

data "megaport_location" "syd_gs" {
  name = "Global Switch Sydney West"
}

// aws partner port
data "megaport_partner_port" "aws_port" {
  connect_type = "AWS"
  company_name = "AWS"
  product_name = "Asia Pacific (Sydney) (ap-southeast-2)"
  location_id  = data.megaport_location.syd_gs.id
}

// mcr
resource "megaport_mcr" "mcr" {
  mcr_name    = "${var.prefix} Terraform Example - MCR"
  location_id = data.megaport_location.bne_nxt1.id

  router {
    port_speed    = 5000
    requested_asn = 64555
  }
}

resource "megaport_azure_connection" "azure_vxc" {
  vxc_name   = "${var.prefix} Terraform Example - Azure VXC"
  rate_limit = var.azure_expressroute_bandwidth
  
  a_end {
    port_id        = megaport_mcr.mcr.id
    requested_vlan = 176
  }

  csp_settings {
    service_key = azurerm_express_route_circuit.express_route_circuit.service_key
    
    peerings {
      private_peer   = true
      microsoft_peer = true
    }
  }
}

// mcr to aws vxc
resource "megaport_aws_connection" "aws_vxc" {
  vxc_name   = "${var.prefix} Terraform Example - AWS VXC"
  rate_limit = 1000

  a_end {
    port_id        = megaport_mcr.mcr.id
    requested_vlan = 191
  }

  csp_settings {
    requested_product_id = data.megaport_partner_port.aws_port.id
    requested_asn        = 64550
    amazon_asn           = aws_dx_gateway.dx_gateway.amazon_side_asn
    amazon_account       = data.aws_caller_identity.current.account_id
  }
}
