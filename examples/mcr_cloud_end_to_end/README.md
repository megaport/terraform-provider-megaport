This example deploys end-to-end multicloud connectivity between an AWS VPC, Azure VNet, and Google Cloud VPC using the Megaport Cloud Router (MCR). Test VMs are deployed in each cloud running the a Gatus test agent, latency and reachablity stats are available on a dashboard for each VM.

* Megaport Cloud Router (MCR)
* Megaport Virtual Cross Connect (VXC) to AWS Direct Connect
* AWS Direct Connect Hosted Connection
* AWS Direct Connect Gateway
* AWS Virtual Private Gateway
* AWS Direct Connect to Virtual Private Gateway Association
* AWS VPC
* AWS VPC Subnet
* AWS EC2 VM instance running a Gatus test agent
* Megaport Virtual Cross Connect (VXC) to Azure ExpressRoute
* Azure Resource Group
* Azure ExpressRoute Circuit
* Azure ExpressRoute Virtual Gateway
* Azure ExpressRoute to ExpressRoute Virtual Gateway Connection
* Azure VNet
* Azure VNet Subnet
* Azure VM running a Gatus test agent
* Megaport Virtual Cross Connect (VXC) to Google Cloud Partner Interconnect VLAN Attachment
* Google Cloud Partner Interconnect VLAN Attachment
* Google Cloud Router
* Google Cloud VPC
* Google Cloud VPC Subnet
* Google Cloud Compute VM running a Gatus test agent

### Prerequisites

* Install Terraform locally on Mac, Linux, or Windows: [Link](https://developer.hashicorp.com/terraform/tutorials/azure-get-started/install-cli)
* Create Megaport API Key: [Link](https://docs.megaport.com/api/api-key/)
* Create AWS Access Key: [Link](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_access-keys.html#Using_CreateAccessKey)
* Install Azure CLI Tools: [Link](https://developer.hashicorp.com/terraform/tutorials/azure-get-started/azure-build)
* Install Google Cloud gcloud CLI (for Mac use brew): [Link](https://cloud.google.com/sdk/docs/install)
* Google Cloud project: [Link](https://console.cloud.google.com/projectcreate)
* Enable the Google Compute Engine API for your project: [Link](https://console.developers.google.com/apis/library/compute.googleapis.com)

### Deployment Instructions

* Download or Clone this Terraform example.
* Modify the provider.tf file with your own Megaport API Key/API Key Secret, AWS Access Key/Secret Key, and Google Cloud project name and region.
* Modify the variables.tf file with your own AWS Account ID.
* The variables.tf file is configured for deployment in Sydney, it can be adjusted for other locations and regions as required.
* Authenticate to your Azure tenant using the Azure CLI: [Link](https://developer.hashicorp.com/terraform/tutorials/azure-get-started/azure-build#authenticate-using-the-azure-cli)
* Save the Azure tenant credentials locally as environment variables for use by Terraform: [Link](https://developer.hashicorp.com/terraform/tutorials/azure-get-started/azure-build#set-your-environment-variables)
* Authenticate to Google Cloud: [Link](https://developer.hashicorp.com/terraform/tutorials/gcp-get-started/google-cloud-platform-build#authenticate-to-google-cloud)
* From the command line change to the directory containing the Terraform files.
* Run `terraform init` to initialise Terraform and the providers.
* Run `terraform apply` to deploy this example.
* Gatus test agent stats are available using the public IP address allocted to the VM - http://vm_public_ip_address

### Notes

* This example will not work on the Megaport Staging environment. This is because it requires
real connections to AWS, Azure, and Google Cloud.
* It helps if you're not logged into mutliple Microsoft accounts while running this.
* The Azure ExpressRoute Gateway can take up to 20 minutes to deploy.
* When you have completed, use `terraform destroy` so that you will stop incurring costs for the resources.
* When destroying the deployment Terraform will report an error stating Azure ExpressRoute Private Peering still exists and the Azure VXC cannot be removed. Log into the Azure portal and manually deplete the ExpressRoute Circuit private peering and re-run Terraform destory.
* When destroying the deployment Terraform will report an error stating the AWS Direct Connect Connection still exists. Log into the AWS console and delete the AWS Direct Connect connection manually, then re-run `terraform destroy`.
