# Query the NAT Gateway speed / session-count availability matrix.
data "megaport_nat_gateway_sessions" "available" {}

# Output the full matrix for reference.
# Run `terraform plan` to see values without applying.
output "nat_gateway_sessions" {
  description = "Map of NAT Gateway speed (Mbps) to permitted session counts"
  value = {
    for entry in data.megaport_nat_gateway_sessions.available.sessions :
    tostring(entry.speed_mbps) => entry.session_count
  }
}

# Example: pick the highest session count permitted at 1000 Mbps.
locals {
  sessions_at_1000_mbps = one([
    for entry in data.megaport_nat_gateway_sessions.available.sessions :
    entry.session_count if entry.speed_mbps == 1000
  ])
  max_sessions_at_1000_mbps = max(local.sessions_at_1000_mbps...)
}

# Example usage with a megaport_nat_gateway resource — validates the chosen
# speed / session_count combination at plan time rather than apply time.
# resource "megaport_nat_gateway" "example" {
#   product_name         = "My NAT Gateway"
#   location_id          = 1234
#   speed                = 1000
#   session_count        = local.max_sessions_at_1000_mbps
#   contract_term_months = 1
#   diversity_zone       = "red"
#   asn                  = 64512
#
#   lifecycle {
#     precondition {
#       condition = contains(local.sessions_at_1000_mbps, self.session_count)
#       error_message = "session_count must be a value advertised by data.megaport_nat_gateway_sessions for speed 1000"
#     }
#   }
# }
