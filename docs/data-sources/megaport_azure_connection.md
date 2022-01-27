# Data Source Megaport Azure Connection
Use this data source to query an Azure ExpressRoute in your account.

> **Note**: This query only returns basic VXC details for the ExpressRoute; it does not return the
> service key.

## Example Usage
```
data "megaport_azure_connection" "azure_vxc" {
  vxc_id = "f2ff815a-ecba-440c-af7e-45a1585df3c1"
}
```

## Argument Reference
- `vxc_id` - (Required) The identifier of the Azure ExpressRoute connection.

## Attributes Reference
- `uid` - The identifier of the Azure ExpressRoute connection.
- `vxc_name` - The Azure ExpressRoute connection name.
- `rate_limit` - The speed of the Azure ExpressRoute connection in Mbps.
- `vxc_type` - The Azure ExpressRoute connection type.
- `provisioning_status` - The current provisioning status of the Azure ExpressRoute connection (this status does not refer to availability).
- `create_date` - A Unix timestamp representing the time the Azure ExpressRoute connection was created.
- `created_by` - The user who created the Azure ExpressRoute connection.
- `live_date` - A Unix timestamp representing the date the Azure ExpressRoute connection went live.
- `company_name` - The name of the company who owns the account where the Azure ExpressRoute connection is located.
- `locked` - Indicates whether the resource has been locked by a user.
- `admin_locked` - Indicates whether the resource has been locked by an administrator.
- `vxc_internal_type` - An internal variable used by Terraform to orchestrate CSP VXCs.
- `a_end`
    - `port_id` - The resource id of the Port (A-End) for the Azure ExpressRoute connection.
    - `owner_uid` - The owner id of the A-End resource for the Azure ExpressRoute connection.
    - `name` - The name of the A-End resource.
    - `location` - The resource location name.
    - `assigned_vlan` - The VLAN assigned by Megaport to the A-End resource.
- `b_end`
    - `port_id` - The resource id of the Azure ExpressRoute connection (B-End).
    - `owner_uid` - The owner id of the B-End resource for the Azure ExpressRoute connection.
    - `name` - The name of the B-End resource.
    - `location` - The resource location name.
    - `assigned_vlan` - The VLAN assigned by Megaport to the B-End resource.
