Tutorial: Building an Agentic Network with Megaport and Terraform
In this tutorial, you will learn how to use an AI agent, powered by the Terraform MCP Server, to intelligently script your networking setup. We will start by generating configurations for individual Megaport resources and then combine them into a single, powerful "master prompt" to deploy a complex, interconnected environment.

This "prompt-to-provision" workflow allows you to build sophisticated network architectures using simple, natural language commands.

Prerequisites
Before you begin, ensure you have the following set up and configured:

Visual Studio Code: With the GitHub Copilot extension installed and Agent mode enabled. For more details, see the official guide on how to set up an MCP server in VSCode.

Terraform MCP Server: You must deploy and integrate the server with VS Code. For detailed instructions, refer to the official HashiCorp documentation:

Terraform MCP Server Main Page

Deployment Guide

Prompting Best Practices

Part 1: Building Resources Individually
Let's start by generating the configuration for each resource separately. This approach is useful when you want to build your infrastructure piece by piece. For each step, you will provide a new prompt to the AI agent.

1. Megaport Port
   A Port is your physical point of connection to the Megaport network.

Prompt:

"Generate a Terraform resource block for a megaport_port named 'agentic-port'. It should have a speed of 1000 Mbps, a 1-month contract term, and be located in 'CoreSite SV1 - San Jose'. Also include the necessary provider configuration and a data source block to look up the location."

2. Megaport Cloud Router (MCR)
   An MCR enables Layer 3 routing and connectivity between your different cloud and service provider connections.

Prompt:

"Generate a Terraform resource block for a megaport_mcr named 'agentic-mcr'. It should have a speed of 1000 Mbps and be located in 'CoreSite SV1 - San Jose'."

3. Aruba Megaport Virtual Edge (MVE)
   An MVE is a virtual networking device that allows you to extend your network to the edge. This prompt is more detailed as it requires finding the correct image and providing vendor-specific details.

Prompt:

"Generate Terraform configuration for an Aruba MVE. Include a data source block for megaport_mve_images to find the latest image for the 'Aruba' vendor. Then, create a megaport_mve resource named 'agentic-mve' in 'CoreSite SV1 - San Jose' with a mve_size of 'small'. The resource must include a vendor_config block with placeholders for account_name and account_key, and it should reference the ID from the image data source."

4. Virtual Cross Connect (VXC)
   A VXC is a Layer 2 connection that links your resources together on the Megaport network. This prompt assumes the MCR and MVE have already been defined in your configuration.

Prompt:

"Generate a Terraform resource block for a megaport_vxc named 'mcr-to-mve-vxc'. The VXC rate_limit should be 500 Mbps. Configure the a_end to connect to the product_uid of megaport_mcr.agentic-mcr and the b_end to connect to the product_uid of megaport_mve.agentic-mve."

Part 2: The Master Prompt: From Words to Infrastructure
Now that you've seen how to build each piece, let's combine everything into a single, comprehensive prompt. This allows the AI to understand the full context and dependencies between resources, generating a complete configuration in one step.

The Prompt
Copy and paste the following prompt into the GitHub Copilot chat window in VS Code:

"Generate a complete, multi-file Terraform configuration using the megaport provider that deploys four distinct resources: a Port, an MCR, an Aruba MVE, and a VXC.

Provider and Variables: The configuration should include terraform, provider, variables.tf, and terraform.tfvars.example files. Configure the provider for the 'staging' environment and use variables for the access and secret keys.

Location Data: Create a single data source to look up the megaport_location for 'CoreSite SV1 - San Jose'. All resources should be deployed in this location.

Megaport Port: Define a megaport_port resource named 'agentic-port'. It should have a speed of 1000 Mbps and a 1-month contract term.

Megaport Cloud Router (MCR): Define a megaport_mcr resource named 'agentic-mcr'. It should also have a speed of 1000 Mbps.

Aruba MVE: Define a data source for megaport_mve_images for the 'Aruba' vendor. Then create a megaport_mve resource named 'agentic-mve' with a mve_size of 'small'. It must include a vendor_config block with placeholders for account_name and account_key, and use the image ID from the data source.

Virtual Cross Connect (VXC): Define a megaport_vxc resource named 'mcr-to-mve-vxc' with a rate_limit of 500 Mbps. Its a_end must connect to the product_uid of the MCR, and its b_end must connect to the product_uid of the MVE.

Ensure that all resource blocks correctly reference the location data source and that the VXC resource correctly references the MCR and MVE resources."

Executing the Plan
After you submit the prompt, the AI agent will generate the necessary files.

1. Review the Generated Files
   The AI should create several files, including main.tf, variables.tf, and terraform.tfvars.example. Open main.tf and verify that all four resource blocks (megaport_port, megaport_mcr, megaport_mve, and megaport_vxc) are present and correctly configured.

2. Prepare for Deployment
   Create a terraform.tfvars file from the terraform.tfvars.example template and populate it with your Megaport API credentials.

3. Initialize, Plan, and Apply
   Run the standard Terraform workflow from your terminal:

# Initialize the Terraform providers

terraform init

# Review the execution plan

terraform plan

# Apply the configuration to provision the resources

terraform apply

After you confirm, Terraform will provision all four resources.

4. Verify in the Megaport Portal
   Once the apply is complete, log in to the Megaport Portal. You should see all four new resources configured and connected as described in your prompt.

Conclusion
You have successfully demonstrated a powerful agentic workflow. Whether you build piece by piece or deploy a full architecture at once, this "prompt-to-provision" capability dramatically accelerates the process of building and managing sophisticated network architectures.
