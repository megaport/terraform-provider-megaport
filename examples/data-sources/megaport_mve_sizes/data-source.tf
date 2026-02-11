# Query all available MVE sizes from the Megaport API
data "megaport_mve_sizes" "available" {}

# Output all available sizes for reference
# Run `terraform plan` to see values without applying
output "available_mve_sizes" {
  description = "Map of MVE size labels to their product_size values"
  value = {
    for size in data.megaport_mve_sizes.available.mve_sizes :
    size.label => size.size
  }
}

# Example: Dynamically select the size string for a specific core count
locals {
  # Find the product_size value for a 32-core MVE
  mve_size_32_core = one([
    for s in data.megaport_mve_sizes.available.mve_sizes :
    s.size if s.cpu_core_count == 32
  ])

  # Find the product_size value for an 8-core MVE
  mve_size_8_core = one([
    for s in data.megaport_mve_sizes.available.mve_sizes :
    s.size if s.cpu_core_count == 8
  ])
}

# Example usage with megaport_mve resource
# resource "megaport_mve" "example" {
#   product_name         = "My MVE"
#   location_id          = 1234
#   contract_term_months = 1
#
#   vendor_config {
#     vendor       = "cisco"
#     image_id     = 123
#     product_size = local.mve_size_32_core  # Dynamically resolves to "X_LARGE_32"
#     # ... other vendor-specific config
#   }
# }
