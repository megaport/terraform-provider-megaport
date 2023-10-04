---
page_title: "Provider: Megaport"
---

# Megaport Terraform Provider

The `terraform-provider-megaport` or Megaport Terraform Provider lets you create and manage 
Megaport's product and services using the [Megaport API](https://dev.megaport.com).

This provides an opportunity for multi-cloud or cloud to DC hybrid environments supported by Megaport's Software 
Defined Network (SDN). Using the Terraform provider, you can create and manage Ports, Virtual Cross Connects (VXCs), 
Megaport Cloud Routers (MCRs), and Partner VXCs 

## Essentials
To learn about Megaport essentials, read these guides:   
* [Environments](guides/environments) - Testing your Terraform before committing to a purchase
* [Getting Started](guides/gettingstarted) - Creating your account and generating an API key
* [Examples](guides/examples) - A suite of 
  tested examples are maintained in the guides

->**Note:** The Megaport Terraform Provider is released as a tool for use with the Megaport API. It does not constitute
part of the official product and is not eligible for support through customer channels.

~>**Important:** The usage of the Megaport Terraform Provider constitutes your acceptance of the terms available
in the Megaport [Acceptable Use Policy](https://www.megaport.com/legal/acceptable-use-policy/) and 
[Global Services Agreement](https://www.megaport.com/legal/global-services-agreement/).

## Installation & Configuration

Setting up the provider configuration for the Megaport Terraform Provider is a simple process.
```
terraform {
  required_providers {
    megaport = {
      source  = "megaport/megaport"
      version = ">=0.3.0"
    }
  }
}

provider "megaport" {
  access_key            = "my-access-key"
  secret_key            = "my-secret-key"
  accept_purchase_terms = true
  delete_ports          = true
  environment           = "staging"
}
```
### Configuration Reference

 - `access_key` [**string**] - (Required) Your access key used to generate a token to authenticate API requests. This can also be provided by the `MEGAPORT_ACCESS_KEY` environment variable.
 - `secret_key` [**string**] - (Required) Your secret key used to generate a token to authenticate API requests. This can also be provided by the `MEGAPORT_SECRET_KEY` environment variable.
 - `accept_purchase_terms` [**boolean**] - (Required) Indicates your acceptance of all terms for using Megaport's services.
 - `delete_ports` [**boolean**] - (Optional) Indicates whether to delete any Ports provisioned by Terraform.
 - `environment` [**string**] - (Optional) For details, see [Environments](guides/environments). This can also be provided by the `MEGAPORT_ENVIRONMENT` environment variable.

The default `environment` is Staging, which is the test platform. To make changes to production systems, set the `environment` to `production`.

## Example Usage

See the [examples](guides/examples) for more detailed examples including Cloud Service Providers and MCR configuration.

### Simple Port Example
```
data "megaport_location" "bne_nxt1" {
  name    = "NextDC B1"
  has_mcr = false
}

resource "megaport_port" "port" {
  port_name   = "Terraform Example - Port"
  port_speed  = 1000
  location_id = data.megaport_location.bne_nxt1.id
  term        = 1
}
```

### Environment Variable Example
```
export MEGAPORT_ACCESS_KEY="my-access-key"
export MEGAPORT_SECRET_KEY="my-secret-key"
export MEGAPORT_ENVIRONMENT="staging"
```
