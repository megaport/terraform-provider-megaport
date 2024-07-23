---
page_title: "Moving VXC End Configurations"
description: |-
  How to move a VXC to a different end configuration
---

# Moving VXC End Configuration

This guide provides an example configuration for moving a VXC to a different end configuration.

- The new endpoint can be in a different location but must be in the same metro area.
- The new endpoint must be of the same type. For example, MEGAPORT to MEGAPORT, MCR to MCR, or MVE to MVE.
- The configured speed of each VXC must be no greater than the speed of the destination Port. If downgrading, it might be necessary for you to lower the speed of the VXC before requesting the move.
- There must not be an IP address or VLAN conflict. The updated services are checked as if a new service is being ordered. For example, an untagged VLAN can't be moved to a - service that already has an untagged VLAN.
- The services being moved must have different VLAN IDs from any services already on the destination.

## Initial Configuration

In this example, we create four Ports and a VXC connecting the first two Ports.
```terraform
provider "megaport" {
  environment           = "staging"
  access_key            = "access_key"
  secret_key            = "secret_Key"
  accept_purchase_terms = true
}

data "megaport_location" "loc" {
  name = "NextDC B1"
}

resource "megaport_port" "port_1" {
  product_name            = "Port 1"
  port_speed              = 1000
  location_id             = data.megaport_location.loc.id
  contract_term_months    = 12
  marketplace_visibility  = false
}

resource "megaport_port" "port_2" {
  product_name            = "Port 2"
  port_speed              = 1000
  location_id             = data.megaport_location.loc.id
  contract_term_months    = 12
  marketplace_visibility  = false
}

resource "megaport_port" "port_3" {
  product_name            = "Port 3"
  port_speed              = 1000
  location_id             = data.megaport_location.loc.id
  contract_term_months    = 12
  marketplace_visibility  = false
}

resource "megaport_port" "port_4" {
  product_name            = "Port 4"
  port_speed              = 1000
  location_id             = data.megaport_location.loc.id
  contract_term_months    = 12
  marketplace_visibility  = false
}

resource "megaport_vxc" "vxc" {
  product_name            = "Example VXC"
  rate_limit              = 500
  contract_term_months    = 12
  cost_centre             = "Example Cost Centre"

  a_end = {
    requested_product_uid = megaport_port.port_1.product_uid
  }

  b_end = {
    requested_product_uid = megaport_port.port_2.product_uid
  }
}

```

## Moving VXC End Configuration

In this example, we move the VXC to connect the third and fourth Ports by re-assigning the `product_uid` field of the A-End and B-End configuration of the VXC.

```terraform
provider "megaport" {
  environment           = "staging"
  access_key            = "access_key"
  secret_key            = "secret_Key"
  accept_purchase_terms = true
}

data "megaport_location" "loc" {
  name = "NextDC B1"
}

resource "megaport_port" "port_1" {
  product_name            = "Port 1"
  port_speed              = 1000
  location_id             = data.megaport_location.loc.id
  contract_term_months    = 12
  marketplace_visibility  = false
}

resource "megaport_port" "port_2" {
  product_name            = "Port 2"
  port_speed              = 1000
  location_id             = data.megaport_location.loc.id
  contract_term_months    = 12
  marketplace_visibility  = false
}

resource "megaport_port" "port_3" {
  product_name            = "Port 3"
  port_speed              = 1000
  location_id             = data.megaport_location.loc.id
  contract_term_months    = 12
  marketplace_visibility  = false
}

resource "megaport_port" "port_4" {
  product_name            = "Port 4"
  port_speed              = 1000
  location_id             = data.megaport_location.loc.id
  contract_term_months    = 12
  marketplace_visibility  = false
}

resource "megaport_vxc" "vxc" {
  product_name            = "Example VXC"
  rate_limit              = 500
  contract_term_months    = 12
  cost_centre             = "Example Cost Centre"

  a_end = {
    requested_product_uid = megaport_port.port_3.product_uid
  }

  b_end = {
    requested_product_uid = megaport_port.port_4.product_uid
  }
}
```

Once the VXCs are moved, the VXC will be connected to the third and fourth Ports.  The user can then delete the first and second Ports if they are no longer required.

