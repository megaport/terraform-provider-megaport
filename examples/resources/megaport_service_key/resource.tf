# Multi-use service key
resource "megaport_service_key" "multi_use" {
  product_uid = megaport_port.example.product_uid
  description = "Multi-use service key for partner"
  max_speed   = 500
  single_use  = false
  active      = true
}

# Single-use service key with VLAN
resource "megaport_service_key" "single_use" {
  product_uid = megaport_port.example.product_uid
  description = "Single-use key for specific connection"
  max_speed   = 100
  single_use  = true
  active      = true
  vlan        = 100
}

# Service key with validity period
resource "megaport_service_key" "time_limited" {
  product_uid  = megaport_port.example.product_uid
  description  = "Time-limited service key"
  max_speed    = 1000
  single_use   = false
  active       = true
  pre_approved = true

  valid_for = {
    start_time = "2025-01-01T00:00:00Z"
    end_time   = "2025-12-31T23:59:59Z"
  }
}
