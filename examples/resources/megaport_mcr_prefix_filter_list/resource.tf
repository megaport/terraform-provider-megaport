# Basic IPv4 prefix filter list example
resource "megaport_mcr_prefix_filter_list" "ipv4_example" {
  mcr_id         = "11111111-1111-1111-1111-111111111111" # Replace with your MCR UID
  description    = "IPv4 Prefix Filter List Example"
  address_family = "IPv4"

  entries = [
    {
      action = "permit"
      prefix = "10.0.0.0/8"
      ge     = 16
      le     = 24
    },
    {
      action = "permit"
      prefix = "172.16.0.0/12"
      # ge and le are optional - if not specified, exact match on prefix length
    },
    {
      action = "deny"
      prefix = "192.168.1.0/24"
      ge     = 24
      le     = 32
    }
  ]
}

# IPv6 prefix filter list example
resource "megaport_mcr_prefix_filter_list" "ipv6_example" {
  mcr_id         = "11111111-1111-1111-1111-111111111111" # Replace with your MCR UID
  description    = "IPv6 Prefix Filter List Example"
  address_family = "IPv6"

  entries = [
    {
      action = "permit"
      prefix = "2001:db8::/32"
      ge     = 48
      le     = 64
    },
    {
      action = "deny"
      prefix = "2001:db8:bad::/48"
    }
  ]
}

# Multiple prefix filter lists on the same MCR
resource "megaport_mcr_prefix_filter_list" "inbound_filter" {
  mcr_id         = "11111111-1111-1111-1111-111111111111" # Replace with your MCR UID
  description    = "Inbound Traffic Filter"
  address_family = "IPv4"

  entries = [
    {
      action = "permit"
      prefix = "203.0.113.0/24"
      ge     = 24
      le     = 30
    },
    {
      action = "deny"
      prefix = "0.0.0.0/0"
      le     = 7 # Deny default routes and very broad prefixes
    }
  ]
}

resource "megaport_mcr_prefix_filter_list" "outbound_filter" {
  mcr_id         = "11111111-1111-1111-1111-111111111111" # Replace with your MCR UID
  description    = "Outbound Traffic Filter"
  address_family = "IPv4"

  entries = [
    {
      action = "permit"
      prefix = "10.0.0.0/8"
      ge     = 24
      le     = 32
    },
    {
      action = "permit"
      prefix = "172.16.0.0/12"
      ge     = 24
      le     = 32
    }
  ]
}

# Using with an existing MCR resource
resource "megaport_mcr" "example_mcr" {
  product_name         = "MCR for Prefix Filter Lists"
  port_speed           = 1000
  location_id          = 6
  contract_term_months = 1
}

resource "megaport_mcr_prefix_filter_list" "dynamic_example" {
  mcr_id         = megaport_mcr.example_mcr.product_uid
  description    = "Dynamic MCR Prefix Filter List"
  address_family = "IPv4"

  entries = [
    {
      action = "permit"
      prefix = "198.51.100.0/24"
      ge     = 28
      le     = 32
    }
  ]
}

# Output the prefix filter list ID for reference
output "prefix_filter_list_id" {
  description = "The ID of the created prefix filter list"
  value       = megaport_mcr_prefix_filter_list.ipv4_example.prefix_filter_list_id
}