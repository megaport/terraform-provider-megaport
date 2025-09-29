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
Virtual Cross Connects (VXCs), Megaport Cloud Routers (MCRs), Megaport Virtual Edges (MVEs), and Partner VXCs.

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

## Partner Port Selection & Stability

When connecting to cloud service providers, you need to select the appropriate partner ports. Megaport provides two data sources for this purpose:

### Recommended: Cloud Port Lookup Data Source

The **`megaport_cloud_port_lookup`** data source is the recommended approach for selecting partner ports. It addresses common issues with partner port selection by:

- **Returning all matching ports** instead of just one, giving you full visibility and control
- **Supporting secure partner ports** for GCP, Oracle, and Azure connections requiring service keys
- **Providing proper validation** with clear error messages for connect types
- **Eliminating confusing warnings** by letting you choose the specific port you want

```terraform
# Find all available AWS ports and choose the best one
data "megaport_cloud_port_lookup" "aws_ports" {
  connect_type   = "AWSHC"
  company_name   = "AWS"
  location_id    = 123
  diversity_zone = "blue"
}

# Select based on your criteria - no more guessing!
locals {
  # Option 1: Use the first available port
  selected_port = data.megaport_cloud_port_lookup.aws_ports.ports[0]
  
  # Option 2: Choose by name pattern
  preferred_port = [
    for port in data.megaport_cloud_port_lookup.aws_ports.ports :
    port if can(regex("us-east-1", lower(port.product_name)))
  ][0]
  
  # Option 3: Select by lowest rank (best performance)
  best_port = [
    for port in data.megaport_cloud_port_lookup.aws_ports.ports :
    port if port.rank == min([for p in data.megaport_cloud_port_lookup.aws_ports.ports : p.rank]...)
  ][0]
}

resource "megaport_vxc" "aws_connection" {
  b_end = {
    requested_product_uid = local.selected_port.product_uid
  }
  # ... other configuration
}
```

#### Secure Partner Ports

For cloud providers requiring service keys (GCP, Oracle, Azure):

```terraform
# GCP with pairing key
data "megaport_cloud_port_lookup" "gcp_secure" {
  connect_type   = "GOOGLE"
  include_secure = true
  service_key    = var.gcp_pairing_key
  location_id    = 123
}

# Oracle with service key  
data "megaport_cloud_port_lookup" "oracle_secure" {
  connect_type   = "ORACLE"
  include_secure = true
  service_key    = var.oracle_service_key
  location_id    = 123
}
```

### Legacy: Partner Data Source

The `megaport_partner` data source is still supported but has limitations:

**Issues with the legacy approach:**
- Returns only one port with potential warnings if multiple matches are found
- No support for secure partner ports
- Less control over port selection
- Partner port UIDs may change over time as Megaport manages capacity, leading to unexpected warnings:

```
Warning: VXC B-End product UID is from a partner port, therefore it will not be changed.
```

**If you must use the legacy data source**, you can prevent warnings by explicitly specifying the `product_uid` once your connections are established:

```terraform
# Initial setup - use standard filtering
data "megaport_partner" "awshc" {
  connect_type    = "AWSHC"
  company_name    = "AWS"
  product_name    = "US East (N. Virginia) (us-east-1)"
  diversity_zone  = "blue"
  location_id     = 123
}

# After connections are established, update to use explicit product_uid
data "megaport_partner" "awshc" {
  product_uid     = "a1b2c3d4-5678-90ef-ghij-klmnopqrstuv"
  # Keep other filters for documentation purposes
}
```

### Migration Recommendation

**We strongly recommend migrating from `megaport_partner` to `megaport_cloud_port_lookup`** for new configurations. The new data source provides better reliability, clearer error handling, and support for modern cloud connectivity patterns.

## Cloud Port Lookup Data Source Reference

### Configuration

The `megaport_cloud_port_lookup` data source accepts the following arguments:

#### Optional Arguments

- `connect_type` (String) - Connection type. Must be one of: `AWS`, `AWSHC`, `AZURE`, `GOOGLE`, `ORACLE`, `IBM`, `OUTSCALE`, `TRANSIT`, `FRANCEIX`
- `location_id` (Number) - Filter by location ID
- `diversity_zone` (String) - Filter by diversity zone: `red` or `blue`
- `company_name` (String) - Filter by company name
- `vxc_permitted` (Boolean) - Filter by VXC permission (default: `true`)
- `include_secure` (Boolean) - Include secure partner ports (default: `false`)
- `service_key` (String, Sensitive) - Required for secure ports when `include_secure = true`

#### Computed Attributes

- `ports` (List) - Array of matching ports, each containing:
  - `product_uid` (String) - Port unique identifier
  - `product_name` (String) - Port name
  - `connect_type` (String) - Connection type
  - `company_uid` (String) - Company unique identifier  
  - `company_name` (String) - Company name
  - `diversity_zone` (String) - Diversity zone
  - `location_id` (Number) - Location ID
  - `speed` (Number) - Port speed in Mbps
  - `rank` (Number) - Port rank (lower = better)
  - `vxc_permitted` (Boolean) - VXC permission status
  - `is_secure` (Boolean) - Whether port requires service key
  - `secure_service_key` (String, Sensitive) - Service key (secure ports only)
  - `vlan` (Number) - VLAN ID (secure ports only)

