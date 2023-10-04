---
page_title: "Getting Started"
subcategory: "Essentials"
---

# Getting Started
Getting Started with `terraform-provider-megaport` is easy! 

## Requirements
Before using the Megaport Terraform Provider, you will need a valid Megaport account and an Active API Key.

To create a new account, follow the [Quick Start Guide](https://docs.megaport.com/getting-started/). Complete
the [Create an account](https://docs.megaport.com/setting-up/registering/), 
[Add a company profile](https://docs.megaport.com/setting-up/registering/#adding-a-company-profile) and 
[Specify your billing market](https://docs.megaport.com/setting-up/registering/#enabling-a-billing-market) procedures.

To create an API Key, follow the steps in [Creating an API Key](https://docs.megaport.com/api/api-key/). Once you have an Active API Key add the `access_key` and `secret_key` into the provider configuration. The Megaport Terraform Provider will use these keys to generate an access token for authentication.
->**Note:** API keys are only valid for the environment they were generated in. Separate API keys are required for staging and production environments.

## Installation

Include the Terraform provider block in your code and run `terraform init` to download the provider.

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
