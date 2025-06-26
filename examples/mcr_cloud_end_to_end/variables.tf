// Megaport variables

variable "megaport_location_1" {
  description = "Megaport Data Centre location 1"
  default     = "Equinix SY1"
}

variable "megaport_location_2" {
  description = "Megaport Data Centre location 2"
  default     = "Equinix SY3"
}

variable "megaport_mcr_1_name" {
  description = "Megaport MCR name"
  default     = "MCR SYD 1"
}

variable "megaport_mcr_1_term" {
  description = "Megaport MCR contract term"
  default     = 1
}

variable "megaport_mcr_1_speed" {
  description = "Megaport MCR speed"
  default     = 1000
}

variable "megaport_mcr_1_asn" {
  description = "Megaport MCR BGP ASN"
  default     = 133937
}

variable "megaport_mcr_1_diversity_zone" {
  description = "MCR Diversity Zone"
  default     = "red"
}

// Megaport - AWS variables

variable "megaport_aws_port_location_1_name" {
  description = "AWS Direct Connect Hosted Connection port name"
  default     = "Asia Pacific (Sydney) (ap-southeast-2)"
}

variable "megaport_aws_port_location_1_diversity_zone" {
  description = "Megaport AWS Direct Connect Port Location 1 Diversity Zone"
  default     = "red"
}

variable "megaport_aws_vxc_1_name" {
  description = "Megaport AWS VXC name"
  default     = "AWS Hosted Connection VXC - Primary"
}

variable "megaport_aws_vxc_1_bandwidth" {
  description = "Megaport AWS VXC bandwidth"
  default     = 50
}

variable "megaport_aws_vxc_1_term" {
  description = "Megaport AWS VXC term"
  default     = 1
}

variable "megaport_aws_vxc_1_local_address" {
  description = "Megaport AWS VXC local address"
  default     = "192.168.60.1"
}

variable "megaport_aws_vxc_1_remote_address" {
  description = "Megaport AWS VXC local address"
  default     = "192.168.60.2"
}

variable "megaport_aws_vxc_1_bgp_password" {
  description = "Megaport AWS VXC BGP password"
  default     = "<password>"
}

// Megaport - Azure variables

variable "megaport_expressroute_vxc_1_name" {
  description = "Megaport Azure ExpressRoute VXC name"
  default     = "Azure VXC - Primary"
}

variable "megaport_expressroute_bandwidth" {
  description = "Megaport Azure ExpressRoute VXC bandwidth"
  default     = 50
}

variable "megaport_expressroute_vxc_1_term" {
  description = "Megaport Azure VXC term"
  default     = 1
}

// Megaport - Google variables

variable "megaport_google_port_location_2_name" {
  description = "Google Cloud Partner Interconnect port name"
  default     = "Sydney (syd-zone1-1605)"
}

variable "megaport_google_vxc_1_name" {
  description = "Google Cloud VXC name"
  default     = "Google Cloud VXC - Primary"
}

variable "megaport_google_vxc_1_bandwidth" {
  description = "Megaport Google Cloud VXC bandwidth"
  default     = 50
}

variable "megaport_google_vxc_1_term" {
  description = "Megaport Google VXC term"
  default     = 1
}

// AWS variables

variable "aws_account_id" {
  description = "AWS Account ID"
  default     = "<aws account number>"
}

variable "aws_region_1" {
  description = "AWS region"
  default     = "ap-southeast-2"
}

variable "aws_vpc_1_name" {
  description = "AWS VPC name"
  default     = "VPC-SYD-1"
}

variable "aws_vpc_1_cidr" {
  description = "AWS VPC CIDR block"
  default     = "10.0.0.0/16"
}

variable "aws_subnet_1" {
  description = "AWS VPC subnet"
  default     = "10.0.1.0/24"
}

variable "aws_subnet_1_name" {
  description = "AWS VPC subnet name"
  default     = "VPC-SYD-1-subnet"
}

variable "aws_route_table_1_name" {
  description = "AWS VPC Route Table name"
  default     = "VPC-SYD-1-route-table"
}

variable "aws_internet_gateway_1_name" {
  description = "AWS Internet Gateway name"
  default     = "IGW-VPC-SYD-1"
}

variable "aws_vpn_gateway_1_name" {
  description = "AWS VPN Gateway name"
  default     = "VGW-VPC-SYD-1"
}

variable "aws_dx_gateway_1_name" {
  description = "AWS Direct Connect Gateway name"
  default     = "DGW-1"
}

variable "aws_dx_gateway_1_asn" {
  description = "AWS Direct Connect Gateway BGP ASN"
  default     = "64512"
}

variable "aws_dx_vif_1_name" {
  description = "AWS Direct Connect VIF name"
  default     = "DGW-1-Private_VIF-1"
}

variable "aws_dx_vif_customer_address" {
  description = "AWS Direct Connect VIF customer address"
  default     = "192.168.60.1/30"
}

variable "aws_dx_vif_amazon_address" {
  description = "AWS Direct Connect VIF Amazon address"
  default     = "192.168.60.2/30"
}

