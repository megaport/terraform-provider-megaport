terraform {
  required_providers {
    megaport = {
      source  = "megaport/megaport"
      version = "1.2.4"
    }
  }
}

provider "megaport" {
  access_key            = "<api access_key>"
  secret_key            = "<api secret_key>"
  accept_purchase_terms = true
  environment           = "production"
}

data "megaport_location" "location_1" {
  id = 36 # Equinix Singapore SG1
}

data "megaport_location" "location_2" {
  id = 37 # Equinix Singapore SG2
}

resource "megaport_port" "port_1_sin" {
  product_name           = "Port 1 SIN"
  port_speed             = 10000
  location_id            = data.megaport_location.location_1.id
  contract_term_months   = 1
  marketplace_visibility = false
  diversity_zone         = "red"
}

data "megaport_partner" "aws_port_1_sin" {
  connect_type   = "AWSHC"
  company_name   = "AWS"
  product_name   = "Asia Pacific (Singapore) (ap-southeast-1)"
  location_id    = data.megaport_location.location_2.id
  diversity_zone = "red"
}

resource "megaport_vxc" "aws_vxc_1_sin" {
  product_name         = "AWS VXC - Primary"
  rate_limit           = 50
  contract_term_months = 1

  a_end = {
    requested_product_uid = megaport_port.port_1_sin.product_uid
    ordered_vlan          = 301
  }

  b_end = {
    requested_product_uid = data.megaport_partner.aws_port_1_sin.product_uid
  }

  b_end_partner_config = {
    partner = "aws"
    aws_config = {
      name          = "AWS VXC - Primary"
      type          = "private"
      connect_type  = "AWSHC"
      owner_account = "<aws account id>"
    }
  }
}

resource "megaport_vxc" "azure_vxc_1_sin" {
  product_name         = "Azure VXC - Primary"
  rate_limit           = 50
  contract_term_months = 1

  a_end = {
    requested_product_uid = megaport_port.port_1_sin.product_uid
    ordered_vlan          = 401
  }

  b_end = {}

  b_end_partner_config = {
    partner = "azure"
    azure_config = {
      port_choice = "primary"
      service_key = "<azure expressroute service key>"
    }
  }
}

data "megaport_partner" "google_port_1_sin" {
  connect_type = "GOOGLE"
  company_name = "Google inc.."
  product_name = "Singapore (sin-zone1-2260)"
  location_id  = data.megaport_location.location_1.id
}

resource "megaport_vxc" "google_vxc_1_sin" {
  product_name         = "Google Cloud VXC - Primary"
  rate_limit           = 50
  contract_term_months = 1

  a_end = {
    requested_product_uid = megaport_port.port_1_sin.product_uid
    ordered_vlan          = 501
  }

  b_end = {
    requested_product_uid = data.megaport_partner.google_port_1_sin.product_uid
  }

  b_end_partner_config = {
    partner = "google"
    google_config = {
      pairing_key = "<google cloud partner interconnect pairing key>"
    }
  }
}

data "megaport_partner" "oracle_port_1_sin" {
  connect_type   = "ORACLE"
  company_name   = "Oracle"
  product_name   = "OCI (ap-singapore-1) (BMC)"
  location_id    = data.megaport_location.location_1.id
  diversity_zone = "red"
}

resource "megaport_vxc" "oracle_vxc_1_sin" {
  product_name         = "Oracle Cloud VXC - Primary"
  rate_limit           = 1000
  contract_term_months = 1

  a_end = {
    requested_product_uid = megaport_port.port_1_sin.product_uid
    ordered_vlan          = 601
  }

  b_end = {
    requested_product_uid = data.megaport_partner.oracle_port_1_sin.product_uid
  }

  b_end_partner_config = {
    partner = "oracle"
    oracle_config = {
      virtual_circuit_id = "<oracle cloud fastconnect virtual circuit id>"
    }
  }
}