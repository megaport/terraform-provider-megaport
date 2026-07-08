resource "megaport_mcr_ipsec_addon" "example" {
  mcr_id       = megaport_mcr.example.product_uid
  tunnel_count = 10
}
