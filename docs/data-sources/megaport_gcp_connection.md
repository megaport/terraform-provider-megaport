# Data Source Megaport GCP Connection
Use this data source to query a Google Cloud Platform (GCP) connection in your account.

> **Note**: This query only returns basic VXC details for the GCP connection; it does not return
> GCP connection-specific details such as the pairing key.

## Example Usage
```
data "megaport_gcp_connection" "test" {
  vxc_id = "10b03cf2-e3c3-46fb-85c5-3ea4121d9178"
}
```

## Argument Reference

- `vxc_id` - (Required) The identifier of the GCP connection.

## Attribute Reference

- `uid` - The identifier of the GCP connection.
- `vxc_name` - The GCP connection name.
- `rate_limit` - The GCP connection speed in Mbps.
- `vxc_type` - The GCP connection type.
- `provisioning_status` - The current provisioning status of the GCP connection (this status does not refer to availability).
- `create_date` - A Unix timestamp representing the time the GCP connection was created.
- `created_by` - The user who created the GCP connection.
- `live_date` - A Unix timestamp representing the date the GCP connection went live.
- `company_name` - The name of the company that owns the account where the GCP connection is located.
- `locked` - Indicates whether the resource has been locked by a user.
- `admin_locked` - Indicates whether the resource has been locked by an administrator.
- `vxc_internal_type` - An internal variable used by terraform to orchestrate CSP VXCs.
- `a_end`
- `port_id` - The resource id of the Port (A-End) for the GCP connection.
- `owner_uid` - The owner id of the A-End resource for the GCP connection.
- `name` - The name of the A-End resource.
- `location` - The resource location name.
- `assigned_vlan` - The VLAN assigned by Megaport to the A-End resource.
- `b_end`
- `port_id` - The resource id of the GCP connection (B-End).
- `owner_uid` - The owner id of the B-End resource for the GCP connection.
- `name` - The name of the B-End resource.
- `location` - The resource location name.
- `assigned_vlan` - The VLAN assigned by Megaport to the B-End resource.
