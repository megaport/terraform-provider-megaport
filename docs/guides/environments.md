---
page_title: "Megaport Environments"
subcategory: "Essentials"
---

# Environments
Megaport maintains separate production and test API environments for you to use with this Provider and Megaport.

| Environment Name  | Variable Name | Portal URL                            | API URL                           | 
| ---               | ---           | ---                                   | ---                               |
| Production        | `production`  | https://portal.megaport.com/          | https://api.megaport.com/         |
| Staging           | `staging`     | https://portal-staging.megaport.com/  | https://api-staging.megaport.com/ |

## Production

The Production environment API commits actual changes on the Megaport network and you are responsible for any services and associated costs ordered in this environment.

To use the Production environment, set `environment = "production"` in your provider configuration.

## Staging

The Staging environment API does not commit any physical changes to the network. All API calls are made against a mock server. You can test orchestration of services in this Staging environment and the API calls and responses mirror the production system, but services will not be deployed and you will not be billed for any activity. All validation still occurs as per the Production API. 

**Note**: On the Staging API, no network resources are orchestrated in either Megaport's cloud network or with cloud provider network's, which makes this API unsuitable for testing integration with cloud providers.

Staging is the default target for the Megaport Terraform Provider; you do not need to set the `environment` variable to connect to the Staging API.

Content on the Staging environment is reset every 24 hours and replaced with a copy of the Production database.

## Further Reading

The Megaport API documentation contains more information in the [Environments](https://dev.megaport.com/#header-environments) section.
