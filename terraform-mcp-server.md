# Tutorial: Building an Agentic Network with Megaport and Terraform

In this tutorial, you will learn how to use an AI agent, powered by the Terraform MCP Server, to intelligently script your networking setup. We will start by generating configurations for individual Megaport resources and then combine them into a single, powerful "master prompt" to deploy a complex, interconnected environment.

This "prompt-to-provision" workflow allows you to build sophisticated network architectures using simple, natural language commands.

### Prerequisites

Before you begin, ensure you have the following set up and configured:

1.  **Visual Studio Code**: With the GitHub Copilot extension installed and `Agent` mode enabled. For more details, see the official guide on [how to set up an MCP server in VSCode](https://code.visualstudio.com/docs/copilot/customization/mcp-servers).

2.  **Terraform MCP Server**: You must deploy and integrate the server with VS Code. For detailed instructions, refer to the official HashiCorp documentation:
    - [Terraform MCP Server Main Page](https://developer.hashicorp.com/terraform/mcp-server)
    - [Deployment Guide](https://developer.hashicorp.com/terraform/mcp-server/deploy)
    - [Prompting Best Practices](https://developer.hashicorp.com/terraform/mcp-server/prompt)

---

## Part 1: Building Resources Individually

Let's start by generating the configuration for each resource separately. This approach is useful when you want to build your infrastructure piece by piece. For each step, you will provide a new prompt to the AI agent.

### 1. Megaport Port

A Port is your physical point of connection to the Megaport network.

**Prompt:**

> "Using the Terraform MCP Server, look up the latest documentation for the `megaport_port` resource from the `megaport/megaport` provider in the Terraform Registry. Based on that documentation, generate a resource block named 'agentic-port' with a `product_name` of 'My Agentic Port', a `port_speed` of 1000 Mbps, a `contract_term_months` of 1, and `marketplace_visibility` set to false. The port should be located in 'NextDC B1', so also generate the necessary `data` source block to look up the location."

### 2. Megaport Cloud Router (MCR)

An MCR enables Layer 3 routing and connectivity between your different cloud and service provider connections.

**Prompt:**

> "Using the Terraform MCP Server, look up the latest documentation for the `megaport_mcr` resource from the `megaport/megaport` provider in the Terraform Registry. Then, generate a resource block named 'agentic-mcr' with a `product_name` of 'My Agentic MCR', a `port_speed` of 1000 Mbps, and a `contract_term_months` of 12, located in 'NextDC B1'."

### 3. Aruba Megaport Virtual Edge (MVE)

An MVE is a virtual networking device that allows you to extend your network to the edge.

**Prompt:**

> "Using the Terraform MCP Server, look up the `megaport_mve` resource from the `megaport/megaport` provider in the Terraform Registry. Generate a Terraform configuration for an Aruba MVE named 'agentic-mve' located in 'NextDC B1'. It must include a `data` source for `megaport_mve_images` to find the latest 'Aruba' image. The main resource block must contain a properly structured `vendor_config` block that includes a `product_size` of 'small', the `vendor` set to 'aruba', an `account_name` that references `var.aruba_account_name`, an `account_key` that references `var.aruba_account_key`, and the `image_id` referencing the ID from the image data source. Also generate the necessary variable definitions for the Aruba account details."

### 4. Virtual Cross Connect (VXC)

A VXC is a Layer 2 connection that links your resources together on the Megaport network.

**Prompt:**

> "Using the Terraform MCP Server, look up the `megaport_vxc` resource from the `megaport/megaport` provider in the Terraform Registry. Generate a resource block for a `megaport_vxc` named 'mcr-to-mve-vxc' with a `rate_limit` of 500 Mbps and a `contract_term_months` of 12. Configure its `a_end` block with a `requested_product_uid` pointing to the `product_uid` of a resource named `megaport_mcr.agentic-mcr`, and its `b_end` block with a `requested_product_uid` pointing to the `product_uid` of a resource named `megaport_mve.agentic-mve`."

---

## Part 2: The Master Prompt: From Words to Infrastructure

Now that you've seen how to build each piece, let's combine everything into a single, comprehensive prompt.

### The Prompt

Copy and paste the following prompt into the GitHub Copilot chat window in VS Code:

> "Using the Terraform MCP Server as the primary tool, generate a complete, multi-file Terraform configuration by looking up the latest documentation for the `megaport/megaport` provider in the Terraform Registry. The configuration must deploy four distinct resources: a Port, an MCR, an Aruba MVE, and a VXC.
>
> 1.  **Provider and Variables**: Include `terraform`, `provider`, `variables.tf`, and `terraform.tfvars.example` files. The `variables.tf` file must define variables for the Megaport access and secret keys, as well as for the Aruba `account_name` and `account_key`. All keys and secrets should be marked as sensitive.
> 2.  **Location Data**: Create a single `data` source for `megaport_location` for 'NextDC B1'. All resources must use this location.
> 3.  **Megaport Port**: Define a `megaport_port` resource named 'agentic-port' with a `product_name` of 'My Agentic Port', a `port_speed` of 1000 Mbps, `marketplace_visibility` set to false, and a `contract_term_months` of 1.
> 4.  **Megaport Cloud Router (MCR)**: Define a `megaport_mcr` resource named 'agentic-mcr' with a `product_name` of 'My Agentic MCR', a `port_speed` of 1000 Mbps, and a `contract_term_months` of 12.
> 5.  **Aruba MVE**: Define a `data` source for `megaport_mve_images` for the 'Aruba' vendor. Create a `megaport_mve` resource named 'agentic-mve'. Its `vendor_config` block must be correctly structured to contain a `product_size` of 'small', the `vendor` set to 'aruba', an `account_name` that references `var.aruba_account_name`, an `account_key` that references `var.aruba_account_key`, and the `image_id` from the data source.
> 6.  **Virtual Cross Connect (VXC)**: Define a `megaport_vxc` resource named 'mcr-to-mve-vxc' with a `rate_limit` of 500 Mbps and a `contract_term_months` of 12. Its `a_end` block must have a `requested_product_uid` referencing the MCR's `product_uid`, and its `b_end` block must have a `requested_product_uid` referencing the MVE's `product_uid`.
>
> Ensure all resource blocks correctly reference the location data source and that the VXC resource correctly references the MCR and MVE resources."

---

## Executing the Plan

After you submit the prompt, the AI agent will generate the necessary files.

### 1. Review the Generated Files

The AI should create several files, including `main.tf`, `variables.tf`, and `terraform.tfvars.example`. Open `variables.tf` and confirm that it contains definitions for `megaport_access_key`, `megaport_secret_key`, `aruba_account_name`, and `aruba_account_key`.

### 2. Prepare for Deployment

Create your `terraform.tfvars` file and add all four required variables:

```hcl
megaport_access_key = "your-megaport-access-key"
megaport_secret_key = "your-megaport-secret-key"
aruba_account_name  = "your-aruba-account-name"
aruba_account_key   = "your-aruba-account-key"
```

**Security reminder:** Never commit `terraform.tfvars` to version control. Add it to your `.gitignore` file.

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
