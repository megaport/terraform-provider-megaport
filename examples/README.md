# Examples

This folder contains examples that demonstrate the functionality of the Megaport Terraform Provider. To use each of these examples, you need to complete all the [Project Essentials](../../../wiki#essentials).  

Include the credentials in the `provider` block, which can be in its own file as per the [Configuration Documentation](../../../wiki/Configuration).

All examples assume you have prior knowledge and understanding of how to use Terraform.  

All Megaport resources built by Terraform will be visible in the Megaport Portal associated with the environment used.  

Here's a description of each example:
1. [single_port](single_port): Create a Single Megaport.
2. [two_ports_and_vxc](two_ports_and_vxc): Provision two Megaports in different locations linked by a VXC (Virtual Cross Connect)
3. [lag_port](lag_port): Create a LAG Port.
4. [cloud_to_cloud](cloud_to_cloud): Create a fully functional example network linking two Cloud Service Providers via a MCR (Megaport Cloud Router). 
_This will only work when executed in Production_.
5. [mcr_and_csp_vxcs](mcr_and_csp_vxcs): Create the Megaport portion of a Mult-Cloud, AWS, Azure, and GCP network.
6. [full_ecosystem](full_ecosystem): Generate all the resources exposed by the Terraform provider.
