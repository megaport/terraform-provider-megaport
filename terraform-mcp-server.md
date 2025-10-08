# Building an Agentic Network with Megaport and Terraform

In this tutorial, you'll learn how to use an AI agent powered by the **Terraform MCP Server** to intelligently script your networking setup. We'll start by generating configurations for individual Megaport resources, then combine them into a single, powerful "master prompt" to deploy a complex, interconnected environment.

This "prompt-to-provision" workflow allows you to build sophisticated network architectures using simple, natural language commands.

---

## Prerequisites

Before you begin, ensure you have the following set up and configured:

1. **Visual Studio Code** with the GitHub Copilot extension installed and **Agent mode enabled**

   - [VSCode MCP Server Setup Guide](https://code.visualstudio.com/docs/copilot/customization/mcp-servers)

2. **Terraform MCP Server** deployed and integrated with VS Code

   - [Terraform MCP Server Main Page](https://developer.hashicorp.com/terraform/mcp-server)
   - [Deployment Guide](https://developer.hashicorp.com/terraform/mcp-server/deploy)
   - [Prompting Best Practices](https://developer.hashicorp.com/terraform/mcp-server/prompt)

3. **Megaport API Credentials** - You'll need access and secret keys for authentication
   - Sign up or log in to the [Megaport Portal](https://portal.megaport.com)
   - Generate API credentials from your account settings

---

## Configuring the Megaport Provider

Before building resources, you need to configure the Megaport Terraform provider. The AI agent should generate this configuration, but it's important to understand what's required.

### Provider Configuration

The Megaport provider requires the following configuration:

```hcl
terraform {
  required_providers {
    megaport = {
      source  = "megaport/megaport"
      version = "~> 1.4"
    }
  }
}

provider "megaport" {
  environment            = "staging"  # Use "production" for live resources
  access_key            = var.megaport_access_key
  secret_key            = var.megaport_secret_key
  accept_purchase_terms = true
}
```

### Variables Configuration

Create a `variables.tf` file to store your credential variables:

```hcl
variable "megaport_access_key" {
  description = "Megaport API Access Key"
  type        = string
  sensitive   = true
}

variable "megaport_secret_key" {
  description = "Megaport API Secret Key"
  type        = string
  sensitive   = true
}
```

### Setting Your Credentials

Create a `terraform.tfvars` file (never commit this to version control):

```hcl
megaport_access_key = "your-actual-access-key"
megaport_secret_key = "your-actual-secret-key"
```

**Important Notes:**

- Use `environment = "staging"` for testing to avoid charges on production resources
- Set `accept_purchase_terms = true` to acknowledge Megaport's terms of service
- Always mark credentials as `sensitive = true` in variable declarations
- Add `terraform.tfvars` to your `.gitignore` file to prevent credential exposure

---

## Part 1: Building Resources Individually

Let's start by generating the configuration for each resource separately. This approach is useful when you want to build your infrastructure piece by piece. For each step, provide a new prompt to the AI agent.

### 1. Megaport Port

A **Port** is your physical point of connection to the Megaport network.

**Prompt:**

```
Generate a Terraform resource block for a megaport_port named 'agentic-port'.
It should have a speed of 1000 Mbps, a 1-month contract term, and be located in
'CoreSite SV1 - San Jose'. Also include the necessary provider configuration and a
data source block to look up the location.
```

### 2. Megaport Cloud Router (MCR)

An **MCR** enables Layer 3 routing and connectivity between your different cloud and service provider connections.

**Prompt:**

```
Generate a Terraform resource block for a megaport_mcr named 'agentic-mcr'.
It should have a speed of 1000 Mbps and be located in 'CoreSite SV1 - San Jose'.
```

### 3. Aruba Megaport Virtual Edge (MVE)

An **MVE** is a virtual networking device that allows you to extend your network to the edge. This prompt is more detailed as it requires finding the correct image and providing vendor-specific details.

**Prompt:**

```
Generate Terraform configuration for an Aruba MVE. Include a data source block
for megaport_mve_images to find the latest image for the 'Aruba' vendor. Then,
create a megaport_mve resource named 'agentic-mve' in 'CoreSite SV1 - San
Jose' with a mve_size of 'small'. The resource must include a vendor_config block
with placeholders for account_name and account_key, and it should reference the
ID from the image data source.
```

### 4. Virtual Cross Connect (VXC)

A **VXC** is a Layer 2 connection that links your resources together on the Megaport network. This prompt assumes the MCR and MVE have already been defined in your configuration.

**Prompt:**

```
Generate a Terraform resource block for a megaport_vxc named 'mcr-to-mve-vxc'.
The VXC rate_limit should be 500 Mbps. Configure the a_end to connect to the
product_uid of megaport_mcr.agentic-mcr and the b_end to connect to the
product_uid of megaport_mve.agentic-mve.
```

---

## Part 2: The Master Prompt — From Words to Infrastructure

Now that you've seen how to build each piece, let's combine everything into a single, comprehensive prompt. This allows the AI to understand the full context and dependencies between resources, generating a complete configuration in one step.

### The Master Prompt

Copy and paste the following prompt into the **GitHub Copilot chat window** in VS Code:

```
Generate a complete, multi-file Terraform configuration using the megaport
provider that deploys four distinct resources: a Port, an MCR, an Aruba MVE, and
a VXC.

1. Provider and Variables: The configuration should include terraform,
   provider, variables.tf, and terraform.tfvars.example files. Configure the
   provider for the 'staging' environment and use variables for the access and
   secret keys.

2. Location Data: Create a single data source to look up the
   megaport_location for 'CoreSite SV1 - San Jose'. All resources should be
   deployed in this location.

3. Megaport Port: Define a megaport_port resource named 'agentic-port'. It
   should have a speed of 1000 Mbps and a 1-month contract term.

4. Megaport Cloud Router (MCR): Define a megaport_mcr resource named
   'agentic-mcr'. It should also have a speed of 1000 Mbps.

5. Aruba MVE: Define a data source for megaport_mve_images for the 'Aruba'
   vendor. Then create a megaport_mve resource named 'agentic-mve' with a
   mve_size of 'small'. It must include a vendor_config block with placeholders
   for account_name and account_key, and use the image ID from the data
   source.

6. Virtual Cross Connect (VXC): Define a megaport_vxc resource named
   'mcr-to-mve-vxc' with a rate_limit of 500 Mbps. Its a_end must connect to
   the product_uid of the MCR, and its b_end must connect to the product_uid
   of the MVE.

Ensure that all resource blocks correctly reference the location data source and
that the VXC resource correctly references the MCR and MVE resources.
```

---

## Executing the Plan

After you submit the prompt, the AI agent will generate the necessary files.

### Step 1: Review the Generated Files

The AI should create several files, including:

- `main.tf`
- `variables.tf`
- `terraform.tfvars.example`

Open `main.tf` and verify that all four resource blocks (`megaport_port`, `megaport_mcr`, `megaport_mve`, and `megaport_vxc`) are present and correctly configured.

### Step 2: Prepare for Deployment

Review the generated `terraform.tfvars.example` file and create your actual `terraform.tfvars` file with your Megaport API credentials:

```hcl
megaport_access_key = "your-actual-access-key"
megaport_secret_key = "your-actual-secret-key"
```

**Security reminder:** Never commit `terraform.tfvars` to version control. Add it to your `.gitignore` file.

### Step 3: Initialize, Plan, and Apply

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

### Step 4: Verify in the Megaport Portal

Once the apply is complete, log in to the **Megaport Portal**. You should see all four new resources configured and connected as described in your prompt.

---

## Conclusion

You've successfully demonstrated a powerful agentic workflow. Whether you build piece by piece or deploy a full architecture at once, this "prompt-to-provision" capability dramatically accelerates the process of building and managing sophisticated network architectures.

With this approach, you can:

- Rapidly prototype network designs using natural language
- Reduce configuration errors through AI-assisted generation
- Iterate quickly on complex multi-resource deployments
- Focus on network architecture rather than syntax

Experiment with different prompts and resource combinations to discover the full potential of agentic infrastructure provisioning!
