# ExpressRoute circuit
# Public IP Address -> Virtual network Gateway
# Virtual Network

resource "azurerm_resource_group" "resource_group" {
  name     = "${lower(var.prefix)}-resource-group"
  location = var.azure_region
}

resource "azurerm_express_route_circuit" "express_route_circuit" {
  name                  = "${lower(var.prefix)}-terraform-expressroute"
  resource_group_name   = azurerm_resource_group.resource_group.name
  location              = azurerm_resource_group.resource_group.location
  service_provider_name = "Megaport"
  peering_location      = "Sydney"
  bandwidth_in_mbps     = var.azure_expressroute_bandwidth

  sku {
    tier   = "Standard"
    family = "MeteredData"
  }

  tags = {
    environment = "Terraform Testing"
  }
}

resource "azurerm_virtual_network" "virtual_network" {
  name                = "${lower(var.prefix)}-terraform-virtual-network"
  location            = azurerm_resource_group.resource_group.location
  resource_group_name = azurerm_resource_group.resource_group.name
  address_space       = ["10.0.0.0/16"]

  tags = {
    environment = "Terraform Testing"
  }
}

resource "azurerm_subnet" "subnet_one" {
  name                 = "subnet_one"
  resource_group_name  = azurerm_resource_group.resource_group.name
  virtual_network_name = azurerm_virtual_network.virtual_network.name
  address_prefixes     = ["10.0.1.0/24"]
}

resource "azurerm_subnet" "subnet_two" {
  name                 = "subnet_two"
  resource_group_name  = azurerm_resource_group.resource_group.name
  virtual_network_name = azurerm_virtual_network.virtual_network.name
  address_prefixes     = ["10.0.2.0/24"]
}

resource "azurerm_subnet" "subnet_three" {
  name                 = "subnet_three"
  resource_group_name  = azurerm_resource_group.resource_group.name
  virtual_network_name = azurerm_virtual_network.virtual_network.name
  address_prefixes     = ["10.0.3.0/24"]
}

resource "azurerm_subnet" "subnet" {
  name                 = "GatewaySubnet"
  resource_group_name  = azurerm_resource_group.resource_group.name
  virtual_network_name = azurerm_virtual_network.virtual_network.name
  address_prefixes     = ["10.0.4.0/24"]
}

resource "azurerm_subnet" "bastion" {
  name                 = "AzureBastionSubnet"
  resource_group_name  = azurerm_resource_group.resource_group.name
  virtual_network_name = azurerm_virtual_network.virtual_network.name
  address_prefixes     = ["10.0.5.0/24"]
}

resource "azurerm_public_ip" "public_ip" {
  name                = "${lower(var.prefix)}-terraform-virtual-network-gateway-ip"
  location            = azurerm_resource_group.resource_group.location
  resource_group_name = azurerm_resource_group.resource_group.name

  allocation_method = "Dynamic"
}

resource "azurerm_virtual_network_gateway" "virtual_network_gateway" {
  name                = "${lower(var.prefix)}-terraform-virtual-network-gateway"
  location            = azurerm_resource_group.resource_group.location
  resource_group_name = azurerm_resource_group.resource_group.name

  type = "ExpressRoute"

  active_active = false
  enable_bgp    = true
  sku           = "Standard"

  ip_configuration {
    name                          = "vnetGatewayConfig"
    public_ip_address_id          = azurerm_public_ip.public_ip.id
    private_ip_address_allocation = "Dynamic"
    subnet_id                     = azurerm_subnet.subnet.id
  }
}

resource "azurerm_virtual_network_gateway_connection" "virtual_network_gateway_connection" {
  name                = "${lower(var.prefix)}-terraform-connection"
  location            = azurerm_resource_group.resource_group.location
  resource_group_name = azurerm_resource_group.resource_group.name

  type                       = "ExpressRoute"
  virtual_network_gateway_id = azurerm_virtual_network_gateway.virtual_network_gateway.id
  express_route_circuit_id   = azurerm_express_route_circuit.express_route_circuit.id

  enable_bgp = true
}

resource "azurerm_network_interface" "network_interface" {
  name                = "${lower(var.prefix)}-terraform-virtualmachine-nic"
  location            = azurerm_resource_group.resource_group.location
  resource_group_name = azurerm_resource_group.resource_group.name

  ip_configuration {
    name                          = "internal"
    subnet_id                     = azurerm_subnet.subnet_one.id
    private_ip_address_allocation = "Dynamic"
  }
}

resource "azurerm_linux_virtual_machine" "linux_virtual_machine" {
  name                = "${lower(var.prefix)}-terraform-virtualmachine"
  resource_group_name = azurerm_resource_group.resource_group.name
  location            = azurerm_resource_group.resource_group.location
  size                = "Standard_F2"
  admin_username      = "azureuser"
  network_interface_ids = [
    azurerm_network_interface.network_interface.id,
  ]

  admin_password                  = "TestPassword01@"
  disable_password_authentication = false

  os_disk {
    caching              = "ReadWrite"
    storage_account_type = "Standard_LRS"
  }

  source_image_reference {
    publisher = "Canonical"
    offer     = "UbuntuServer"
    sku       = "16.04-LTS"
    version   = "latest"
  }
}

resource "azurerm_public_ip" "public_ip" {
  name                = "${lower(var.prefix)}-terraform-public-ip-bastion"
  location            = azurerm_resource_group.resource_group.location
  resource_group_name = azurerm_resource_group.resource_group.name
  allocation_method   = "Static"
  sku                 = "Standard"
}

resource "azurerm_bastion_host" "bastion_host" {
  name                = "${lower(var.prefix)}-terraform-bastion"
  location            = azurerm_resource_group.resource_group.location
  resource_group_name = azurerm_resource_group.resource_group.name

  ip_configuration {
    name                 = "configuration"
    subnet_id            = azurerm_subnet.bastion.id
    public_ip_address_id = azurerm_public_ip.public_ip.id
  }
}
