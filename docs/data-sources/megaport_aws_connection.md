Use this data source to query an AWS connection in your account. This connection can be one of the following:

 - Hosted Virtual Interface (VIF) - A single VIF that is added to the target AWS account.
 - Hosted Connection - A full Direct Connect that has a VIF added to your account.

> **Note**: This query only returns basic VXC details for the AWS connection; it does not return 
> AWS connection-specific details such as BGP Auth Key or ASN values.


## Example Usage
```
data "megaport_aws_connection" "test" {
  vxc_id = "deb28049-a881-4066-b2fd-27bca092f3d3"
}
```

## Argument Reference
- `vxc_id` - (Required) The identifier of the AWS connection.

## Attribute Reference
- `uid` - The identifier of the AWS connection.
- `vxc_name` - The AWS connection name.
- `rate_limit` - The speed of the AWS connection in Mbps.
- `vxc_type` - The AWS connection type.
- `provisioning_status` - The current provisioning status of the AWS connection (this status does not refer to availability).
- `create_date` - A Unix timestamp representing the time the AWS connection was created.
- `created_by` - The user who created the AWS connection.
- `live_date` - A Unix timestamp representing the date the AWS Connection went live.
- `company_name` - The name of the company that owns the account where the AWS connection is located.
- `locked` - Indicates whether the resource has been locked by a user.
- `admin_locked` - Indicates whether the resource has been locked by an administrator.
- `vxc_internal_type` - An internal variable used by Terraform to orchestrate CSP VXCs.
- `a_end`
    - `port_id` - The resource id of the Port (A-End) for the AWS connection.
    - `owner_uid` - The owner id of the A-End resource for the AWS connection.
    - `name` - The name of the A-End resource.
    - `location` - The resource location name.
    - `assigned_vlan` - The VLAN assigned by Megaport to the A-End resource.
- `b_end`
    - `port_id` - The resource id of the AWS connection (B-End).
    - `owner_uid` - The owner id of the B-End resource for the AWS connection.
    - `name` - The name of the B-End resource.
    - `location` - The resource location name.
    - `assigned_vlan` - The VLAN assigned by Megaport to the B-End resource.
