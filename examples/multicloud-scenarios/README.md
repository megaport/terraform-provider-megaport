This folder contains 13 Megaport common multicloud connectivity scenarios, courtesy of Solutions Architect Mark Austen. These scenarios are documented in the Megaport docs here: [Link](https://docs.megaport.com/deployment/multicloud/)

The 13 common connectivity scenarios use Megaport Ports, Megaport Cloud Routers (MCR), and the Megaport Virtual Edge (MVE). The scenarios range from a simple partially-protected network to an end-to-end highly available resilient network for multicloud and hybrid cloud.

Summary of connectivity scenarios:

Scenario 1: One port connecting to AWS, Azure, Google Cloud, Oracle Cloud.

Scenario 2: Two ports in LAG connecting to AWS, Azure, Google Cloud, Oracle Cloud.

Scenario 3: One MCR connecting to a port and to AWS, Azure, Google Cloud, Oracle Cloud.

Scenario 4: Two ports in each diversity zone (one location) connecting to a port and to AWS, Azure, Google Cloud, Oracle Cloud.

Scenario 5: Two MCRs connecting to AWS, Azure, Google Cloud, Oracle Cloud.

Scenario 6: Two MCRs connecting to two ports (one location) and to AWS, Azure, Google Cloud, Oracle Cloud.

Scenario 7: Two ports (two locations) connecting to AWS, Azure, Google Cloud, Oracle Cloud.

Scenario 8: Two MCRs connecting to two ports (two locations) and to AWS, Azure, Google Cloud, Oracle Cloud.

Scenario 9: Two MCRs connecting to four ports (two locations) and to AWS, Azure, Google Cloud, Oracle Cloud.

Scenario 10: One MVE connecting to AWS, Azure, Google Cloud, Oracle Cloud, and Internet enabled branch locations.

Scenario 11: Two MVEs connecting to AWS, Azure, Google Cloud, Oracle Cloud, and Internet enabled branch locations.

Scenario 12: One MVE connecting to a data centre port, AWS, Azure, Google Cloud, Oracle Cloud, and Internet enabled branch locations.

Scenario 13: Two MVEs connecting to a data centre port, AWS, Azure, Google Cloud, Oracle Cloud, and Internet enabled branch locations.

### Prerequisites

* Install Terraform locally on Mac, Linux, or Windows: [Link](https://developer.hashicorp.com/terraform/tutorials/azure-get-started/install-cli)
* Create Megaport API Key: [Link](https://docs.megaport.com/api/api-key/)

### Deployment Instructions

* Download or Clone this Terraform example.
* Modify the provider.tf file with your own Megaport API Key/API Key Secret.
* From the command line change to the directory containing the Terraform files.
* Run `terraform init` to initialise Terraform and the providers.
* Run `terraform apply` to deploy this example.