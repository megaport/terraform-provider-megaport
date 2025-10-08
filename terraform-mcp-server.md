# Building an Agentic Network with Megaport and Terraform

In this tutorial, you'll learn how to use an AI agent powered by the Terraform MCP Server to intelligently script your networking setup. We'll start by generating configurations for individual Megaport resources, then combine them into a single "master prompt" to deploy a complex, interconnected environment.

This "prompt-to-provision" workflow allows you to build sophisticated network architectures using simple, natural language commands.

## Prerequisites

Before you begin, ensure you have the following set up and configured:

1. **Visual Studio Code** with the GitHub Copilot extension installed and Agent mode enabled. See the [official guide on setting up an MCP server in VSCode](https://code.visualstudio.com/docs/copilot/customization/mcp-servers).

2. **Terraform MCP Server** deployed and integrated with VS Code. Refer to the official HashiCorp documentation:
   - [Terraform MCP Server Main Page](https://developer.hashicorp.com/terraform/mcp-server)
   - [Deployment Guide](https://developer.hashicorp.com/terraform/mcp-server/deploy)
   - [Prompting Best Practices](https://developer.hashicorp.com/terraform/mcp-server/prompt)

## Part 1: Building Resources Individually

Let's start by generating the configuration for each resource separately. This approach is useful when you want to build your infrastructure piece by piece.

### 1. Megaport Port

A Port is your physical point of connection to the Megaport network.

**Prompt:**

```
Generate a Terraform resource block for a megaport_port named 'agentic-port'. It should have a port_speed of 1000 Mbps, a 1-month contract term, and be located in 'CoreSite SV1 - San Jose'. Also include the necessary provider configuration and a data source block to look up the location.
```

### 2. Megaport Cloud Router (MCR)

An MCR enables Layer 3 routing and connectivity between your different cloud and service provider connections.

**Prompt:**

```
Generate a Terraform resource block for a megaport_mcr named 'agentic-mcr'.
It should have a port_speed of 1000 Mbps and be located in 'CoreSite SV1 - San Jose'.
```

### 3. Aruba Megaport Virtual Edge (MVE)

An MVE is a virtual networking device that allows you to extend your network to the edge.

**Prompt:**

```
Generate Terraform configuration for an Aruba MVE. Include a data source block for megaport_mve_images to find the latest image for the 'Aruba' vendor. Then, create a megaport_mve resource named 'agentic-mve' in 'CoreSite SV1 - San Jose' with a product_size of 'small'. The resource must include a vendor_config block with vendor = "aruba" and placeholders for account_name and account_key, and it should reference the ID from the image data source.
```

### 4. Virtual Cross Connect (VXC)

A VXC is a Layer 2 connection that links your resources together on the Megaport network. This prompt assumes the MCR and MVE have already been defined.

**Prompt:**

```
Generate a Terraform resource block for a megaport_vxc named 'mcr-to-mve-vxc'. The VXC rate_limit should be 500 Mbps. Configure the a_end block with a requested_product_uid that points to the product_uid of megaport_mcr.agentic-mcr. Configure the b_end block with a requested_product_uid that points to the product_uid of megaport_mve.agentic-mve.
```

## Part 2: The Master Prompt

Now that you've seen how to build each piece, let's combine everything into a single, comprehensive prompt.

### The Complete Prompt

Copy and paste the following prompt into the GitHub Copilot chat window in VS Code:

```
Generate a complete, multi-file Terraform configuration using the megaport provider that deploys four distinct resources: a Port, an MCR, an Aruba MVE, and a VXC.

1. Provider and Variables: Include terraform, provider, variables.tf, and terraform.tfvars.example files. Configure the provider for the 'staging' environment and use variables for the access and secret keys.

2. Location Data: Create a single data source for megaport_location for 'CoreSite SV1 - San Jose'.

3. Megaport Port: Define a megaport_port named 'agentic-port' with a port_speed of 1000 Mbps and a 1-month contract.

4. Megaport Cloud Router (MCR): Define a megaport_mcr named 'agentic-mcr' with a port_speed of 1000 Mbps.

5. Aruba MVE: Define a data source for megaport_mve_images for the 'Aruba' vendor. Create a megaport_mve resource named 'agentic-mve' with a product_size of 'small'. It must include a vendor_config block with vendor = "aruba", placeholders for account_name and account_key, and use the image ID from the data source.

6. Virtual Cross Connect (VXC): Define a megaport_vxc named 'mcr-to-mve-vxc' with a rate_limit of 500 Mbps. Its a_end block must have a requested_product_uid referencing the MCR's product_uid. Its b_end block must have a requested_product_uid referencing the MVE's product_uid.

Ensure all resources correctly reference the location data source.
```

## Executing the Plan

After you submit the prompt, the AI agent will generate the necessary files.

### 1. Review the Generated Files

The AI should create several files, including `main.tf`, `variables.tf`, and `terraform.tfvars.example`. Open `main.tf` and verify that all four resource blocks are present and correctly configured.

### 2. Prepare for Deployment

Create a `terraform.tfvars` file from the `terraform.tfvars.example` template and populate it with your Megaport API credentials.

### 3. Initialize, Plan, and Apply

Run the standard Terraform workflow from your terminal:

```bash
# Initialize the Terraform providers
terraform init

# Review the execution plan
terraform plan

# Apply the configuration to provision the resources
terraform apply
```

After you confirm, Terraform will provision all four resources.

### 4. Verify in the Megaport Portal

Once the apply is complete, log in to the Megaport Portal. You should see all four new resources configured and connected as described in your prompt.

## Conclusion

You've successfully demonstrated a powerful agentic workflow. This "prompt-to-provision" capability dramatically accelerates the process of building and managing sophisticated network architectures.
