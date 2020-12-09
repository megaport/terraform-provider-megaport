---
page_title: "Examples"
subcategory: "Examples"
---

# Examples

These examples demonstrate the functionality of the Megaport Terraform Provider. 
To use each of these examples, you need to complete the [Getting Started Requirements](gettingstarted).  

Update the credentials in the `provider` block, which can be moved to its own file.
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

Here's a description of each example:
1. [single_port](example_single_port): Create a Single Megaport.
1. [two_ports_and_vxc](example_two_ports_and_vxc): Provision two Megaports in different locations linked by a VXC (Virtual Cross Connect)
1. [lag_port](example_lag_port): Create a LAG Port.
1. [cloud_to_cloud_aws_azure](example_multicloud_aws_azure): Create a fully functional example network linking AWS & Azure 
   Cloud Service Providers via a MCR (Megaport Cloud Router). _This will only work when executed in Production_.
1. [cloud_to_cloud_aws_google](example_multicloud_aws_google): Create a fully functional example network linking AWS & Google 
   Cloud Service Providers via a MCR (Megaport Cloud Router). _This will only work when executed in Production_.
1. [mcr_and_csp_vxcs](example_mcr_and_csp_vxcs): Create the Megaport portion of a Mult-Cloud, AWS, Azure, and GCP network.
1. [full_ecosystem](example_full_ecosystem): Generate all the resources exposed by the Terraform provider.
