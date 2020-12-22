# Data Source Megaport Location
Use this data source to find a Megaport-enabled Location ID. Typically, the `id` of the location
will be used in other resource calls.

## Example Usage
```
data megaport_location my_loc {
    name        = "NextDC B"
    market_code = "AU"
    has_mcr     = true
}
```
This will return the location details for `NextDC B1` as it is the only MCR-enabled location in Brisbane.

```
data megaport_location my_loc {
    name        = "NextDC B1"
}
```

This will return the location identifier for the NextDC B1 data center.

## Argument Reference
 - `name` - (Optional) The name of the data center to search for. These can be found on the [Megaport Enabled Locations](https://www.megaport.com/megaport-enabled-locations/) page.
 - `market_code` - (Optional) The short market code of the country you want to search for. These can be found [here](https://api.megaport.com/v2/networkRegions).
 - `has_mcr` - (Optional) If set to true, only MCR-enabled locations will be displayed.
 - `match_exact` - (Optional) If set to true, the name search return must be an exact match or no results will be displayed. Other filters will not be applied.
 
> **Note**: If more than one result exists after all filters have been applied, a `too many results` error will be returned.

## Attributes Reference
- `id` - Set to the identifier of the location, which is an integer.
- `name` - The full name of the resource.
- `country` - The full name of the country the data center is located in.
- `live_date` - The date the data center was made available to customers.
- `site_code` - The internal site identifier.
- `network_region` - The internal network service identifier.
- `address` - The data center address, with standard address attributes:
    - `street`
    - `suburb`
    - `city`
    - `state`
    - `postcode`
    - `country`
- `latitude` - The latitudinal coordinates for the data center.
- `longitude` - The longitudinal coordinates for the data center.
- `market` - A short code for the billing market the Port is in.
- `metro` - The name of the metro the data center is located in.
- `mcr_available` - Indicates whether the site is MCR-enabled.
- `status` - Indicates whether the site is active.
