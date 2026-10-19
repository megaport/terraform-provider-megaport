# List the products the company behind the provider credentials can order.
data "megaport_product_availability" "available" {}

# Variant and market codes for each orderable product.
output "orderable_products" {
  value = [
    for product in data.megaport_product_availability.available.products : {
      variant = product.variant
      markets = [for market in product.markets : market.market]
    } if product.category == "PRODUCT"
  ]
}

# Every add-on the company can order.
output "addons" {
  value = [
    for product in data.megaport_product_availability.available.products :
    product.display_name if product.category == "ADDON"
  ]
}
