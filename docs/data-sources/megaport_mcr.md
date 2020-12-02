# Data Source Megaport MCR
Use this data source to find an existing Megaport Cloud Router (MCR) to use in resource calls.

## Example Usage (Single)
```
data megaport_mcr example_mcr {
    mcr_id = "e7bcbc04-be2d-4947-98b6-55b72a88e25b"
}
```

## Argument Reference
The following arguments are supported:
 - `mcr_id` - (Required) The identifier of the MCR.

## Attributes Reference
- `uid` - The UID of the MCR.
- `mcr_name` - The name of the MCR.
- `location_id` - The identifier of the MCR location.
- `type` - The type of MCR (MCR or MCR 2.0).
- `provisioning_status` - The current provisioning status of the MCR (this status does not refer to availability).
- `create_date` - A Unix timestamp representing the time the MCR was created.
- `created_by` - The user who created the MCR.
- `live_date` - A Unix timestamp representing the time the MCR went live.
- `market_code` - A short code for the billing market where the MCR is located.
- `marketplace_visibility` - Indicates whether the MCR is available on the Megaport Marketplace.
- `company_name` - The name of the company that owns the account where the MCR is located.
- `locked` - Indicates whether the resource has been locked by a user.
- `admin_locked` - Indicates whether the resource has been locked by an administrator.
- `router`
    - `assigned_asn` - The ASN assigned by Megaport.
    - `port_speed` - (Required) The MCR speed in Mbps.
