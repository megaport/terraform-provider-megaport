provider "oci" {}

resource "oci_core_virtual_circuit" "generated_oci_core_virtual_circuit" {
	bandwidth_shape_name = "1 Gbps"
	compartment_id = "ocid1.compartment.oc1..aaaaaaaatzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz"
	cross_connect_mappings {
		customer_bgp_peering_ip = var.customer_bgp_peering_ip 
		oracle_bgp_peering_ip = var.oracle_bgp_peering_ip
	}
	customer_asn = "133937"
	display_name = "Primary FastConnect"
	gateway_id = "ocid1.drg.oc1.ap-sydney-1.aaaaaaaa3yyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy"
	ip_mtu = "MTU_1500"
	is_bfd_enabled = "false"
	provider_service_id = "ocid1.providerservice.oc1.ap-sydney-1.aaaaaaaaewwwwwwwwwwwwwwwwwwwwwwwwwwww"
	type = "PRIVATE"
}

output fast_connect_ocid {
    value = oci_core_virtual_circuit.generated_oci_core_virtual_circuit.id
}