### Advanced Usage Patterns

#### Error Handling and Validation

```terraform
data "megaport_cloud_port_lookup" "aws_ports" {
  connect_type = "AWS"
  location_id  = 3
}

# Validate ports are available
locals {
  has_ports = length(data.megaport_cloud_port_lookup.aws_ports.ports) > 0
}

# Use check blocks (Terraform 1.5+)
check "ports_available" {
  assert {
    condition = local.has_ports
    error_message = "No AWS ports available in location 3"
  }
}

# Conditional resource creation
resource "megaport_vxc" "conditional_connection" {
  count = local.has_ports ? 1 : 0
  # ... configuration
}
```

#### Multi-Region Deployments

```terraform
# Define regions
locals {
  regions = {
    primary = { location_id = 3, diversity_zone = "red" }
    backup  = { location_id = 5, diversity_zone = "blue" }
  }
}

# Get ports for each region
data "megaport_cloud_port_lookup" "aws_ports" {
  for_each = local.regions
  
  connect_type   = "AWSHC"
  location_id    = each.value.location_id
  diversity_zone = each.value.diversity_zone
}

# Create connections for each region
resource "megaport_vxc" "aws_connections" {
  for_each = local.regions
  
  product_name = "AWS-${each.key}"
  # ... configuration
  
  b_end = {
    requested_product_uid = data.megaport_cloud_port_lookup.aws_ports[each.key].ports[0].product_uid
  }
}
```

## Migration from megaport_partner

### Quick Migration Steps

1. **Replace data source name**: `megaport_partner` → `megaport_cloud_port_lookup`
2. **Update attribute access**: Add `.ports[0]` to access the first port
3. **Add validation**: Check port availability before use
4. **Update secure connections**: Use `include_secure` and `service_key`

### Migration Examples

#### Basic Migration

**Before:**
```terraform
data "megaport_partner" "aws_port" {
  connect_type = "AWS"
  location_id  = 3
}

resource "megaport_vxc" "connection" {
  b_end = {
    requested_product_uid = data.megaport_partner.aws_port.product_uid
  }
}
```

**After:**
```terraform
data "megaport_cloud_port_lookup" "aws_ports" {
  connect_type = "AWS"
  location_id  = 3
}

resource "megaport_vxc" "connection" {
  b_end = {
    requested_product_uid = data.megaport_cloud_port_lookup.aws_ports.ports[0].product_uid
  }
}
```

#### Secure Connection Migration

**Before (not possible):**
```terraform
# Secure connections required hardcoded UIDs
resource "megaport_vxc" "gcp_connection" {
  b_end = {
    requested_product_uid = "hardcoded-gcp-port-uid"
  }
  service_key = var.gcp_pairing_key
}
```

**After:**
```terraform
data "megaport_cloud_port_lookup" "gcp_secure" {
  connect_type   = "GOOGLE"
  include_secure = true
  service_key    = var.gcp_pairing_key
  location_id    = 3
}

resource "megaport_vxc" "gcp_connection" {
  b_end = {
    requested_product_uid = data.megaport_cloud_port_lookup.gcp_secure.ports[0].product_uid
  }
  service_key = var.gcp_pairing_key
}
```

#### Shared Port Selection

**Before:**
```terraform
data "megaport_partner" "aws_port" {
  connect_type = "AWS"
  location_id  = 3
}

# Multiple resources using same port
resource "megaport_vxc" "connection_1" {
  b_end = { requested_product_uid = data.megaport_partner.aws_port.product_uid }
}

resource "megaport_vxc" "connection_2" {
  b_end = { requested_product_uid = data.megaport_partner.aws_port.product_uid }
}
```

**After:**
```terraform
data "megaport_cloud_port_lookup" "aws_ports" {
  connect_type = "AWS"
  location_id  = 3
}

locals {
  selected_aws_port = data.megaport_cloud_port_lookup.aws_ports.ports[0].product_uid
}

resource "megaport_vxc" "connection_1" {
  b_end = { requested_product_uid = local.selected_aws_port }
}

resource "megaport_vxc" "connection_2" {
  b_end = { requested_product_uid = local.selected_aws_port }
}
```

### Connect Type Reference

| Connect Type | Description | Secure Support |
|--------------|-------------|----------------|
| `AWS` | Amazon Web Services Private VIF | No |
| `AWSHC` | AWS Hosted Connection | No |
| `AZURE` | Microsoft Azure ExpressRoute | Yes |
| `GOOGLE` | Google Cloud Partner Interconnect | Yes |
| `ORACLE` | Oracle FastConnect | Yes |
| `IBM` | IBM Cloud Direct Link | No |
| `OUTSCALE` | Outscale Direct Connection | No |
| `TRANSIT` | Megaport Internet | No |
| `FRANCEIX` | France-IX | No |

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