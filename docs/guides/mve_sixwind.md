---
page_title: "6WIND Megaport Virtual Edge (MVE)"
description: |-
  6WIND Megaport Virtual Edge (MVE) Deployment
---

# 6WIND Megaport Virtual Edge (MVE)

This guide provides an example configuration for deploying a 6WIND Megaport Virtual Edge (MVE).

## Important Requirements

### SSH Key Requirements

6WIND MVE **requires a 2048-bit RSA SSH key**. Other key types such as ED25519, ECDSA, or RSA keys with different bit lengths (1024, 4096) are **not supported** and will cause provisioning failures.

To generate a compatible SSH key:

```bash
ssh-keygen -t rsa -b 2048 -C "your_email@example.com"
```

After generation, use the content of the `.pub` file (your public key) in the `ssh_public_key` field of the `vendor_config`.

### vNIC Configuration

6WIND MVE supports up to 5 vNICs. The network interfaces in the guest OS are named:
- `ens4` (first vNIC)
- `ens5` (second vNIC)
- `ens6` (third vNIC)
- `ens7` (fourth vNIC)
- `ens8` (fifth vNIC)

You can configure between 1 and 5 vNICs depending on your networking requirements. The examples below demonstrate both static and dynamic vNIC configuration approaches.

## Example Configuration

This example configuration creates a 6WIND MVE with both static and dynamic vNIC configurations.

```terraform
data "megaport_location" "sixwind_location" {
  name = "Global Switch Sydney West"
}

data "megaport_mve_images" "sixwind" {
  vendor_filter = "6wind"
}

# Example 1: Basic 6WIND MVE with static vNICs configuration
# IMPORTANT: 6WIND requires a 2048-bit RSA SSH key. ED25519 and other key types are NOT supported.
# Generate a compatible key using: ssh-keygen -t rsa -b 2048 -C "your_email@example.com"
resource "megaport_mve" "mve_sixwind" {
  product_name         = "6WIND MVE Example"
  location_id          = data.megaport_location.sixwind_location.id
  contract_term_months = 1

  vendor_config = {
    vendor       = "6wind"
    product_size = "SMALL"
    image_id     = data.megaport_mve_images.sixwind.mve_images[0].id
    # EXAMPLE RSA 2048-bit key - REPLACE WITH YOUR ACTUAL PUBLIC KEY
    # Generate using: ssh-keygen -t rsa -b 2048 -C "your_email@example.com"
    ssh_public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDChMevHRnL3gDRXyGduArHROH8IkZhdVmVBLkR/0F6RhP7Jw6a8T3xGjFLvQj3jfvXDxKDfqRQvLLJ3CgqnLvHuQjVZ/vYGdFCCXSxYbg2fCj2VIUjPHOBqkBEG8a1HDx2P8qN6WD8nBHkLExampleKeyForDocumentationOnlyNotForUse example@example.com"
  }

  # 6WIND supports up to 5 vNICs. Interface names in the guest OS are ens4, ens5, ens6, ens7, ens8
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
  product_name         = "6WIND MVE Dynamic Example"
  location_id          = data.megaport_location.sixwind_location.id
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
```

## MVE Documentation

For more information on creating and using a 6WIND Megaport Virtual Edge, additional documentation is available:
- [Megaport MVE Documentation](https://docs.megaport.com/mve/)
- [6WIND MVE Specific Documentation](https://docs.megaport.com/mve/6wind/)
