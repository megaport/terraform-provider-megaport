---
layout: "megaport"
page_title: "Provider: Megaport"
description: |-
  The Megaport provider is used to interact with the many resources supported by Megaport. The provider needs to be configured with the proper credentials before it can be used.
---

# Megaport Terraform Provider

The `terraform-provider-megaport` or Megaport Terraform Provider lets you create and manage 
Megaport's product and services using the [Megaport API](https://dev.megaport.com).

This provides an opportunity for true multi-cloud hybrid environments supported by Megaport's Software 
Defined Network (SDN). Using the Terraform provider, you can create and manage Ports, Virtual Cross Connects (VXCs), Megaport Cloud Routers (MCRs), and Partner VXCs 
(for the full list, see [Resources](Resources_Overview)).

# Essentials
 To learn about the project essentials, read these topics:   
* [Environments](Environments) - Testing your Terraform before committing to a purchase
* [Getting Started](GettingStarted) - Creating your account  
* [Installation](Installation) - Setting up the Provider  
* [Configuration](Configuration) - Required configuration and provider authentication
* [Examples](Examples) - A suite of tested examples are maintained in the repository

To manage your account, go to the 
[Megaport Portal](https://portal.megaport.com/). For information about the technical details of Megaport's 
offerings, explore the [Megaport Documentation](https://docs.megaport.com/).

The Megaport Terraform Provider is released as a tool for use with the Megaport API. It does not constitute
any part of the official paid product and is not eligible for support through customer channels.

**Important:** The usage of the Megaport Terraform Provider constitutes your acceptance of the terms available
in the Megaport [Acceptable Use Policy](https://www.megaport.com/legal/acceptable-use-policy/) and 
[Global Services Agreement](https://www.megaport.com/legal/global-services-agreement/).

| **[Data Sources](Data_Sources_Overview)**                             | **[Resources](Resources_Overview)**                               |
| ---                                                                   | ---                                                               |
| [megaport_partner_port](data-sources/megaport_partner_port)           |                                                                   |
| [megaport_location](data-sources/megaport_azure_connection)           |                                                                   |
| [megaport_port](data-sources/megaport_port)                           | [megaport_port](resources/megaport_port)                          |
| [megaport_mcr](data-sources/megaport_mcr)                             | [megaport_mcr](resources/megaport_mcr)                            |
| [megaport_vxc](data-sources/megaport_vxc)                             | [megaport_vxc](resources/megaport_vxc)                            |
| [megaport_aws_connection](data-sources/megaport_aws_connection)       | [megaport_aws_connection](resources/megaport_aws_connection)      |
| [megaport_azure_connection](data-sources/megaport_azure_connection)   | [megaport_azure_connection](resources/megaport_azure_connection)  |
| [megaport_gcp_connection](data-sources/megaport_gcp_connection)       | [megaport_gcp_connection](resources/megaport_gcp_connection)      |
