# Resource Megaport Azure Connection
Connects a Port or an MCR to an Azure ExpressRoute Circuit. Supported peering configurations include 
Private and Microsoft peerings.

## Example Usage
```
data megaport_location glb_switch_sydney {
  name = "Global Switch Sydney West"
}

resource megaport_port my_port {
    port_name       = "My Example Port"
    port_speed      = 1000
    location_id     = data.megaport_location.glb_switch_sydney.id
}

resource megaport_azure_connection test {
  vxc_name = "My Example ExpressRoute"
  rate_limit = 1000

  a_end {
    requested_vlan = 191
  }

  csp_settings {
    attached_to = megaport_port.my_port.id
    service_key = "1b2329a5-56dc-45d0-8a0d-87b706297777"

    peerings {
      private = true
      microsoft = true
    }
  }
}
```

## Argument Reference
- `vxc_name` - (Required) The name of the VXC.
- `rate_limit` - (Required) The speed of the VXC in Mbps.
- `a_end`
    - `requested_vlan` - (Required) The VLAN to assign to the A-End Port.
- `csp_settings`:
    - `attached_to` - (Required) The identifier of the product (Port/MCR) to attach the connection to.
    - `service_key` - (Required) The service key for the new ExpressRoute generated from your Azure subscription.
    - `peerings`:
        - `private`: (Optional, default false) enable private peering between your Megaport Resources and internal Azure
        network.
        - `microsoft`: (Optional, default false) enable peering between Megaport Resources and the Microsoft Cloud
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
    - `owner_uid` - The identifier for the owner of the A-End Port.
    - `name` - The name of the A-End Port.
    - `location` - The location name for the A-End Port.
    - `assigned_vlan` - The VLAN that was assigned by Megaport to the A-End Port.
- `b_end`:
    - `owner_uid` - The identifier for the owner of the B-End port.
    - `name` - The name of the B-End port.
    - `location` - The location name for the B-End port.
    - `assigned_vlan` - The VLAN that was assigned by Megaport to the B-End port.

## Import
VXCs can be imported using the `uid`, for example:
 ```shell script
# terraform import megaport_azure_connection.example_expressroute 2bea989c-4329-4a79-ae07-3522eb148c8f
```
