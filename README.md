# Megaport Terraform Provider

The `terraform-provider-megaport` or Megaport Terraform Provider lets you create and manage
Megaport's product and services using the [Megaport API](https://dev.megaport.com).

This provides an opportunity for true multi-cloud hybrid environments supported by Megaport's Software
Defined Network (SDN). Using the Terraform provider, you can create and manage Ports,
Virtual Cross Connects (VXCs), Megaport Cloud Routers (MCRs), and Partner VXCs.

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

## Safe Delete Protection

The Megaport Terraform Provider includes automatic **Safe Delete Protection** to prevent accidental deletion of resources that have active dependencies. This feature is always enabled and works at the API level to protect your infrastructure.

### What is Safe Delete?

Safe Delete is a built-in safety mechanism that prevents you from deleting Ports, MCRs (Megaport Cloud Routers), or MVEs (Megaport Virtual Edge) that have VXCs (Virtual Cross Connects) attached to them. This prevents accidental service disruptions caused by deleting infrastructure that is actively in use.

### How It Works

When you attempt to delete a resource using Terraform:

1. The provider sends a delete request with the `safeDelete=true` parameter
2. The Megaport API checks if the resource has any attached VXCs
3. If VXCs are attached, the API returns an error and **prevents the deletion**
4. If no VXCs are attached, the deletion proceeds normally

### Protected Resources

Safe Delete protection applies to:

- **Single Ports** - Physical network ports
- **LAG Ports** - Link Aggregation Group ports
- **MCRs** - Megaport Cloud Routers
- **MVEs** - Megaport Virtual Edge instances

### Example Scenario

```terraform
# Create a port
resource "megaport_port" "my_port" {
  product_name = "My Production Port"
  port_speed   = 10000
  location_id  = 123
}

# Create a VXC attached to the port
resource "megaport_vxc" "my_vxc" {
  product_name = "My VXC"
  rate_limit   = 1000

  a_end {
    port_id = megaport_port.my_port.id
  }

  b_end {
    port_id = megaport_mcr.my_mcr.id
  }
}
```

If you try to run `terraform destroy -target=megaport_port.my_port` while the VXC is still attached, the operation will fail with an error:

```
Error: Could not delete port, unexpected error: Cannot delete product with active VXCs attached
```

### Proper Deletion Order

To successfully delete resources with dependencies, delete them in the correct order:

1. **First**: Delete all VXCs attached to the resource
2. **Then**: Delete the Port, MCR, or MVE

```bash
# Correct order for targeted deletions
terraform destroy -target=megaport_vxc.my_vxc      # Delete VXC first
terraform destroy -target=megaport_port.my_port    # Then delete the port

# Or simply destroy everything (Terraform handles dependencies automatically)
terraform destroy
```

When you run a full `terraform destroy`, Terraform automatically determines the correct deletion order based on resource dependencies, so you don't need to worry about the order.

### Benefits

Safe Delete protection provides several benefits:

- **Prevents service disruptions** - Can't accidentally delete infrastructure carrying active traffic
- **Enforces proper cleanup** - Forces you to delete VXCs before their parent resources
- **Automatic protection** - No configuration needed, always enabled
- **Clear error messages** - Tells you exactly why deletion failed

### Difference from Lifecycle Prevent Destroy

Safe Delete protection is different from Terraform's `lifecycle { prevent_destroy = true }` feature:

| Feature                       | Level        | When It Protects                               | Configuration Required |
| ----------------------------- | ------------ | ---------------------------------------------- | ---------------------- |
| **Safe Delete**               | API/Provider | When resource has attached VXCs                | None (always enabled)  |
| **Lifecycle prevent_destroy** | Terraform    | Always prevents deletion of specific resources | Yes (per resource)     |

**Safe Delete** is automatic dependency protection, while **lifecycle prevent_destroy** is optional protection for critical resources you specifically designate.

For more information about using lifecycle blocks to protect critical resources, see the [prevent_destroy example](examples/prevent_destroy/).

