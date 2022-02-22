# Resource Megaport Azure Connection
Connects a Port or an MCR to an Azure ExpressRoute Circuit. Supported peering configurations include 
Private and Microsoft peerings.

## Example Usage
```
data "megaport_location" "syd_gs" {
  name = "Global Switch Sydney West"
}

resource "megaport_port" "port" {
  port_name   = "Terraform Example - Port"
  port_speed  = 1000
  location_id = data.megaport_location.syd_gs.id
}

resource "megaport_azure_connection" "azure_vxc" {
  vxc_name   = "Terraform Example - Azure VXC"
  rate_limit = 1000

  a_end {
    port_id        = megaport_port.port.id
    requested_vlan = 191
  }

  csp_settings {
    service_key = "1b2329a5-56dc-45d0-8a0d-87b706297777"

    peerings {
      private_peer   = true
      microsoft_peer = true
    }
  }
}
```

## Argument Reference
- `vxc_name` - (Required) The name of the VXC.
- `rate_limit` - (Required) The speed of the VXC in Mbps.
- `a_end` - (Required) ** See VXC Documentation
- `a_end_mcr_configuration` - (Optional) ** See VXC Documentation
- `csp_settings`:
    - `service_key` - (Required) The service key for the new ExpressRoute generated from your Azure subscription.
    - `peerings`:
        - `private_peer` - (Optional, default false) enable private peering between your Megaport Resources and internal Azure
        network.
        - `microsoft_peer` - (Optional, default false) enable peering between Megaport Resources and the Microsoft Cloud
        (Office 365, Dynamics, etc).

## Attribute Reference
- `uid` - The Port identifier.
- `vxc_type` - The type of VXC.
- `provisioning_status` - The current provisioning status of the VXC (the status does not refer to availability).
- `create_date` - A Unix timestamp representing the time the VXC was created.
- `created_by` - The user who created the VXC.
- `live_date` - A Unix timestamp representing the date the VXC went live.
- `company_name` - The name of the company that owns the account for the VXC.
- `locked` - Indicates whether the resource has been locked by a user.
- `admin_locked` - Indicates whether the resource has been locked by an admin.
- `vxc_internal_type` - An internal variable used by Terraform to orchestrate CSP VXCs.
- `a_end`:
    - `port_id` - The resource id of the Port (A-End) for the Azure ExpressRoute connection.
    - `owner_uid` - The identifier for the owner of the A-End Port.
    - `name` - The name of the A-End Port.
    - `location` - The location name for the A-End Port.
    - `assigned_vlan` - The VLAN that was assigned by Megaport to the A-End Port.
- `b_end`:
    - `port_id` - The resource id of the Azure ExpressRoute connection (B-End).
    - `owner_uid` - The identifier for the owner of the B-End port.
    - `name` - The name of the B-End port.
    - `location` - The location name for the B-End port.
    - `assigned_vlan` - The VLAN that was assigned by Megaport to the B-End port.
