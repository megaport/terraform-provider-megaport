---
page_title: "MultiCloud AWS to Azure"
subcategory: "Examples"
---

# MultiCloud AWS to Azure
This examples provisions a full multi-cloud demonstration environment including networking and compute instances. It 
requires account credentials for Megaport, Amazon Web Services, and Azure.

This example requires some prior understanding of AWS and Azure platforms, as well as usage of SSH and key pairs.  

This example will create an Azure Bastion host which will allow you to logon to the Azure Portal using the
bastion functionality.

# Full Example
This example is split over multiple Terraform files to segregate AWS, Azure and Megaport resources.
Use the [Megaport GitHub repository](https://github.com/megaport/terraform-provider-megaport/tree/main/examples/cloud_to_cloud_aws_azure)
to obtain the full configuration.

