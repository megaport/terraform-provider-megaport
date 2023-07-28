# Data Source Megaport OCI Connection
Use this data source to query an OCI VXC in your account.

> **Note**: This query only returns basic VXC details for the OCI VXC.

## Example Usage
```
data "megaport_oci_connection" "oci_vxc" {
  vxc_id = "3dabec96-fa03-4d71-8285-afca12264579"
}
```

## Argument Reference
- `vxc_id` - (Required) The identifier of the OCI connection.

## Attributes Reference
- `uid` - The identifier of the OCI connection.
- `vxc_name` - The OCI connection name.
- `rate_limit` - The speed of the OCI connection in Mbps.
- `vxc_type` - The OCI connection type.
- `provisioning_status` - The current provisioning status of the OCI connection (this status does not refer to availability).
- `create_date` - A Unix timestamp representing the time the OCI connection was created.
- `created_by` - The user who created the OCI connection.
- `live_date` - A Unix timestamp representing the date the OCI connection went live.
- `company_name` - The name of the company who owns the account where the OCI connection is located.
- `locked` - Indicates whether the resource has been locked by a user.
- `admin_locked` - Indicates whether the resource has been locked by an administrator.
- `vxc_internal_type` - An internal variable used by Terraform to orchestrate CSP VXCs.
- `a_end`:
    - `port_id` - The resource id of the Port (A-End) for the OCI connection.
    - `owner_uid` - The owner id of the A-End resource for the OCI connection.
    - `name` - The name of the A-End resource.
    - `location` - The resource location name.
    - `assigned_vlan` - The VLAN assigned by Megaport to the A-End resource.
- `b_end`:
    - `port_id` - The resource id of the OCI connection (B-End).
    - `owner_uid` - The owner id of the B-End resource for the OCI connection.
    - `name` - The name of the B-End resource.
    - `location` - The resource location name.
    - `assigned_vlan` - The VLAN assigned by Megaport to the B-End resource.
