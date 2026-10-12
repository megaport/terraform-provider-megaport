# Read the IPsec tunnels configured on an MCR.
data "megaport_mcr_ipsec" "example" {
  mcr_id = megaport_mcr.example.product_uid
}

output "ipsec_tunnel_count" {
  value = data.megaport_mcr_ipsec.example.total_tunnel_count
}

# Map each VXC UID to the peer addresses of its tunnels.
output "ipsec_tunnel_destinations" {
  value = {
    for vxc in data.megaport_mcr_ipsec.example.vxcs :
    vxc.product_uid => [for tunnel in vxc.tunnels : tunnel.destination_ip_address]
  }
}
