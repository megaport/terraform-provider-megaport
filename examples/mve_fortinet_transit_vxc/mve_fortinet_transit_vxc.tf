data "megaport_location" "location_1" {
  id = 2 // Equinix SY1
}
data "megaport_location" "location_2" {
  id = 10 // NextDC S1
}
data "megaport_location" "location_3" {
  id = 3 // Global Switch Sydney West 
}
data "megaport_location" "location_4" {
  id = 6 // Equinix SY3
}
data "megaport_mve_images" "fortinet" {
  vendor_filter  = "Fortinet"
  version_filter = "7.6.3"
}
resource "megaport_mve" "mve_location_1" {
  product_name         = "MVE 1"
  location_id          = data.megaport_location.location_1.id
  contract_term_months = 1
  diversity_zone       = "blue"
  vendor_config = {
    vendor         = "fortinet"
    product_size   = "SMALL"
    image_id       = data.megaport_mve_images.fortinet.mve_images.0.id
    ssh_public_key = "ssh-rsa <public key>"
    license_data   = "<base64 encode of fortigate .lic file>" // optional
  }
  vnics = [
    {
      description = "Port1"
    },
    {
      description = "Port2"
    },
    {
      description = "Port3"
    },
    {
      description = "Port4"
    },
    {
      description = "Port5"
    }
  ]
}
resource "megaport_mve" "mve_location_2" {
  product_name         = "MVE 2"
  location_id          = data.megaport_location.location_2.id
  contract_term_months = 1
  diversity_zone       = "red"
  vendor_config = {
    vendor       = "fortinet"
    product_size = "SMALL"
    image_id     = data.megaport_mve_images.fortinet.mve_images.0.id

    ssh_public_key = "ssh-rsa <public key>"

    license_data = "<base64 encode of fortigate .lic file>" // optional
  }
  vnics = [
    {
      description = "Port1"
    },
    {
      description = "Port2"
    },
    {
      description = "Port3"
    },
    {
      description = "Port4"
    },
    {
      description = "Port5"
    }
  ]
}
data "megaport_partner" "internet_zone_blue" {
  connect_type = "TRANSIT"
  company_name = "Networks"
  product_name = "Megaport Internet"
  location_id  = data.megaport_location.location_3.id
}
data "megaport_partner" "internet_zone_red" {
  connect_type = "TRANSIT"
  company_name = "Networks"
  product_name = "Megaport Internet"
  location_id  = data.megaport_location.location_2.id
}
resource "megaport_vxc" "transit_vxc_blue" {
  product_name         = data.megaport_location.location_1.name
  rate_limit           = 1000
  contract_term_months = 1

  a_end = {
    requested_product_uid = megaport_mve.mve_location_1.product_uid
    vnic_index            = 0
  }

  b_end = {
    requested_product_uid = data.megaport_partner.internet_zone_blue.product_uid
  }

  b_end_partner_config = {
    partner = "transit"
  }
}
resource "megaport_vxc" "transit_vxc_red" {
  product_name         = data.megaport_location.location_2.name
  rate_limit           = 1000
  contract_term_months = 1

  a_end = {
    requested_product_uid = megaport_mve.mve_location_2.product_uid
    vnic_index            = 0
  }

  b_end = {
    requested_product_uid = data.megaport_partner.internet_zone_red.product_uid
  }

  b_end_partner_config = {
    partner = "transit"
  }
}
data "megaport_partner" "aws_port_1_syd" {
  connect_type   = "AWSHC"
  company_name   = "AWS"
  product_name   = "Asia Pacific (Sydney) (ap-southeast-2)"
  location_id    = data.megaport_location.location_4.id
  diversity_zone = "blue"
}
data "megaport_partner" "aws_port_2_syd" {
  connect_type   = "AWSHC"
  company_name   = "AWS"
  product_name   = "Asia Pacific (Sydney) (ap-southeast-2)"
  location_id    = data.megaport_location.location_2.id
  diversity_zone = "red"
}
resource "megaport_vxc" "aws_vxc_syd_1" {
  product_name         = "AWS VXC 1"
  rate_limit           = 50
  contract_term_months = 1
  a_end = {
    requested_product_uid = megaport_mve.mve_location_1.product_uid
    vnic_index            = 1
    inner_vlan            = 100
  }
  b_end = {
    requested_product_uid = data.megaport_partner.aws_port_1_syd.product_uid
  }
  b_end_partner_config = {
    partner = "aws"
    aws_config = {
      name          = "AWS VXC 1"
      connect_type  = "AWSHC"
      owner_account = "<aws account id>"
    }
  }
}
resource "megaport_vxc" "aws_vxc_syd_2" {
  product_name         = "AWS VXC 2"
  rate_limit           = 50
  contract_term_months = 1
  a_end = {
    requested_product_uid = megaport_mve.mve_location_2.product_uid
    vnic_index            = 1
    inner_vlan            = 100
  }
  b_end = {
    requested_product_uid = data.megaport_partner.aws_port_2_syd.product_uid
  }
  b_end_partner_config = {
    partner = "aws"
    aws_config = {
      name          = "AWS VXC 2"
      connect_type  = "AWSHC"
      owner_account = "<aws account id>"
    }
  }
}
resource "megaport_vxc" "fgsp_vxc_1" {
  product_name         = "FGSP VXC 1"
  rate_limit           = 1000
  contract_term_months = 1

  a_end = {
    requested_product_uid = megaport_mve.mve_location_1.product_uid
    vnic_index            = 4
    inner_vlan            = 1000
  }

  b_end = {
    requested_product_uid = megaport_mve.mve_location_2.product_uid
    vnic_index            = 4
    inner_vlan            = 1000
  }
}