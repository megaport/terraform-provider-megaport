
provider "megaport" {}

data megaport_location ndc_b1 {
  name    = "Equinix SY1"
  has_mcr = true
}

data "megaport_partner_port" "primary_oci_port" {
  product_name   = "OCI (ap-sydney-1) (BMC)"
  diversity_zone = "blue"
  location_id    = data.megaport_location.ndc_b1.id
}

resource "megaport_mcr" "mcr" {
  mcr_name    = "Terraform Example - MCR"
  location_id = data.megaport_location.ndc_b1.id

  router {
    port_speed    = 2500
  }
}

resource "megaport_oci_connection" "oci_vxc" {
  vxc_name   = "Terraform Example - OCI VXC"
  rate_limit = 1000

  a_end {
    port_id        = megaport_mcr.mcr.id
  }

    a_end_mcr_configuration {
    ip_addresses     = [var.customer_bgp_peering_ip]

    bgp_connection {
      peer_asn           = 31898
      local_ip_address   = var.customer_bgp_peering_ip
      peer_ip_address    = var.oracle_bgp_peering_ip
    }
  }


  csp_settings {

   virtual_circut_id = oci_core_virtual_circuit.generated_oci_core_virtual_circuit.id
   requested_product_id = data.megaport_partner_port.primary_oci_port.id

  }
}