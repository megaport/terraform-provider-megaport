# Examples

This folder contains examples that demonstrate the functionality of the Megaport Terraform Provider. 
To use each of these examples, you need to complete all the [Getting Started](https://registry.terraform.io/providers/megaport/megaport/latest/docs/guides/gettingstarted) Requirements.

Update the credentials in the `provider` block, which can be moved to its own file.

All examples assume you have prior knowledge and understanding of how to use Terraform.  

All Megaport resources built by Terraform will be visible in the Megaport Portal associated with the environment used.  

Here's a description of each example:
1. [single_port](single_port): Create a Single Megaport.
1. [two_ports_and_vxc](two_ports_and_vxc): Provision two Megaports in different locations linked by a VXC (Virtual Cross Connect)
1. [lag_port](lag_port): Create a LAG Port.
1. [cloud_to_cloud_aws_azure](cloud_to_cloud_aws_azure): Create a fully functional example network linking AWS & Azure 
   Cloud Service Providers via a MCR (Megaport Cloud Router). _This will only work when executed in Production_.
1. [cloud_to_cloud_aws_google](cloud_to_cloud_aws_google): Create a fully functional example network linking AWS & Google 
   Cloud Service Providers via a MCR (Megaport Cloud Router). _This will only work when executed in Production_.
1. [mcr_and_csp_vxcs](mcr_and_csp_vxcs): Create the Megaport portion of a Mult-Cloud, AWS, Azure, and GCP network.
1. [full_ecosystem](full_ecosystem): Generate all the resources exposed by the Terraform provider.
