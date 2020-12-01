Adds a connection to Google Compute Cloud on a Port or an MCR. 

To provision this connection, you need a partner key from the Google Cloud Console. You can get this key by adding a new VLAN attachment and selecting "Megaport" as the interconnect partner.
 
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

resource megaport_gcp_connection test {
  vxc_name = "My Example Google Connection"
  rate_limit = 1000

  a_end {
    requested_vlan = 191
  }

  csp_settings {
    attached_to = megaport_port.my_port.id
    pairing_key = "19f9d93e-05c8-4c18-81fc-095d679ff645/australia-southeast-1/1"
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
    - `pairing_key` - (Required) The pairing key for the new GCP connection.

## Attributes Reference
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
    - `owner_uid` - The identifier of the owner of the A-End Port.
    - `name` - The name of the A-End Port.
    - `location` - The location name for the A-End Port.
    - `assigned_vlan` - The VLAN assigned by Megaport to the A-End Port.
- `b_end`:
    - `owner_uid` - The identifier of the owner of the B-End port.
    - `name` - The name of the B-End port.
    - `location` - The location name for the B-End port.
    - `assigned_vlan` - The VLAN assigned by Megaport to the B-End port.

## Import
VXCs can be imported using the `uid`, for example:
 ```shell script
# terraform import megaport_gcp_connection.example_gcp a5b5464b-b5c5-41b4-85b2-86560bb85207
```
