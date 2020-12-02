# Configuration
Setting up the provider configuration for the Megaport Terraform Provider is a simple process. To start, specify the following values:

 - `username` [**string**] - (Required) Your email address used to log in to the Megaport Portal.
 - `password` [**string**] - (Required) Your Megaport Portal password.
 - `mfa_otp_key` [**string**] - (Optional) The multi-factor authentication (MFA) key displayed in the Megaport Portal when you set up MFA on your account. For details, see [Requirements](GettingStarted)).
 - `accept_purchase_terms` [**boolean**] - (Required) Indicates your acceptance of all terms for using Megaport's services.
 - `delete_ports` [**boolean**] - (Optional) Indicates whether to delete any Ports provisioned by Terraform.
 - `environment` [**string**] - (Optional) For details, see [Environments](Environments).

The default `environment` is Staging, which is the test platform. To make changes to production systems, set the `environment` to `production`.

### Terraform version 0.13 or higher  
Locally-hosted plugins require a mapping block in your Terraform code. You can include the mapping block with the `provider` block or elsewhere:
```
terraform {
  required_providers {
    megaport = {
      source  = "megaport.com/megaport/megaport"
      version = "0.1.0"
    }
  }
}
```

### Example
Include the following provider block in your Terraform project. The block can be located in its own file, for example `megaport_provider.tf`.

```
provider "megaport" {
    username                = "my.test.user@example.org"
    password                = "n0t@re4lPassw0rd"
    mfa_otp_key             = "ABCDEFGHIJK01234"
    accept_purchase_terms   = true
    delete_ports            = true
    environment             = "staging"
}
```

### Security

It is strongly discouraged to commit secrets to Git or other code repositories. Ensure that you use a Secrets Manager or the `.gitignore` technique to securely manage your credentials.