variable "aws_instance_1_name" {
  default = "aws-gatus-instance"
}

variable "inbound_tcp" {
  type        = map(list(string))
  description = "Inbound tcp ports for gatus instances"
  default = {
    80 = ["0.0.0.0/0"]
    22 = ["0.0.0.0/0"]
  }
}

// Azure variables

variable "azure_resource_group_name_1" {
  description = "Azure resource group name"
  default     = "resource-group-syd-1"
}

variable "azure_region_1" {
  description = "Azure region"
  default     = "Australia East"
}

variable "azure_virtual_network_name_1" {
  description = "Azure Virtual Network name"
  default     = "vnet-syd-1"
}

variable "azure_virtual_network_cidr_1" {
  description = "Azure Virtual Network CIDR"
  default     = ["10.32.0.0/16"]
}

variable "azure_virtual_network_subnet_name_1" {
  description = "Azure Virtual Network subnet name"
  default     = "vnet-subnet-syd-1"
}

variable "azure_virtual_network_subnet_1" {
  description = "The Azure Virtual Network subnet"
  default     = ["10.32.1.0/24"]
}

variable "azure_virtual_network_gateway_subnet_1" {
  description = "Azure Virtual Network Gateway Subnet"
  default     = ["10.32.255.0/24"]
}

variable "azure_expressroute_name_1" {
  description = "Azure ExpressRoute name"
  default     = "expressroute-syd-1"
}

variable "azure_expressroute_peering_location_1" {
  description = "Azure ExpressRoute location"
  default     = "Sydney"
}

variable "azure_expressroute_bandwidth_1" {
  description = "ExpressRoute Circuit bandwidth"
  default     = 50
}

variable "azure_expressroute_tier" {
  description = "Azure ExpressRoute Tier - Local/Standard/Premium"
  default     = "Standard"
}

variable "azure_expressroute_family" {
  description = "Azure ExpressRoute Family - MeteredData/Unlimited"
  default     = "MeteredData"
}

variable "azure_express_route_circuit_vlan_1" {
  description = "Azure ExpressRoute VLAN"
  default     = 100
}

variable "azure_express_route_circuit_primary_subnet_1" {
  description = "Azure ExpressRoute primary subnet"
  default     = "192.168.100.0/30"
}

variable "azure_express_route_circuit_secondary_subnet_1" {
  description = "Azure ExpressRoute secondary subnet"
  default     = "192.168.101.0/30"
}

variable "megaport_azure_bgp_password" {
  description = "Azure ExpresRoute BGP Password"
  default     = "password"
}

variable "azure_er_gateway_1_public_ip_name" {
  description = "Azure Virtual Network Gateway Public IP"
  default     = "er-gw-vnet-syd-1-public-ip"
}

variable "azure_er_gateway_1_name" {
  description = "The name of the Azure Virtual Network Gateway"
  default     = "er-gw-vnet-syd-1"
}

variable "azure_er_gateway_1_sku" {
  description = "Azure Virtual Network Gateway SKU"
  default     = "Standard"
}

variable "azure_virtual_network_gateway_1_connection_name" {
  description = "Azure Virtual Network Gateway Connection name"
  default     = "er-gw-vnet-syd-1-connection"
}

variable "azure_instance_1_name" {
  default = "azure-gatus-instance"
}

// Google Cloud Variables

variable "google_region_1_name" {
  description = "Google Cloud region name."
  default     = "australia-southeast1"
}

variable "google_vpc_1_name" {
  description = "Google Cloud VPC name"
  default     = "vpc-syd-1"
}

variable "google_subnet_1_name" {
  description = "Google Cloud VPC subnet name"
  default     = "vpc-syd-1-subnet-1"
}

variable "google_vpc_1_subnet_1" {
  description = "Google Cloud VPC subnet"
  default     = "10.64.1.0/24"
}

variable "google_cloud_router_1_name" {
  description = "Google Cloud router name."
  default     = "cloud-router-syd-1"
}

variable "google_interconnect_attachment_1_name" {
  description = "Google Cloud Interconnect Attachment name"
  default     = "attachment-syd-1"
}

variable "google_instance_1_name" {
  default = "google-gatus-instance"
}

variable "google_instance_1_zone" {
  default = "australia-southeast1-b"
}

variable "ssh_user" {
  default = "ubuntu"
}

variable "public_key_path" {
  default = "<public ssh key location>"
}

// Generic

variable "instance_password" {
  default = "<instance password>"
}

variable "instance_sizes" {
  type        = map(string)
  description = "Instance sizes for each cloud provider"
  default = {
    aws   = "t3.micro"
    azure = "Standard_B1ms"
  }
}

variable "gatus_interval" {
  type        = string
  description = "Interval for gatus polling (in seconds)"
  default     = "5"
}

variable "gatus_private_ips" {
  type        = map(string)
  description = "Private ips for the gatus instances"
  default = {
    aws    = "10.0.1.40"
    azure  = "10.32.1.40"
    google = "10.64.1.40"
  }
}
