# Resource Megaport GCP Connection
Adds a connection to Google Compute Cloud on a Port or an MCR. 

To provision this connection, you need a partner key from the Google Cloud Console. You can get this key by adding a new VLAN attachment and selecting "Megaport" as the interconnect partner.

## Example Usage (Automatically Select Google Port Location)
```
data "megaport_location" "syd_gs" {
  name = "Global Switch Sydney West"
}

resource "megaport_port" "port" {
  port_name   = "Terraform Example - Port"
  port_speed  = 1000
  location_id = data.megaport_location.syd_gs.id
}

resource "megaport_gcp_connection" "gcp_vxc" {
  vxc_name   = "Terraform Example - GCP VXC"
  rate_limit = 1000

  a_end {
    port_id        = megaport_port.port.id
    requested_vlan = 191
  }

  csp_settings {
    pairing_key = "19f9d93e-05c8-4c18-81fc-095d679ff645/australia-southeast-1/1"
  }
}
```

## Example Usage (Specify Google Port Location)
```
data "megaport_location" "syd_gs" {
  name = "Global Switch Sydney West"
}

data "megaport_location" "syd_nxt1" {
  name = "NextDC S1"
}

data "megaport_partner_port" "gcp_port" {
  connect_type = "GOOGLE"
  company_name = "Google Inc"
  product_name = "Sydney (syd-zone1-1660)"
  location_id  = data.megaport_location.syd_nxt1.id
}

resource "megaport_port" "port" {
  port_name   = "Terraform Example - Port"
  port_speed  = 1000
  location_id = data.megaport_location.syd_gs.id
}

resource "megaport_gcp_connection" "gcp_vxc" {
  vxc_name   = "Terraform Example - GCP VXC"
  rate_limit = 1000

  a_end {
    port_id        = megaport_port.port.id
    requested_vlan = 191
  }

  csp_settings {
    pairing_key          = "19f9d93e-05c8-4c18-81fc-095d679ff645/australia-southeast-1/1"
    requested_product_id = data.megaport_partner_port.gcp_port.id
  }
}
```

## Argument Reference
- `vxc_name` - (Required) The name of the VXC.
- `rate_limit` - (Required) The speed of the VXC in Mbps.
- `a_end` - (Required) ** See VXC Documentation
- `a_end_mcr_configuration` - (Optional) ** See VXC Documentation
- `csp_settings`:
    - `pairing_key` - (Required) The pairing key for the new GCP connection.
    - `requested_product_id` - (Optional) The partner port location you want to connect to.

## Attribute Reference
- `uid` - The identifier of the Port.
- `vxc_type` - The VXC type.
- `provisioning_status` - The current provisioning status of the VXC (this status does not refer to availability).
- `create_date` - A Unix timestamp representing the time the VXC was created.
- `created_by` - The user who created the VXC.
- `live_date` - A Unix timestamp representing the date the VXC went live.
- `company_name` - The name of the company that owns the account for the VXC.
- `locked` - Indicates whether the resource has been locked by a user.
- `admin_locked` - Indicates whether the resource has been locked by an admin.
- `vxc_internal_type` - An internal variable used by Terraform to orchestrate CSP VXCs.
- `a_end`:
    - `port_id` - The resource id of the Port (A-End) for the GCP connection.
    - `owner_uid` - The identifier of the owner of the A-End Port.
    - `name` - The name of the A-End Port.
    - `location` - The location name for the A-End Port.
    - `assigned_vlan` - The VLAN assigned by Megaport to the A-End Port.
- `b_end`:
    - `port_id` - The resource id of the GCP connection (B-End).
    - `owner_uid` - The identifier of the owner of the B-End port.
    - `name` - The name of the B-End port.
    - `location` - The location name for the B-End port.
    - `assigned_vlan` - The VLAN assigned by Megaport to the B-End port.
