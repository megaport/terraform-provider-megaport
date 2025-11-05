data "megaport_mve_images" "aruba" {
  vendor_filter = "Aruba"
  id_filter     = 23
}

resource "megaport_mve" "mve_aruba" {
  product_name         = "Megaport MVE Example"
  location_id          = 6
  contract_term_months = 1

  vnics = [
    {
      description = "Data Plane"
    },
    {
      description = "Control Plane"
    },
    {
      description = "Management Plane"
    }
  ]

  vendor_config = {
    vendor       = "aruba"
    product_size = "MEDIUM"
    image_id     = data.megaport_mve_images.aruba.mve_images.0.id
    account_name = "Aruba Test Account"
    account_key  = "12345678"
    system_tag   = "Preconfiguration-aruba-test-1"
  }
}

data "megaport_mve_images" "aviatrix" {
  vendor_filter = "Aviatrix"
  id_filter     = 70
}

# Sample Cloud Init Config for Aviatrix Edge - EXAMPLE ONLY
# cloud-config
# users:
# - lock-passwd: false
#   name: admin
#   passwd: $6$example$XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
# write_files:
# - content: '{"gateway_name": "EXAMPLE-GW", "controller_ip": "10.0.0.1", "launch_version":
#     "7.0.0", "dhcp": "False", "edge": "True", "caag": "False", "mgmt_interface_name":
#     "eth2", "mgmt_ip": "$PUBLIC_ADDRESS_WITH_MASK", "mgmt_default_gateway": "$PUBLIC_GATEWAY",
#     "ssh_public_key": "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCxample/DUMMY/KEY/PLACEHOLDER/VALUE/LONG/STRING/
#     example@localhost\n", "rollback": "False", "gateway_uuid": "00000000000000000000000000000000",
#     "trustdomain": "example.com", "trustbundle": "-----BEGIN CERTIFICATE-----\nEXAMPLE
#     CERTIFICATE PLACEHOLDER - REPLACE WITH YOUR ACTUAL CERTIFICATE\n-----END
#     CERTIFICATE-----\n", "tmp_svid": "-----BEGIN CERTIFICATE-----\nEXAMPLE
#     CERTIFICATE PLACEHOLDER - REPLACE WITH YOUR ACTUAL CERTIFICATE\n-----END
#     CERTIFICATE-----\n", "tmp_key": "-----BEGIN PRIVATE KEY-----\nEXAMPLE
#     PRIVATE KEY PLACEHOLDER - REPLACE WITH YOUR ACTUAL PRIVATE KEY\n-----END
#     PRIVATE KEY-----\n", "tmp_svid_expiry": "0000000000"}'
#   owner: ubuntu:ubuntu
#   path: /etc/cloudx/sample_config.cfg
#   permissions: '0755'

resource "megaport_mve" "mve_aviatrix" {
  product_name         = "Aviatrix-Edge"
  location_id          = 6
  contract_term_months = 12

  vendor_config = {
    vendor       = "aviatrix"
    image_id     = data.megaport_mve_images.aviatrix.mve_images.0.id
    product_size = "SMALL"
    mve_label    = "MVE 2/8"
    cloud_init   = "IyBjbG91ZC1jb25maWcKdXNlcnM6Ci0gbG9jay1wYXNzd2Q6IGZhbHNlCiAgbmFtZTogYWRtaW4KICBwYXNzd2Q6ICQ2JGV4YW1wbGUkWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYCndyaXRlX2ZpbGVzOgotIGNvbnRlbnQ6ICd7ImdhdGV3YXlfbmFtZSI6ICJFWEFNUExFLUdXIiwgImNvbnRyb2xsZXJfaXAiOiAiMTAuMC4wLjEiLCAibGF1bmNoX3ZlcnNpb24iOgogICAgIjcuMC4wIiwgImRoY3AiOiAiRmFsc2UiLCAiZWRnZSI6ICJUcnVlIiwgImNhYWciOiAiRmFsc2UiLCAibWdtdF9pbnRlcmZhY2VfbmFtZSI6CiAgICAiZXRoMiIsICJtZ210X2lwIjogIiRQVUJMSUNfQUREUkVTU19XSVRIX01BU0siLCAibWdtdF9kZWZhdWx0X2dhdGV3YXkiOiAiJFBVQkxJQ19HQVRFV0FZIiwKICAgICJzc2hfcHVibGljX2tleSI6ICJzc2gtcnNhIEFBQUFCM056YUMxeWMyRUFBQUFEQVFBQkFBQUJBUUN4YW1wbGUvRFVNTVkvS0VZL1BMQUNFSE9MREVSL1ZBTFVFL0xPTkcvU1RSSU5HLwogICAgZXhhbXBsZUBsb2NhbGhvc3RcbiIsICJyb2xsYmFjayI6ICJGYWxzZSIsICJnYXRld2F5X3V1aWQiOiAiMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAiLAogICAgInRydXN0ZG9tYWluIjogImV4YW1wbGUuY29tIiwgInRydXN0YnVuZGxlIjogIi0tLS0tQkVHSU4gQ0VSVElGSUNBVEUtLS0tLVxuRVhBTVBMRQogICAgQ0VSVElGSUNBVEUgUExBQ0VIT0xERVIgLSBSRVBMQUNFIFdJVEggWU9VUiBBQ1RVQUwgQ0VSVElGSUNBVEVcbi0tLS0tRU5ECiAgICBDRVJUSUZJQ0FURS0tLS0tXG4iLCAidG1wX3N2aWQiOiAiLS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tXG5FWEFNUExFCiAgICBDRVJUSUZJQ0FURSBQTEFDRUhPTERFUiAtIFJFUExBQ0UgV0lUSCBZT1VSIEFDVFVBTCBDRVJUSUZJQ0FURVxuLS0tLS1FTkQKICAgIENFUlRJRklDQVRFLS0tLS1cbiIsICJ0bXBfa2V5IjogIi0tLS0tQkVHSU4gUFJJVkFURSBLRVktLS0tLVxuRVhBTVBMRQogICAgUFJJVkFURSBLRVkgUExBQ0VIT0xERVIgLSBSRVBMQUNFIFdJVEggWU9VUiBBQ1RVQUwgUFJJVkFURSBLRVlcbi0tLS0tRU5ECiAgICBQUklWQVRFIEtFWS0tLS0tXG4iLCAidG1wX3N2aWRfZXhwaXJ5IjogIjAwMDAwMDAwMDAifScKICBvd25lcjogdWJ1bnR1OnVidW50dQogIHBhdGg6IC9ldGMvY2xvdWR4L3NhbXBsZV9jb25maWcuY2ZnCiAgcGVybWlzc2lvbnM6ICcwNzU1Jw==" # Base64 Encoded Cloud Init for Aviatrix Edge
  }
}