## Protecting Critical Resources with Lifecycle Prevent Destroy

In addition to the automatic Safe Delete protection, you can use Terraform's `lifecycle` block with `prevent_destroy = true` to explicitly protect critical production resources from accidental deletion.

### When to Use Prevent Destroy

Use `prevent_destroy` for resources that are critical to your infrastructure and should never be accidentally deleted:

- Production ports carrying live traffic
- MCRs routing traffic between multiple cloud providers
- VXCs connecting to critical services
- Any resource that would be expensive or time-consuming to recreate

### Example Usage

```terraform
resource "megaport_port" "production_port" {
  product_name           = "Production Port - Protected"
  port_speed             = 10000
  location_id            = data.megaport_location.my_location.id
  contract_term_months   = 12
  marketplace_visibility = false

  # Lifecycle block to prevent accidental destruction
  lifecycle {
    prevent_destroy = true
  }
}

resource "megaport_mcr" "production_mcr" {
  product_name         = "Production MCR - Protected"
  location_id          = data.megaport_location.my_location.id
  contract_term_months = 12

  router {
    port_speed    = 5000
    requested_asn = 64512
  }

  # Lifecycle block to prevent accidental destruction
  lifecycle {
    prevent_destroy = true
  }
}

resource "megaport_vxc" "production_vxc" {
  product_name         = "Production VXC - Protected"
  rate_limit           = 1000
  contract_term_months = 12

  a_end {
    port_id = megaport_port.production_port.id
  }

  b_end {
    port_id = megaport_mcr.production_mcr.id
  }

  # Lifecycle block to prevent accidental destruction
  lifecycle {
    prevent_destroy = true
  }
}
```

### How It Works

When you add `prevent_destroy = true` to a resource:

- ✅ Terraform allows creation and updates to the resource
- ❌ Terraform refuses to destroy the resource via `terraform destroy`
- ❌ Terraform refuses to destroy the resource when it's removed from configuration
- ❌ Terraform refuses targeted destroy attempts with `-target`

If you attempt to destroy a protected resource, Terraform will fail with an error:

```
Error: Instance cannot be destroyed

Resource megaport_port.production_port has lifecycle.prevent_destroy set,
but the plan calls for this resource to be destroyed.
```

### Removing Protection

To destroy a protected resource:

1. Remove the `lifecycle { prevent_destroy = true }` block from your configuration
2. Run `terraform apply` to update the configuration
3. Run `terraform destroy` to destroy the resource

### Best Practices

- **Apply to all production resources** - Add `prevent_destroy` to all critical infrastructure
- **Document protection reasons** - Use comments to explain why resources are protected
- **Combine with Portal locking** - Use Megaport Portal's [service locking feature](https://docs.megaport.com/portal-admin/locking/) for additional protection
- **Use version control** - Track changes to lifecycle blocks in Git and require code review
- **Separate environments** - Use different Terraform workspaces for production and non-production resources

### Comparison of Protection Features

| Feature             | Protection Level | When It Activates               | Configuration              |
| ------------------- | ---------------- | ------------------------------- | -------------------------- |
| **Safe Delete**     | API/Provider     | When resource has attached VXCs | Automatic (always enabled) |
| **prevent_destroy** | Terraform        | For any deletion attempt        | Manual (per resource)      |
| **Portal Locking**  | Megaport Portal  | Prevents all modifications      | Manual (via Portal)        |

For a complete example with detailed explanations, see the [prevent_destroy example](examples/prevent_destroy/).

For more information about Terraform lifecycle management, see:

- [Terraform Lifecycle Meta-Arguments](https://developer.hashicorp.com/terraform/language/meta-arguments/lifecycle)
- [Manage Resource Lifecycle Tutorial](https://developer.hashicorp.com/terraform/tutorials/state/resource-lifecycle)
- [Megaport Documentation - Terraform State Management](https://docs.megaport.com/cloud/terraform/state-management/)

```

```
