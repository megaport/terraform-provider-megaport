resource "megaport_nat_gateway_packet_filter" "example" {
  nat_gateway_product_uid = megaport_nat_gateway.example.product_uid
  description             = "Ingress filter"

  entries = [
    {
      action              = "permit"
      description         = "Allow HTTPS"
      source_address      = "0.0.0.0/0"
      destination_address = "10.0.0.0/24"
      destination_ports   = "443"
      ip_protocol         = 6 # TCP
    },
    {
      action              = "permit"
      description         = "Allow DNS"
      source_address      = "0.0.0.0/0"
      destination_address = "10.0.0.0/24"
      destination_ports   = "53"
      ip_protocol         = 17 # UDP
    },
    {
      action              = "deny"
      description         = "Default deny"
      source_address      = "0.0.0.0/0"
      destination_address = "0.0.0.0/0"
    },
  ]
}
