# Resource Megaport MCR
Creates a Megaport Cloud Router (MCR). An MCR is a virtual Layer 3 router that enables interconnection between resources in Megaport's network. Common connection scenarios include:

 - **Multi-cloud** - Use an MCR to connect networks between two or more separate cloud service providers.
 - **Transit** - Use an MCR as the central touchpoint between your Ports and cloud providers instead of provisioning VXCs between every Port in your network.

Unlike [Ports](Resource_megaport_port), MCRs are not available at all locations.

## Example Usage
```
data megaport_location nextdc_brisbane_1 {
    name    = "NextDC B1"
    has_mcr = true
}

resource megaport_mcr test {
    mcr_name = "Terraform Test - MCR"
    location_id = data.megaport_location.ndc_b1.id

    router {
        port_speed = 10000
        requested_asn = 64555
    }
}
```

## Argument Reference
The following arguments are supported:
- `mcr_name` - (Required) The name for the MCR.
- `location_id` - (Required) The identifier of the preferred data center location for the MCR. This location must be MCR-enabled.
- `router`
    - `requested_asn` - (Optional) The Autonomous System Number (ASN) to assign to the MCR.
    - `port_speed` - (Required) The speed of the MCR in Mbps. The value can be between 1000 and 10000 Mbps.
    
## Attribute Reference

- `uid` - The UID of the MCR.
- `type` - The type of MCR (MCR/MCR2).
- `mcr_name` - The name for the MCR.
- `location_id` - The identifier of the preferred data center location for the MCR. This location must be MCR-enabled.
- `provisioning_status` - The current provisioning status of the MCR (this status does not refer to availability).
- `create_date` - A Unix timestamp representing the time the MCR was created.
- `created_by` - The user who created the MCR.
- `live_date` - A Unix timestamp representing the time the MCR became live.
- `market_code` - A short code for the billing market of the MCR.
- `marketplace_visibility` - Indicates whether the MCR is available on the Megaport Marketplace.
- `company_name` - The name of the company that owns the account for the MCR.
- `locked` - Indicates whether the resource has been locked by a user.
- `admin_locked` - Indicates whether the resource has been locked by an admin.
- `router`
    - `assigned_asn` - The ASN assigned by Megaport.
    - `requested_asn` - The ASN to assign to the MCR.
    - `port_speed` - The speed of the MCR in Mbps. The value can be between 1000 and 10000 Mbps.

## Import
MCRs can be imported using the `uid`, for example
 ```shell script
# terraform import megaport_mcr.example_mcr f7998669-4488-4488-9be9-6696f7998a10
```