## Example with AWS Virtual Interface and two MCRs.

In this example, we move a VXC connecting an AWS Virtual Interface and an MCR to connect to a different MCR.

First provision the resources for the two MCRs and the AWS Virtual Interface VXC.
```terraform
provider "megaport" {
  environment           = "staging"
  access_key            = "access_key"
  secret_key            = "secret_Key"
  accept_purchase_terms = true
}

data "megaport_location" "bne_nxt1" {
  name = "NextDC B1"
}

data "megaport_location" "syd_gs" {
  name = "Global Switch Sydney West"
}

data "megaport_partner" "aws_port" {
  connect_type = "AWS"
  company_name = "AWS"
  product_name = "Asia Pacific (Sydney) (ap-southeast-2)"
  location_id  = data.megaport_location.syd_gs.id
}

resource "megaport_mcr" "mcr" {
  product_name         = "Megaport Example MCR A-End"
  location_id          = data.megaport_location.bne_nxt1.id
  contract_term_months = 1
  port_speed           = 5000
  asn                  = 64555
  cost_centre          = "MCR Example"
}

resource "megaport_mcr" "mcr_2" {
  product_name         = "Megaport Example MCR A-End 2"
  location_id          = data.megaport_location.bne_nxt1.id
  contract_term_months = 1
  port_speed           = 10000
  asn                  = 64555
  cost_centre          = "MCR Example 2"
}

resource "megaport_vxc" "aws_vxc" {
  product_name         = "Megaport VXC Example - AWS"
  rate_limit           = 1000
  contract_term_months = 1

  a_end = {
    requested_product_uid = megaport.mcr.mcr.product_uid
    ordered_vlan          = 2191
  }

  b_end = {
    requested_product_uid = data.megaport_partner.aws_port.product_uid
  }

  b_end_partner_config = {
    partner = "aws"
    aws_config = {
      name          = "Megaport VXC Example - AWS"
      asn           = 64550
      type          = "private"
      connect_type  = "AWS"
      amazon_asn    = 64551
      owner_account = "123456789012"
    }
  }
}
```

Then, move the VXC to a different MCR.

```terraform
provider "megaport" {
  environment           = "staging"
  access_key            = "access_key"
  secret_key            = "secret_Key"
  accept_purchase_terms = true
}

data "megaport_location" "bne_nxt1" {
  name = "NextDC B1"
}

data "megaport_location" "syd_gs" {
  name = "Global Switch Sydney West"
}

data "megaport_partner" "aws_port" {
  connect_type = "AWS"
  company_name = "AWS"
  product_name = "Asia Pacific (Sydney) (ap-southeast-2)"
  location_id  = data.megaport_location.syd_gs.id
}

resource "megaport_mcr" "mcr" {
  product_name         = "Megaport Example MCR A-End"
  location_id          = data.megaport_location.bne_nxt1.id
  contract_term_months = 1
  port_speed           = 5000
  asn                  = 64555
  cost_centre          = "MCR Example"
}

resource "megaport_mcr" "mcr_2" {
  product_name         = "Megaport Example MCR A-End 2"
  location_id          = data.megaport_location.bne_nxt1.id
  contract_term_months = 1
  port_speed           = 10000
  asn                  = 64555
  cost_centre          = "MCR Example 2"
}

resource "megaport_vxc" "aws_vxc" {
  product_name         = "Megaport VXC Example - AWS"
  rate_limit           = 1000
  contract_term_months = 1

  a_end = {
    requested_product_uid = megaport.mcr.mcr_2.product_uid
    ordered_vlan          = 2191
  }

  b_end = {
    requested_product_uid = data.megaport_partner.aws_port.product_uid
  }

  b_end_partner_config = {
    partner = "aws"
    aws_config = {
      name          = "Megaport VXC Example - AWS"
      asn           = 64550
      type          = "private"
      connect_type  = "AWS"
      amazon_asn    = 64551
      owner_account = "123456789012"
    }
  }
}
```

Once the VXC is successfully moved to the second MCR, the user can delete the first MCR if it is no longer required.

## Additional Documentation on Moving VXCs

For additional documentation on moving VXCs, please visit [Moving a VXC](https://docs.megaport.com/connections/move-vxc/).