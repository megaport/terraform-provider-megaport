/**
 * Copyright 2020 Megaport Pty Ltd
 *
 * Licensed under the Mozilla Public License, Version 2.0 (the
 * "License"); you may not use this file except in compliance with
 * the License. You may obtain a copy of the License at
 *
 *       https://mozilla.org/MPL/2.0/
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

# aws variables
variable "aws_region" {
  description = "The AWS region to create resources in."
  default     = "ap-southeast-2"
}

variable "aws_vpc_cidr" {
  description = "The CIDR block for the AWS VPC."
  default     = "172.16.1.0/24"
}

variable "aws_dx_gateway_asn" {
  description = "The ASN to be configured on the Amazon side of the connection."
  default     = "64512"
}

variable "aws_ec2_instance_type" {
  description = "The type of EC2 instance to be deployed into the AWS VPC."
  default     = "t3.micro"
}

# Azure Region
variable "azure_region" {
  description = "The region to create the resource group and resources in on Azure."
  default     = "Australia East"
}

variable "prefix" {
  description = "A prefix to add to all the environments."
  default     = "DemoEnv"
}

variable "aws_ec2_key_pair_name" {
  description = "The name of a keypair you have created in the AWS account."
  default     = "terraform-testing"
}

variable "azure_expressroute_bandwidth" {
  description = "Bandwidth required on the ExpressConnect circuit/connection."
  default     = 1000
}
