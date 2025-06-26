resource "azurerm_resource_group" "resource_group_1" {
  name     = var.azure_resource_group_name_1
  location = var.azure_region_1
}

resource "azurerm_virtual_network" "virtual_network_1" {
  name                = var.azure_virtual_network_name_1
  location            = azurerm_resource_group.resource_group_1.location
  resource_group_name = azurerm_resource_group.resource_group_1.name
  address_space       = var.azure_virtual_network_cidr_1
}

resource "azurerm_subnet" "virtual_network_subnet_1" {
  name                 = var.azure_virtual_network_subnet_name_1
  resource_group_name  = azurerm_resource_group.resource_group_1.name
  virtual_network_name = azurerm_virtual_network.virtual_network_1.name
  address_prefixes     = var.azure_virtual_network_subnet_1
}

resource "azurerm_subnet" "gateway_subnet_1" {
  name                 = "GatewaySubnet"
  resource_group_name  = azurerm_resource_group.resource_group_1.name
  virtual_network_name = azurerm_virtual_network.virtual_network_1.name
  address_prefixes     = var.azure_virtual_network_gateway_subnet_1
}

resource "azurerm_express_route_circuit" "express_route_circuit_1" {
  name                  = var.azure_expressroute_name_1
  resource_group_name   = azurerm_resource_group.resource_group_1.name
  location              = azurerm_resource_group.resource_group_1.location
  service_provider_name = "Megaport"
  peering_location      = var.azure_expressroute_peering_location_1
  bandwidth_in_mbps     = var.azure_expressroute_bandwidth_1

  sku {
    tier   = var.azure_expressroute_tier
    family = var.azure_expressroute_family
  }
}

resource "azurerm_public_ip" "er_gateway_1_public_ip" {
  name                = var.azure_er_gateway_1_public_ip_name
  location            = azurerm_resource_group.resource_group_1.location
  resource_group_name = azurerm_resource_group.resource_group_1.name
    
  sku               = "Standard"
  allocation_method = "Static"
}

resource "azurerm_virtual_network_gateway" "er_gateway_1" {
  name                = var.azure_er_gateway_1_name
  location            = azurerm_resource_group.resource_group_1.location
  resource_group_name = azurerm_resource_group.resource_group_1.name

  type = "ExpressRoute"

  active_active = false
  enable_bgp    = true
  sku           = var.azure_er_gateway_1_sku

  ip_configuration {
    name                          = "erGatewayConfig"
    public_ip_address_id          = azurerm_public_ip.er_gateway_1_public_ip.id
    private_ip_address_allocation = "Dynamic"
    subnet_id                     = azurerm_subnet.gateway_subnet_1.id
  }
}

resource "azurerm_virtual_network_gateway_connection" "virtual_network_gateway_1_connection" {
  name                = var.azure_virtual_network_gateway_1_connection_name
  location            = azurerm_resource_group.resource_group_1.location
  resource_group_name = azurerm_resource_group.resource_group_1.name

  type                       = "ExpressRoute"
  virtual_network_gateway_id = azurerm_virtual_network_gateway.er_gateway_1.id
  express_route_circuit_id   = azurerm_express_route_circuit.express_route_circuit_1.id
}

// Gatus VM Instance

data "azurerm_subscription" "current" {}

resource "azurerm_public_ip" "gatus_public_ip" {
  name                = "${var.azure_instance_1_name}-public-ip"
  location            = azurerm_resource_group.resource_group_1.location
  resource_group_name = azurerm_resource_group.resource_group_1.name
  allocation_method   = "Static"
  sku                 = "Standard"
}

resource "azurerm_network_interface" "gatus_network_interface" {
  name                = var.azure_instance_1_name
  location            = azurerm_resource_group.resource_group_1.location
  resource_group_name = azurerm_resource_group.resource_group_1.name
  ip_configuration {
    name                          = var.azure_instance_1_name
    subnet_id                     = azurerm_subnet.virtual_network_subnet_1.id
    private_ip_address_allocation = "Static"
    private_ip_address            = var.gatus_private_ips["azure"]
    public_ip_address_id          = azurerm_public_ip.gatus_public_ip.id
  }
}

resource "azurerm_linux_virtual_machine" "gatus_instance" {
  name                            = var.azure_instance_1_name
  location                        = azurerm_resource_group.resource_group_1.location
  resource_group_name             = azurerm_resource_group.resource_group_1.name
  network_interface_ids           = [azurerm_network_interface.gatus_network_interface.id]
  admin_username                  = "ubuntu"
  admin_password                  = var.instance_password
  computer_name                   = var.azure_instance_1_name
  size                            = var.instance_sizes["azure"]
  source_image_id                 = null
  disable_password_authentication = false

  admin_ssh_key {
    username   = "ubuntu"
    public_key = file("~/.ssh/<public ssh key>")
  }

  custom_data = base64encode(templatefile("${path.module}/templates/gatus.tpl",
    {
      name     = var.azure_instance_1_name
      cloud    = "Azure"
      interval = var.gatus_interval
      inter    = "${var.gatus_private_ips["aws"]},${var.gatus_private_ips["google"]}"
      password = var.instance_password
  }))

  dynamic "source_image_reference" {
    for_each = ["ubuntu"]

    content {
      publisher = "Canonical"
      offer     = "0001-com-ubuntu-server-jammy"
      sku       = "22_04-lts-gen2"
      version   = "latest"
    }
  }

  os_disk {
    caching              = "ReadWrite"
    storage_account_type = "Standard_LRS"
  }
}

resource "azurerm_network_security_group" "gatus_network_security_group" {
  name                = var.azure_instance_1_name
  resource_group_name = azurerm_resource_group.resource_group_1.name
  location            = azurerm_resource_group.resource_group_1.location
}

resource "azurerm_network_interface_security_group_association" "gatus_security_group_association" {
  network_interface_id      = azurerm_network_interface.gatus_network_interface.id
  network_security_group_id = azurerm_network_security_group.gatus_network_security_group.id
}

resource "azurerm_network_security_rule" "gatus_rfc_1918" {
  access                      = "Allow"
  direction                   = "Inbound"
  name                        = "rfc-1918"
  priority                    = 100
  protocol                    = "*"
  source_port_range           = "*"
  source_address_prefixes     = ["10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"]
  destination_port_range      = "*"
  destination_address_prefix  = "*"
  resource_group_name         = azurerm_resource_group.resource_group_1.name
  network_security_group_name = azurerm_network_security_group.gatus_network_security_group.name
}

resource "azurerm_network_security_rule" "gatus_inbound_tcp" {
  for_each                    = var.inbound_tcp
  access                      = "Allow"
  direction                   = "Inbound"
  name                        = "inbound_tcp_${each.key}"
  priority                    = (index(keys(var.inbound_tcp), each.key) + 101)
  protocol                    = "Tcp"
  source_port_range           = "*"
  source_address_prefixes     = each.value
  destination_port_range      = each.key
  destination_address_prefix  = "*"
  resource_group_name         = azurerm_resource_group.resource_group_1.name
  network_security_group_name = azurerm_network_security_group.gatus_network_security_group.name
}
