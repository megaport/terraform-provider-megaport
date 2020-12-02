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

# Essentials
 To learn about the project essentials, read these topics:   
* [Environments](Environments.md) - Testing your Terraform before committing to a purchase
* [Getting Started](GettingStarted.md) - Creating your account  
* [Installation](Installation.md) - Setting up the Provider  
* [Configuration](Configuration.md) - Required configuration and provider authentication
* [Examples](https://github.com/megaport/terraform-provider-megaport/tree/main/examples) - A suite of tested examples are maintained in the repository

To manage your account, go to the 
[Megaport Portal](https://portal.megaport.com/). For information about the technical details of Megaport's 
offerings, explore the [Megaport Documentation](https://docs.megaport.com/).

The Megaport Terraform Provider is released as a tool for use with the Megaport API. It does not constitute
any part of the official paid product and is not eligible for support through customer channels.

**Important:** The usage of the Megaport Terraform Provider constitutes your acceptance of the terms available
in the Megaport [Acceptable Use Policy](https://www.megaport.com/legal/acceptable-use-policy/) and 
[Global Services Agreement](https://www.megaport.com/legal/global-services-agreement/).

# Data Sources & Resources

| **Data Sources**                                                      | **Resources**                               |
| ---                                                                   | ---                                                               |
| [megaport_location](data-sources/megaport_azure_connection.md)           |                                                                   |
| [megaport_port](data-sources/megaport_port.md)                           | [megaport_port](resources/megaport_port.md)                          |
| [megaport_mcr](data-sources/megaport_mcr.md)                             | [megaport_mcr](resources/megaport_mcr.md)                            |
| [megaport_vxc](data-sources/megaport_vxc.md)                             | [megaport_vxc](resources/megaport_vxc.md)                            |
| [megaport_aws_connection](data-sources/megaport_aws_connection.md)       | [megaport_aws_connection](resources/megaport_aws_connection.md)      |
| [megaport_azure_connection](data-sources/megaport_azure_connection.md)   | [megaport_azure_connection](resources/megaport_azure_connection.md)  |
| [megaport_gcp_connection](data-sources/megaport_gcp_connection.md)       | [megaport_gcp_connection](resources/megaport_gcp_connection.md)      |
