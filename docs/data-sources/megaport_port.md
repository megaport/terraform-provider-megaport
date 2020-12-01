Use this data source to find an existing Port to use in resource calls.

## Example Usage (Single)
```
data megaport_port example_port {
    port_id = "e7bcbc04-be2d-4947-98b6-55b72a88e25b"
}
```

## Argument Reference
The following arguments are supported:
 - `port_id` - (Required) The Port identifier.
 
 ## Attributes Reference
 - `port_id` - The Port identifier.
 - `uid` - The Globally Unique Identifier (GUID) for the service.
 - `port_name` - The Port name.
 - `port_speed` - The Port speed in Mbps.
 - `location_id` - The identifier for the data center where the port is located.
 - `term` - The contract term for the Port. "1" means month-to-month; otherwise, the number represents the number of months in the term.
 - `type` - The Port type (`Single` or part of a Link Aggregation Group `LAG`).
 - `provisioning_status` - The current provisioning status of the Port (the status does not refer to availability).
 - `create_date` - A Unix timestamp representing the time the Port was created.
 - `created_by` - The user who created the Port.
 - `live_date` - A Unix timestamp representing the date the Port went live.
 - `market_code` - A short code for the billing market of the Port.
 - `marketplace_visibility` - Indicates whether the Port is available on the Megaport Marketplace.
 - `company_name` - The name of the company that owns the account for the Port.
 - `lag_primary` - Indicates whether the Port is a primary for a LAG.
 - `lag_id` - The LAG identifier.
 - `locked` - Indicates whether the resource has been locked by a user.
 - `admin_locked` - Indicates whether the resource has been locked by an administrator.

