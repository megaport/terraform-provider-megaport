# Configuration

## Terraform Provider Registry
Installation and Configuration is normally via the Terraform Provider Registry

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
## Attribute Reference
 - `access_key` [**string**] - (Required) Your access key used to generate a token to authenticate API requests.
 - `secret_key` [**string**] - (Required) Your secret key used to generate a token to authenticate API requests.
 - `accept_purchase_terms` [**boolean**] - (Required) Indicates your acceptance of all terms for using Megaport's services.
 - `delete_ports` [**boolean**] - (Optional) Indicates whether to delete any Ports provisioned by Terraform.
 - `environment` [**string**] - (Optional) For details, see [Environments](https://registry.terraform.io/providers/megaport/megaport/l
atest/docs/guides/environments).

The default `environment` is Staging, which is the test platform. To make changes to production systems, set the `environment` to `production`.
