---
page_title: "Megaport Provider"
description: |-
  The Megaport Terraform Provider.
---

# Megaport Terraform Provider

The `terraform-provider-megaport` or Megaport Terraform Provider lets you create and manage
Megaport's product and services using the [Megaport API](https://dev.megaport.com).

This provides an opportunity for true multi-cloud hybrid environments supported by Megaport's Software
Defined Network (SDN). Using the Terraform provider, you can create and manage Ports,
Virtual Cross Connects (VXCs), Megaport Cloud Routers (MCRs), MCR Prefix Filter Lists, Megaport Virtual Edges (MVEs), and Partner VXCs.

This provider is compatible with HashiCorp Terraform, and we have tested compatibility with OpenTofu and haven't seen issues.

The Megaport Terraform Provider is released as a tool for use with the Megaport API.

**Important:** The usage of the Megaport Terraform Provider constitutes your acceptance of the terms available
in the Megaport [Acceptable Use Policy](https://www.megaport.com/legal/acceptable-use-policy/) and
[Global Services Agreement](https://www.megaport.com/legal/global-services-agreement/).

## Documentation

Documentation is published on the [Terraform Provider Megaport](https://registry.terraform.io/providers/megaport/megaport/latest/docs) registry and the [OpenTofu Provider Megaport](https://search.opentofu.org/provider/megaport/megaport/latest) registry.

## Installation

### Terraform Installation

The preferred installation method is via the [Terraform Provider Megaport](https://registry.terraform.io/providers/megaport/megaport/latest/docs)
registry.

### OpenTofu Installation

For OpenTofu users, the provider is available via the [OpenTofu Registry](https://search.opentofu.org/provider/megaport/megaport/latest). No configuration changes are needed - use the same provider source as you would with Terraform.

## Configuration

The provider can be configured in the same way whether using HashiCorp Terraform or OpenTofu:

```terraform
terraform {
  required_providers {
    megaport = {
      source  = "megaport/megaport"
      version = "~> 1.3"
    }
  }
}

provider "megaport" {
  # Configuration options
  environment           = "production"
  access_key            = "your-access-key"
  secret_key            = "your-secret-key"
  accept_purchase_terms = true
}
```

## 🚨 NEW FEATURE: MCR Prefix Filter List Resources

### Enhanced MCR Management with Standalone Resources

The Megaport Terraform Provider now supports managing MCR prefix filter lists as individual resources, providing better lifecycle management and improved state handling compared to the previous inline approach.

#### ✅ New Standalone Approach (Recommended)

```terraform
# Create MCR without inline prefix filter lists
resource "megaport_mcr" "example" {
  product_name         = "my-mcr"
  port_speed          = 1000
  location_id         = 5
  contract_term_months = 12
  cost_centre         = "networking"
}

# Manage prefix filter lists as individual resources
resource "megaport_mcr_prefix_filter_list" "allow_private_ipv4" {
  mcr_id         = megaport_mcr.example.product_uid
  description    = "Allow private IPv4 networks"
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
      prefix = "192.168.0.0/16"
      ge     = 24
      le     = 32
    }
  ]
}

resource "megaport_mcr_prefix_filter_list" "allow_private_ipv6" {
  mcr_id         = megaport_mcr.example.product_uid
  description    = "Allow private IPv6 networks"
  address_family = "IPv6"
  
  entries = [
    {
      action = "permit"
      prefix = "fd00::/8"
      ge     = 48
      le     = 64
    }
  ]
}
```

#### ⚠️ Deprecated Inline Approach

```terraform
# ❌ DEPRECATED: Inline prefix filter lists (will be removed in future version)
resource "megaport_mcr" "deprecated_example" {
  product_name         = "my-mcr"
  port_speed          = 1000
  location_id         = 5
  contract_term_months = 12
  cost_centre         = "networking"
  
  # This approach is deprecated and will show warnings
  prefix_filter_lists = [
    {
      description    = "Allow private networks"
      address_family = "IPv4"
      entries = [
        {
          action = "permit"
          prefix = "10.0.0.0/8"
          ge     = 16
          le     = 24
        }
      ]
    }
  ]
}
```

### Benefits of Standalone Resources

- **Individual Lifecycle Management**: Each prefix filter list has its own Terraform state and lifecycle
- **Better Error Handling**: Failures in one list don't affect others
- **Enhanced Reusability**: Lists can be referenced and managed independently
- **Cleaner State**: Avoid complex nested object handling in Terraform state
- **Import Support**: Easy migration of existing lists using `terraform import`

### Migration Guide

#### Step 1: Inventory Existing Lists

Use the data source to see what prefix filter lists you currently have:

```terraform
data "megaport_mcr_prefix_filter_lists" "existing" {
  mcr_id = "your-mcr-uid-here"
}

output "current_lists" {
  value = data.megaport_mcr_prefix_filter_lists.existing.prefix_filter_lists
}
```

#### Step 2: Create Standalone Resources

For each existing list, create a corresponding resource:

```terraform
resource "megaport_mcr_prefix_filter_list" "migrated_list_1" {
  mcr_id         = "your-mcr-uid-here"
  description    = "Copy description from existing list"
  address_family = "IPv4"
  entries = [
    # Copy entries from existing configuration
  ]
}
```

#### Step 3: Import Existing Lists

Import each existing list to avoid recreation:

```bash
terraform import megaport_mcr_prefix_filter_list.migrated_list_1 mcr-uid:prefix-list-id
```

#### Step 4: Update MCR Resource

Remove the `prefix_filter_lists` attribute from your MCR resource and add a lifecycle rule:

```terraform
resource "megaport_mcr" "example" {
  product_name         = "my-mcr"
  port_speed          = 1000
  location_id         = 5
  contract_term_months = 12
  
  # Remove or comment out the old prefix_filter_lists attribute
  # prefix_filter_lists = [...]
  
  # Add lifecycle rule to prevent drift warnings
  lifecycle {
    ignore_changes = [prefix_filter_lists]
  }
}
```

#### Step 5: Verify Migration

Run `terraform plan` to ensure no unexpected changes are detected.

### Mixed Usage Prevention

The provider includes validation to prevent managing the same prefix filter lists through both methods simultaneously. If you attempt to use both inline and standalone management for the same MCR, you'll receive warnings about potential conflicts.

### Timeline for Deprecation

- **Current**: Inline `prefix_filter_lists` shows deprecation warnings
- **3 months**: Enhanced warnings encourage migration to standalone resources
- **6 months**: Removal of inline `prefix_filter_lists` support

### Troubleshooting and Best Practices

#### Common Issues

**MCR Resource Shows Drift with Standalone Resources**

If your MCR resource shows unexpected changes to `prefix_filter_lists` when using standalone resources, add a lifecycle rule:

```terraform
resource "megaport_mcr" "example" {
  # ... configuration ...
  
  lifecycle {
    ignore_changes = [prefix_filter_lists]
  }
}
```

**Mixed Usage Warning**

If you see warnings about mixed usage, ensure you're not managing the same prefix filter lists through both inline and standalone methods simultaneously.

**Import Format**

When importing existing prefix filter lists, use the format `mcr_uid:prefix_list_id`:

```bash
# Get the MCR UID and prefix list ID from the Megaport Portal or API
terraform import megaport_mcr_prefix_filter_list.example a1b2c3d4-5678-90ef-ghij-klmnopqrstuv:1234
```

#### Best Practices

- **Use Location IDs**: Always use location IDs instead of names for MCR placement (more stable)
- **Validate Prefix Ranges**: Ensure ge (greater than or equal) and le (less than or equal) values make sense for your prefix lengths
- **Group Related Lists**: Create logically grouped prefix filter lists for easier management
- **Test Migrations**: Always test migrations in a non-production environment first
- **Document Purposes**: Use descriptive names and descriptions for prefix filter lists

#### Example Production Configuration

```terraform
# Production MCR with standalone prefix filter lists
resource "megaport_mcr" "production" {
  product_name         = "prod-mcr-us-east"
  port_speed          = 2500
  location_id         = 1  # Use stable location ID
  contract_term_months = 12
  cost_centre         = "networking"
  
  resource_tags = {
    Environment = "production"
    Owner       = "network-team"
    Purpose     = "multi-cloud-connectivity"
  }
  
  lifecycle {
    ignore_changes = [prefix_filter_lists]
  }
}

# Allow internal corporate networks
resource "megaport_mcr_prefix_filter_list" "corporate_networks" {
  mcr_id         = megaport_mcr.production.product_uid
  description    = "Corporate internal networks"
  address_family = "IPv4"
  
  entries = [
    {
      action = "permit"
      prefix = "10.100.0.0/16"
      ge     = 24
      le     = 28
    },
    {
      action = "permit"
      prefix = "10.200.0.0/16"
      ge     = 24
      le     = 28
    }
  ]
}

# Allow cloud provider networks
resource "megaport_mcr_prefix_filter_list" "cloud_networks" {
  mcr_id         = megaport_mcr.production.product_uid
  description    = "AWS and Azure networks"
  address_family = "IPv4"
  
  entries = [
    {
      action = "permit"
      prefix = "172.16.0.0/12"
      ge     = 16
      le     = 24
    }
  ]
}

# IPv6 support for future expansion
resource "megaport_mcr_prefix_filter_list" "ipv6_networks" {
  mcr_id         = megaport_mcr.production.product_uid
  description    = "IPv6 corporate networks"
  address_family = "IPv6"
  
  entries = [
    {
      action = "permit"
      prefix = "2001:db8:100::/48"
      ge     = 56
      le     = 64
    }
  ]
}
```

## Local Development

### Set up a Go workspace

You don't need to do this if you're not modifying `megaportgo`, but if you need to modify it you can use a Go workspace to make this process easier. Take a look at [this tutorial](https://go.dev/doc/tutorial/workspaces) first to get familiar with how Go workspaces work, then create a workspace for local development. This will let you edit the megaportgo library while working on the Terraform Provider without needing to publish changes to Git or modify your go.mod file in the Terraform Provider with a replace statement.

```go.work
go 1.22.0
use (
	./megaportgo
	./terraform-provider-megaport
)
```

### Allow the provider to be run locally

First, find the GOBIN path where Go installs your binaries. Your path may vary depending on how your Go environment variables are configured.

```bash
$ go env GOBIN
/Users/<Username>/go/bin
```

If the GOBIN Go environment variable is not set, use the default path, `/Users/<Username>/go/bin`

Create a new file called `.terraformrc` in your home directory (~), then add the `dev_overrides` block below. Change the `<PATH>` to the value returned from the `go env GOBIN` command above.

```terraform
provider_installation {
  dev_overrides {
      "registry.terraform.io/megaport/megaport" = "<PATH>"
  }
  # For all other providers, install them directly from their origin provider
  # registries as normal. If you omit this, Terraform will _only_ use
  # the dev_overrides block, and so no other providers will be available.
  direct {}
}
```

Once you’ve done that you can test out changes to the provider by installing it with

```bash
go install .
```

inside of the `terraform-provider-megaport` folder

## Example Use Cases

A suite of tested examples is in the [examples](examples) directory

## Contributing

Contributions via pull request are welcome. Familiarize yourself with these guidelines to increase the likelihood of your pull request being accepted.

All contributions are subject to the [Megaport Contributor Licence Agreement](CLA.md).
The CLA clarifies the terms of the [Mozilla Public Licence 2.0](LICENSE) used to Open Source this respository and ensures that contributors are explictly informed of the conditions. Megaport requires all contributors to accept these terms to ensure that the Megaport Terraform Provider remains available and licensed for the community.

When you open a Pull Request, all authors of the contributions are required to comment on the Pull Request confirming
acceptance of the CLA terms. Pull Requests can not be merged until this is complete.

Megaport users are also bound by the [Acceptable Use Policy](https://www.megaport.com/legal/acceptable-use-policy).

## 🚨 BREAKING CHANGE: Location Data Source Migration

### ⚠️ URGENT: `site_code` Filtering No Longer Supported

**If you are using `site_code` to filter locations in your Terraform configurations, you must update your code immediately or your configurations will fail.**

The Megaport Location API has been upgraded to v3, and several important changes affect how you interact with location data:

#### ❌ What No Longer Works

```terraform
# THIS WILL FAIL - site_code filtering is no longer supported
data "megaport_location" "broken_example" {
  site_code = "NTT-TOK"  # ❌ This will cause an error
}
```

#### ✅ What You Should Use Instead

```terraform
# ✅ RECOMMENDED: Use location ID for most reliable results
data "megaport_location" "recommended" {
  id = 123  # IDs never change and are the most stable
}

# ✅ ALTERNATIVE: Use location name (may change over time)
data "megaport_location" "alternative" {
  name = "NextDC B1"  # Names can change, but currently supported
}

# 💡 TIP: Save the location ID for future use
output "location_id_for_nextdc_b1" {
  value = data.megaport_location.alternative.id
  description = "Location ID for NextDC B1 - save this for consistent future references"
}

# Then use the saved ID in future configurations:
# data "megaport_location" "stable_reference" {
#   id = 5  # Use the ID from the output above
# }
```

### 🔧 Migration Guide

**Step 1: Identify affected configurations**
Search your Terraform files for any usage of `site_code`:

```bash
grep -r "site_code" *.tf
```

**Step 2: Replace with `id` or `name`**

- **Best option**: Replace with the location `id` (most stable)
- **Alternative**: Replace with the location `name` (may change over time)

**Step 3: Update deprecated field usage**
Several location fields are now deprecated and will show warnings:

```terraform
# These fields are deprecated and will show warnings:
# - site_code (also no longer available for filtering)
# - campus
# - network_region
# - live_date
# - v_router_available
```

### 📋 Complete Migration Checklist

- [ ] Replace all `site_code = "..."` filters with `id = ...` or `name = "..."`
- [ ] Remove any code that depends on deprecated fields
- [ ] Test your configurations thoroughly
- [ ] Update any documentation or comments

### 🆘 Need Help?

If you need to find the location ID for a specific site code, you can:

1. **Use Terraform data source**: Query by name to get the ID:

   ```terraform
   data "megaport_location" "lookup" {
     name = "Your Location Name"
   }

   output "location_id" {
     value = data.megaport_location.lookup.id
   }
   ```

2. **Use the API directly**: Call `GET /v3/locations` to see all available locations
3. **Contact Support**: Megaport support can help map site codes to location IDs

---

## Datacenter Location Data Source

Locations for Megaport Data Centers can be retrieved using the Locations Data Source in the Megaport Terraform Provider.

**Current supported search methods:**

- `id` - **RECOMMENDED** (most reliable and stable)
- `name` - Alternative option (may change over time)

Examples:

```terraform
# ✅ RECOMMENDED: Use ID for most reliable results
data "megaport_location" "stable_example" {
  id = 5
}

# ✅ ALTERNATIVE: Use name (less stable, may change)
data "megaport_location" "name_example" {
  name = "NextDC B1"
}

# 💡 TIP: Save the location ID for future use
output "location_id_for_nextdc_b1" {
  value = data.megaport_location.name_example.id
  description = "Location ID for NextDC B1 - save this for consistent future references"
}
```

**Important:** Location IDs never change and provide the most reliable and deterministic behavior. Location names may be updated over time, which could cause Terraform configurations to break unexpectedly.

The most up-to-date listing of Megaport Datacenter Locations can be accessed through the Megaport API at `GET /v3/locations`

## Partner Port Stability

When using filter criteria to select partner ports (used to connect to cloud service providers), the specific partner port (and therefore UID) that best matches your filters may change over time as Megaport manages capacity by rotating ports. This can lead to unexpected warning messages during Terraform operations even when the VXCs themselves are not being modified:

```
Warning: VXC B-End product UID is from a partner port, therefore it will not be changed.
```

This warning appears because Terraform detects a difference in the partner port UID even when applying changes unrelated to those specific VXCs.

### Workaround

To prevent these warnings and ensure configuration stability, we recommend explicitly specifying the `product_uid` in your partner port data source once your connections are established:

```terraform
# Initial setup - use standard filtering to find the partner port
data "megaport_partner" "awshc" {
  connect_type    = "AWSHC"
  company_name    = "AWS"
  product_name    = "US East (N. Virginia) (us-east-1)"
  diversity_zone  = "blue"
  location_id     = 123
}

# After your connections are established, update your configuration
# to explicitly specify the product_uid for stability
data "megaport_partner" "awshc" {
  product_uid     = "a1b2c3d4-5678-90ef-ghij-klmnopqrstuv"
  # You can keep other filters for documentation purposes
  # but product_uid takes precedence
}
```

## End-of-Term Cancellation

By default, when Terraform deletes resources, they are immediately cancelled in the Megaport portal. However, you may prefer to have resources marked for cancellation at the end of their current billing term instead of immediate cancellation.

The provider supports this with the `cancel_at_end_of_term` configuration option:

```terraform
provider "megaport" {
  environment           = "production"
  access_key            = "your-access-key"
  secret_key            = "your-secret-key"
  accept_purchase_terms = true
  cancel_at_end_of_term = true  # Mark resources for end-of-term cancellation
}
```

**Important notes:**

- This feature is currently only supported for Single Ports and LAG Ports
- For other resource types, the option will be ignored and immediate cancellation will occur
- When `cancel_at_end_of_term` is set to `true`, resources will show as "CANCELLING" in the Megaport portal until the end of their billing term
- Resources are removed from Terraform state as soon as the API call returns successfully, regardless of whether immediate or end-of-term cancellation is used
- If you reapply your configuration after a resource has been deleted, Terraform will create a new resource, even if the original resource is still visible in the Megaport portal with "CANCELLING" status