# Data Source Megaport Partner Port
Use this data source to find Partner Ports from the Megaport Marketplace. This is primarily used by AWS-based Megaport services, as other providers have built-in lookups as part of their key-based setup.

## Example Usage (AWS Hosted VIF)
```
data "megaport_location" "syd_gs" {
  name = "Global Switch Sydney West"
}

data "megaport_partner_port" "aws_port_1" {
  connect_type = "AWS"
  company_name = "AWS"
  product_name = "Asia Pacific (Sydney) (ap-southeast-2)"
  location_id  = data.megaport_location.syd_gs.id
}
```

## Example Usage (AWS Hosted Connection)
```
data "megaport_location" "syd_gs" {
  name = "Global Switch Sydney West"
}

data "megaport_partner_port" "aws_port_2" {
  connect_type   = "AWSHC"
  company_name   = "AWS"
  product_name   = "Asia Pacific (Sydney) (ap-southeast-2) [DZ-RED]"
  diversity_zone = "red"
  location_id    = data.megaport_location.syd_gs.id
}
```

## Example Usage (Google Connection)
```
data "megaport_location" "sea_eq2" {
  name = "Equinix SE2"
}

data "megaport_partner_port" "google_port" {
  connect_type = "GOOGLE"
  company_name = "Google Inc"
  product_name = "Seattle (sea-zone1-86)"
  location_id  = data.megaport_location.sea_eq2.id
}
```

## Argument Reference
 - `connect_type` - (Optional) The type of connection you will create. In the case of AWS, specify `AWS` for a Hosted VIF or `AWSHC` for a Hosted Connection).
 - `company_name` - (Optional) The company name to search for (from the company's Megaport Marketplace profile).
 - `product_name` - (Optional) The product name, as it appears in the Megaport Marketplace.
 - `diversity_zone` - (Optional) The diversity zone of the product, as it appears in the Megaport Marketplace.
 - `location_id` - (Optional) The id of the location where you want to provision the product.

> **Note**: This lookup is based on a filter. Be sure to only include the arguments that will get you the results you need.
> For example, to get a Hosted Connection at Global Switch Sydney West, enter the `location_id = 3` for
> the data center, 'AWSHC' for the `connect_type`, 'AWS' for the `company_name` and 'red' or 'blue' for the `diversity_zone`. 
> `product_name` is not needed in this case. 
>
> **Important**: If more than one result is found, a `too many results message` error will be returned. Ensure that the filter is *specific*.

## Attribute Reference
- `company_uid` - The unique identifier for the company.
- `speed` - The Port speed in Mbps.
