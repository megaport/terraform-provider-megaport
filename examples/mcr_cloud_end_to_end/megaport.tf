data "megaport_location" "megaport_location_1" {
  name = var.megaport_location_1
}

data "megaport_location" "megaport_location_2" {
  name = var.megaport_location_2
}

resource "megaport_mcr" "mcr_1" {
  product_name         = var.megaport_mcr_1_name
  port_speed           = var.megaport_mcr_1_speed
  location_id          = data.megaport_location.megaport_location_1.id
  diversity_zone       = var.megaport_mcr_1_diversity_zone
  contract_term_months = var.megaport_mcr_1_term
}

data "megaport_partner" "aws_port_location_1" {
  connect_type   = "AWSHC"
  company_name   = "AWS"
  product_name   = var.megaport_aws_port_location_1_name
  location_id    = data.megaport_location.megaport_location_2.id
  diversity_zone = var.megaport_aws_port_location_1_diversity_zone
}

resource "megaport_vxc" "aws_vxc_1" {
  product_name         = var.megaport_aws_vxc_1_name
  rate_limit           = var.megaport_aws_vxc_1_bandwidth
  contract_term_months = var.megaport_aws_vxc_1_term

  a_end = {
    requested_product_uid = megaport_mcr.mcr_1.product_uid
  }

  a_end_partner_config = {
    partner = "vrouter"
    vrouter_config = {
      interfaces = [
        {
          ip_addresses = ["192.168.60.1/30"]
          bgp_connections = [
            {
              peer_asn         = "64512"
              local_ip_address = "192.168.60.1"
              peer_ip_address  = "192.168.60.2"
              password         = "<password>"
            }
          ]
        }
      ]
    }
  }

  b_end = {
    requested_product_uid = data.megaport_partner.aws_port_location_1.product_uid
  }

  b_end_partner_config = {
    partner = "aws"
    aws_config = {
      name          = var.megaport_aws_vxc_1_name
      connect_type  = "AWSHC"
      owner_account = var.aws_account_id
    }
  }
}

resource "megaport_vxc" "expressroute_vxc_1" {
  product_name         = var.megaport_expressroute_vxc_1_name
  rate_limit           = var.azure_expressroute_bandwidth_1
  contract_term_months = var.megaport_expressroute_vxc_1_term

  a_end = {
    requested_product_uid = megaport_mcr.mcr_1.product_uid
  }

  b_end = {}

  b_end_partner_config = {
    partner = "azure"
    azure_config = {
      port_choice = "primary"
      service_key = azurerm_express_route_circuit.express_route_circuit_1.service_key
        peers = [{
        type             = "private"
        vlan             = var.azure_express_route_circuit_vlan_1
        peer_asn         = var.megaport_mcr_1_asn
        primary_subnet   = var.azure_express_route_circuit_primary_subnet_1
        secondary_subnet = var.azure_express_route_circuit_secondary_subnet_1
      }]
    }
  }
}

data "megaport_partner" "google_port_location_2" {
  connect_type = "GOOGLE"
  company_name = "Google inc.."
  product_name = var.megaport_google_port_location_2_name
  location_id  = data.megaport_location.megaport_location_2.id
}

resource "megaport_vxc" "google_vxc_1" {
  product_name         = var.megaport_google_vxc_1_name
  rate_limit           = var.megaport_google_vxc_1_bandwidth
  contract_term_months = var.megaport_google_vxc_1_term

  a_end = {
    requested_product_uid = megaport_mcr.mcr_1.product_uid
  }

  b_end = {
    requested_product_uid = data.megaport_partner.google_port_location_2.product_uid
  }

  b_end_partner_config = {
    partner = "google"
    google_config = {
      pairing_key = google_compute_interconnect_attachment.vlan_attach_1.pairing_key
    }
  }
}
