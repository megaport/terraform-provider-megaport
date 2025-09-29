# Find all AWS ports in Sydney
data "megaport_cloud_port_lookup" "aws_ports_sydney" {
  connect_type = "AWS"
  location_id  = 3 # Sydney location
}

# Select the first available AWS port
locals {
  selected_aws_port = data.megaport_cloud_port_lookup.aws_ports_sydney.ports[0]
}

# Find AWS Hosted Connection ports in specific diversity zone
data "megaport_cloud_port_lookup" "aws_hc_red_zone" {
  connect_type   = "AWSHC"
  location_id    = 3
  diversity_zone = "red"
}

# Filter by company name and select based on custom criteria
data "megaport_cloud_port_lookup" "aws_company_specific" {
  connect_type = "AWS"
  company_name = "AWS"
  location_id  = 3
}

locals {
  # Select port with highest speed
  highest_speed_port = [
    for port in data.megaport_cloud_port_lookup.aws_company_specific.ports :
    port if port.speed == max([for p in data.megaport_cloud_port_lookup.aws_company_specific.ports : p.speed]...)
  ][0]

  # Select port containing "Sydney" in the name
  sydney_port = [
    for port in data.megaport_cloud_port_lookup.aws_company_specific.ports :
    port if contains(lower(port.product_name), "sydney")
  ][0]
}

# Example with secure ports (GCP with pairing key)
data "megaport_cloud_port_lookup" "gcp_secure_ports" {
  connect_type   = "GOOGLE"
  include_secure = true
  service_key    = "your-gcp-pairing-key-here"
  location_id    = 3
}

# Use the cloud port in a VXC resource
resource "megaport_vxc" "aws_connection" {
  product_name         = "My AWS Connection"
  rate_limit           = 1000
  contract_term_months = 12

  a_end = {
    requested_product_uid = megaport_port.my_port.product_uid
  }

  b_end = {
    requested_product_uid = local.selected_aws_port.product_uid
  }

  b_end_partner_config = {
    partner = "aws"
  }
}