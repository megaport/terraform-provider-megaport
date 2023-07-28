# Resource Megaport OCI Connection
Connects an MCR to OCI using a Megaport VXC (Virtual Cross Connect).

## Example Usage
```
data "megaport_location" "syd_sy1" {
  name    = "Equinix SY1"
  has_mcr = true
}

data "megaport_partner_port" "primary_oci_port" {
  product_name   = "OCI (ap-sydney-1) (BMC)"
  diversity_zone = "blue"
  location_id    = data.megaport_location.syd_sy1.id
}

resource "megaport_mcr" "mcr" {
  mcr_name    = "Terraform Example - MCR"
  location_id = data.megaport_location.syd_sy1.id

  router {
    port_speed = 2500
  }
}

resource "megaport_oci_connection" "oci_vxc" {
  vxc_name   = "Terraform Example - OCI VXC"
  rate_limit = 1000

  a_end {
    port_id = megaport_mcr.mcr.id
  }

  a_end_mcr_configuration {
    ip_addresses = [var.customer_bgp_peering_ip]

    bgp_connection {
      peer_asn         = 31898
      local_ip_address = var.customer_bgp_peering_ip
      peer_ip_address  = var.oracle_bgp_peering_ip
    }
  }

  csp_settings {
    virtual_circut_id    = oci_core_virtual_circuit.generated_oci_core_virtual_circuit.id
    requested_product_id = data.megaport_partner_port.primary_oci_port.id
  }
}
```

## Argument Reference
- `vxc_name` - (Required) The name of the VXC.
- `rate_limit` - (Required) The speed of the VXC in Mbps.
- `a_end` - (Required) ** See VXC Documentation
- `a_end_mcr_configuration` - (Optional) ** See VXC Documentation
- `csp_settings`:
    - `virtual_circuit_id` - (Required) The virtual circuit id for the new OCI connection.
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
    - `port_id` - The resource id of the Port (A-End) for the OCI connection.
    - `owner_uid` - The identifier of the owner of the A-End Port.
    - `name` - The name of the A-End Port.
    - `location` - The location name for the A-End Port.
    - `assigned_vlan` - The VLAN assigned by Megaport to the A-End Port.
- `b_end`:
    - `port_id` - The resource id of the OCI connection (B-End).
    - `owner_uid` - The identifier of the owner of the B-End port.
    - `name` - The name of the B-End port.
    - `location` - The location name for the B-End port.
    - `assigned_vlan` - The VLAN assigned by Megaport to the B-End port.
