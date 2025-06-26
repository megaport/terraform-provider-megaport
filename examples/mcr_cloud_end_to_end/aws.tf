resource "aws_vpc" "aws_vpc_1" {
  cidr_block = var.aws_vpc_1_cidr
  
  tags = {
    Name = var.aws_vpc_1_name
  }
}

resource "aws_subnet" "aws_subnet_1" {
  vpc_id                  = aws_vpc.aws_vpc_1.id
  cidr_block              = var.aws_subnet_1
  map_public_ip_on_launch = false

  tags = {
    Name = var.aws_subnet_1_name
  }
}

resource "aws_route_table" "aws_route_table_1" {
  vpc_id = aws_vpc.aws_vpc_1.id

  tags = {
    Name = var.aws_route_table_1_name
  }
}

resource "aws_route_table_association" "aws_vpc_1_route_table_association" {
  subnet_id      = aws_subnet.aws_subnet_1.id
  route_table_id = aws_route_table.aws_route_table_1.id
}

resource "aws_main_route_table_association" "aws_route_table_1" {
  vpc_id         = aws_vpc.aws_vpc_1.id
  route_table_id = aws_route_table.aws_route_table_1.id
}

resource "aws_internet_gateway" "aws_internet_gateway_1" {
  vpc_id = aws_vpc.aws_vpc_1.id

  tags = {
    name  = var.aws_internet_gateway_1_name
  }
}

resource "aws_route" "aws_vpc_1_default_route" {
  route_table_id         = aws_route_table.aws_route_table_1.id
  destination_cidr_block = "0.0.0.0/0"
  gateway_id             = aws_internet_gateway.aws_internet_gateway_1.id
}

resource "aws_vpn_gateway" "aws_vpn_gateway_1" {
  vpc_id = aws_vpc.aws_vpc_1.id

  tags = {
    Name = var.aws_vpn_gateway_1_name
  }
}

resource "aws_vpn_gateway_route_propagation" "propagation_1" {
  vpn_gateway_id = aws_vpn_gateway.aws_vpn_gateway_1.id
  route_table_id = aws_route_table.aws_route_table_1.id
}

resource "aws_dx_gateway" "aws_dx_gateway_1" {
  name            = var.aws_dx_gateway_1_name
  amazon_side_asn = var.aws_dx_gateway_1_asn
}

resource "aws_dx_gateway_association" "dx_gateway_association" {
  dx_gateway_id         = aws_dx_gateway.aws_dx_gateway_1.id
  associated_gateway_id = aws_vpn_gateway.aws_vpn_gateway_1.id
}

resource "aws_dx_connection_confirmation" "confirmation" {
  connection_id = megaport_vxc.aws_vxc_1.csp_connections[1].connection_id
}

resource "aws_dx_private_virtual_interface" "aws_dx_private_virtual_interface_1" {
  connection_id    = megaport_vxc.aws_vxc_1.csp_connections[1].connection_id
  dx_gateway_id    = aws_dx_gateway.aws_dx_gateway_1.id
  name             = var.aws_dx_vif_1_name
  vlan             = megaport_vxc.aws_vxc_1.b_end.vlan
  address_family   = "ipv4"
  bgp_asn          = var.megaport_mcr_1_asn
  customer_address = var.aws_dx_vif_customer_address
  amazon_address   = var.aws_dx_vif_amazon_address
  bgp_auth_key     = var.megaport_aws_vxc_1_bgp_password

  depends_on = [
    aws_dx_connection_confirmation.confirmation
  ]
}

// Gatus VM Instance

resource "aws_security_group" "aws_instance_1_sg" {
  name        = "${var.aws_instance_1_name}-sg"
  description = "Instance security group"
  vpc_id      = aws_vpc.aws_vpc_1.id
}

resource "aws_security_group_rule" "aws_instance_1_rfc1918" {
  type              = "ingress"
  description       = "Allow all inbound from rfc1918"
  from_port         = -1
  to_port           = -1
  protocol          = -1
  cidr_blocks       = ["10.0.0.0/8", "192.168.0.0/16", "172.16.0.0/12"]
  security_group_id = aws_security_group.aws_instance_1_sg.id
}

resource "aws_security_group_rule" "gatus_inbound_tcp" {
  for_each          = var.inbound_tcp
  type              = "ingress"
  description       = "Allow inbound access from cidrs"
  from_port         = strcontains(each.key, "-") ? split("-", each.key)[0] : each.key
  to_port           = strcontains(each.key, "-") ? split("-", each.key)[1] : each.key
  protocol          = each.key == "0" ? "-1" : "tcp"
  cidr_blocks       = each.value
  security_group_id = aws_security_group.aws_instance_1_sg.id
}

resource "aws_security_group_rule" "gatus_egress" {
  type              = "egress"
  description       = "Allow all outbound"
  from_port         = 0
  to_port           = 0
  protocol          = "-1"
  cidr_blocks       = ["0.0.0.0/0"]
  security_group_id = aws_security_group.aws_instance_1_sg.id
}

data "aws_ami" "ubuntu" {
  most_recent = true
  filter {
    name   = "name"
    values = ["ubuntu/images/hvm-ssd/ubuntu-jammy-22.04-amd64-server-*"]
  }
  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }
  owners = ["099720109477"] # Canonical
}

resource "aws_instance" "gatus_instance" {
  ami                         = data.aws_ami.ubuntu.id
  instance_type               = var.instance_sizes["aws"]
  key_name                    = "<aws ssh key name>"
  ebs_optimized               = false
  source_dest_check           = false
  monitoring                  = true
  subnet_id                   = aws_subnet.aws_subnet_1.id
  associate_public_ip_address = true
  vpc_security_group_ids      = [aws_security_group.aws_instance_1_sg.id]
  private_ip                  = var.gatus_private_ips["aws"]

  user_data = templatefile("${path.module}/templates/gatus.tpl",
    {
      name     = var.aws_instance_1_name
      cloud    = "AWS"
      interval = var.gatus_interval
      inter    = "${var.gatus_private_ips["azure"]},${var.gatus_private_ips["google"]}"
      password = var.instance_password
  })

  root_block_device {
    volume_type = "gp2"
    volume_size = 8
  }
  tags = {
    name = var.aws_instance_1_name
  }
}