data "megaport_mve_images" "sixwind" {
  vendor_filter = "6wind"
}

# Example 1: 6WIND MVE with static vNICs configuration
# IMPORTANT: 6WIND requires a 2048-bit RSA SSH key. ED25519 and other key types are NOT supported.
# Generate a compatible key using: ssh-keygen -t rsa -b 2048 -C "your_email@example.com"
resource "megaport_mve" "mve_sixwind_static" {
  product_name         = "6WIND MVE Example - Static vNICs"
  location_id          = 6
  contract_term_months = 1

  vendor_config = {
    vendor       = "6wind"
    product_size = "SMALL"
    image_id     = data.megaport_mve_images.sixwind.mve_images[0].id
    # EXAMPLE RSA 2048-bit key - REPLACE WITH YOUR ACTUAL PUBLIC KEY
    ssh_public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDChMevHRnL3gDRXyGduArHROH8IkZhdVmVBLkR/0F6RhP7Jw6a8T3xGjFLvQj3jfvXDxKDfqRQvLLJ3CgqnLvHuQjVZ/vYGdFCCXSxYbg2fCj2VIUjPHOBqkBEG8a1HDx2P8qN6WD8nBHkLExampleKeyForDocumentationOnlyNotForUse example@example.com"
  }

  vnics = [
    {
      description = "ens4"
    },
    {
      description = "ens5"
    },
    {
      description = "ens6"
    },
    {
      description = "ens7"
    },
    {
      description = "ens8"
    }
  ]
}

# Example 2: 6WIND MVE with dynamic vNICs configuration using variables
# This pattern allows flexibility in the number of vNICs provisioned
variable "sixwind_vnic_count" {
  description = "Number of vNICs to provision for 6WIND MVE (1-5)"
  type        = number
  default     = 5
  validation {
    condition     = var.sixwind_vnic_count >= 1 && var.sixwind_vnic_count <= 5
    error_message = "vnic_count must be between 1 and 5"
  }
}

variable "sixwind_ssh_public_key" {
  description = "RSA 2048-bit SSH public key for 6WIND MVE (ED25519 not supported)"
  type        = string
  sensitive   = true
}

resource "megaport_mve" "mve_sixwind_dynamic" {
  product_name         = "6WIND MVE Example - Dynamic vNICs"
  location_id          = 3 # DigiCo Sydney SYD1 (Global Switch)	
  contract_term_months = 1

  vendor_config = {
    vendor         = "6wind"
    product_size   = "SMALL"
    image_id       = data.megaport_mve_images.sixwind.mve_images[0].id
    ssh_public_key = var.sixwind_ssh_public_key
  }

  # Dynamically generate vNICs based on the count variable
  # Interface names start at ens4 and increment (ens4, ens5, ens6, etc.)
  vnics = [for i in range(var.sixwind_vnic_count) : {
    description = "ens${i + 4}"
  }]
}


