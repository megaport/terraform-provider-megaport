# Configuration

## Terraform Provider Registry
Installation and Configuration is normally via the Terraform Provider Registry

```
terraform {
  required_providers {
    megaport = {
      source = "megaport/megaport"
      version = "0.1.1"
    }
  }
}

provider "megaport" {
    username                = "my.test.user@example.org"
    password                = "n0t@re4lPassw0rd"
    mfa_otp_key             = "ABCDEFGHIJK01234"
    accept_purchase_terms   = true
    delete_ports            = true
    environment             = "staging"
}
```
## Attribute Reference
 - `username` [**string**] - (Required) Your email address used to log in to the Megaport Portal.
 - `password` [**string**] - (Required) Your Megaport Portal password.
 - `mfa_otp_key` [**string**] - (Optional) The multi-factor authentication (MFA) key displayed in the Megaport Portal when you set up MFA on your account. For details, see [Requirements](https://registry.terraform.io/providers/megaport/megaport/latest/docs/guides/gettingstarted)).
 - `accept_purchase_terms` [**boolean**] - (Required) Indicates your acceptance of all terms for using Megaport's services.
 - `delete_ports` [**boolean**] - (Optional) Indicates whether to delete any Ports provisioned by Terraform.
 - `environment` [**string**] - (Optional) For details, see [Environments](https://registry.terraform.io/providers/megaport/megaport/l
atest/docs/guides/environments).

The default `environment` is Staging, which is the test platform. To make changes to production systems, set the `environment` to `production`.


