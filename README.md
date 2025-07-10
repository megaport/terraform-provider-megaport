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

## Datacenter Location Data Source

Locations for Megaport Data Centers can be retrieved using the Locations Data Source in the Megaport Terraform Provider.

They can be retrieved by searching by `id`, `name`, or `site_code`. **For the most reliable and deterministic behavior, we strongly recommend using the location `id` as your search criterion**, since IDs never change while names and site codes may be updated over time.

Examples:

```terraform
# Recommended approach - using ID (most reliable)
data "megaport_location" "my_location_3" {
  id = 5
}

# Alternative approaches - may be subject to change
data "megaport_location" "my_location_1" {
  name = "NextDC B1"
}

data "megaport_location" "my_location_2" {
  site_code = "bne_nxt1"
}
```

**Important:** Datacenter locations can change their name or site code in the API over time, which may cause Terraform configurations using these attributes to break unexpectedly. However, the numeric ID will always remain consistent in the Megaport API, ensuring your Terraform configurations remain stable.

The most up-to-date listing of Megaport Datacenter Locations can be accessed through the Megaport API at `GET /v2/locations`

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
- Resources are removed from Terraform state immediately, regardless of this setting
- If you re-apply your configuration after marking a resource for cancellation, Terraform will not attempt to recreate the resource as long as it remains in your state file
