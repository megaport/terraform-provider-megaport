# Data Source Megaport VXC
Use this data source to find an existing Virtual Cross Connect (VXC) to use in resource calls.

## Example Usage
```
data "megaport_vxc" "vxc" {
  vxc_id = "522b1d65-d232-404c-bae9-bb8643cd4b3e"
}
```

## Argument Reference
- `vxc_id` - (Required) The VXC identifier.

## Attributes Reference
- `uid` - The VXC identifier.
- `vxc_name` - The VXC name.
- `rate_limit` - The VXC speed in Mbps.
- `vxc_type` - The VXC type.
- `provisioning_status` - The current provisioning status of the VXC (this status does not refer to availability).
- `create_date` - A Unix timestamp representing the time the VXC was created.
- `created_by` - The user who created the VXC.
- `live_date` - A Unix timestamp representing the date the VXC went live.
- `company_name` - The name of the company that owns the account where the VXC is located.
- `locked` - Indicates whether the resource has been locked by a user.
- `admin_locked` - Indicates whether the resource has been locked by an administrator.
- `vxc_internal_type` - An internal variable used by Terraform to orchestrate CSP VXCs.
- `a_end`
    - `port_id` - The Port that the VXC A-End is attached to.
    - `owner_uid` - The identifier for the owner of the A-End Port.
    - `name` - The A-End Port name.
    - `location` - The name of the data center where the Port is located.
    - `assigned_vlan` - The VLAN that Megaport assigned to the A-End Port.
- `b_end`
    - `port_id` - The Port that the VXC B-End is attached to.
    - `owner_uid` - The identifier for the owner of the B-End Port.
    - `name` - The B-End Port name.
    - `location` - The name of the data center where the Port is located.
    - `assigned_vlan` - The VLAN that Megaport assigned to the B-End port.
 

## Import
You can import VXCs using the `uid`. For example:
 ```shell script
# terraform import megaport_vxc.example_vxc 522b1d65-d232-404c-bae9-bb8643cd4b3e
```
