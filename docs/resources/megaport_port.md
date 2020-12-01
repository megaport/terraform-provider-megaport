Creates a Port, which is a physical Port in a Megaport-enabled location. Common connection scenarios include:

 - **Port-to-Port** - Create a VXC between two Ports to enable cross-data center connectivity.
 - **Port-to-Cloud** - Create a VXC between your Port and a cloud service provider.
 - **Port-to-MCR** - Create a VXC between your Port and an MCR to enable access to services connected to the MCR.
 - **Port-to-IX** - Create a VXC between your Port and IX to provide your customers with access to service providers.

 **Note**: A Port is a physical resource. Before you can use a Port, you must establish a cross connect to it in your
 provider's data center. To do this, download the Letter of Authorization from the Megaport Portal and send the completed document to the data center provider requesting a physical connection to Megaport's network.

## Example Usage (Single)
```
data megaport_location nextdc_brisbane_1 {
    name = "NextDC B1"
}

resource megaport_port my_port {
    port_name       = "My Example Port"
    port_speed      = 10000
    location_id     = data.megaport_location.nextdc_brisbane_1.id
    term            = 12
}
```

This example results in the creation of a single Port located at NextDC Brisbane 1, under a 12 month term with a Port
speed of 10 Gbps.

## Example Usage (Link Aggregation Group)
```
data megaport_location nextdc_brisbane_1 {
    name = "NextDC B1"
}

resource megaport_port my_lag_port {
    port_name       = "My Example LAG Port"
    port_speed      = 10000
    location_id     = data.megaport_location.nextdc_brisbane_1.id
    term            = 1
    lag             = true
    lag_port_count  = 4
}
```

This example results in the creation of a LAG Port with 4 Ports located at NextDC Brisbane 1, under a 1 month term with 
an aggregate speed of 40 Gbps.

## Argument Reference

The following arguments are supported:

 - `port_name` - (Required) The name of the Port.
 - `port_speed` - (Required) The speed of the Port in Mbps.
 - `location_id` - (Required) The identifier for the data center where you want to create the Port.
 - `term` - (Optional) The contract term for your Port. The default is month-to-month.
 - `lag` - (Optional) Indicates that you want the Port to be a member of a Link Aggregation Port (LAG). A LAG is a set of physical Ports that are grouped into a single logical connection.
 - `lag_port_count` - (Optional) The number of Ports you would like in your LAG. This argument should only be used if
 `lag` is true. Note, the LAG Port speed will be this number multiplied by `port_speed`.
 - `marketplace_visibility` - (Optional) Whether to make this Port public on the 
 [Megaport Marketplace](https://docs.megaport.com/marketplace/).
 
 ## Attribute Reference

 In addition to the arguments, the following attributes are exported:
 - `uid` - The Globally Unique Identifier (GUID) for the service.
 - `type` - The type of Port (single or LAG).
 - `provisioning_status` - The current provisioning status of the Port (this status does not refer to availability).
 - `create_date` - A Unix timestamp representing the time the Port was created.
 - `created_by` - The user who created the Port.
 - `live_date` - A Unix timestamp representing when the Port went live.
 - `market_code` - A short code for the billing market of the Port.
 - `marketplace_visibility` - Indicates whether the Port is available on the [Megaport Marketplace](https://docs.megaport.com/marketplace/).
 - `company_name` - The name of the company that owns the account for the Port.
 - `lag_primary` - Indicates whether the Port is a primary for a LAG.
 - `lag_id` - The LAG identifier.
 - `locked` - Indicates whether the resource has been locked by a user.
 - `admin_locked` - Indicates whether the resource has been locked by an administrator.
 
 ## Import
 Ports can be imported using the `uid`, for example:
 ```shell script
# terraform import megaport_port.example_port 03fec669-f2a8-4488-9be9-def6f7998a10
```
