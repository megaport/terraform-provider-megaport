# Data Source Megaport Internet
Use this data source to find a Megaport partner port ID within a given metro that is able to deliver an Internet VXC. Typically, the `id` of the location
will be used in other resource calls.

## Example Usage
```
data "megaport_internet" "sydney_blue" {
  metro                    = "Sydney"
  requested_diversity_zone = "blue"
}
```

This will return a port UID which can be used as a VXC B-end for ordering an Internet VXC.

To support some specific use cases, the B-end port may not be honoured when ordering an Internet VXC on an MVE. In this case, re-running `terraform apply` will try to update the VXC in-place, the update will "complete" successfully but will not actually make any changes to the network.

## Argument Reference
 - `metro` - (Required) The metro/city to search.
 - `requested_diversity_zone` - (Optional) The preferred diversity zone for the B-end port.
 
## Attributes Reference
- `id` - Set to the identifier of the location, which is an integer.
