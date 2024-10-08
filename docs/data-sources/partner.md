---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "megaport_partner Data Source - terraform-provider-megaport"
subcategory: ""
description: |-
  Partner Port Data Source. Returns the interfaces Megaport has with cloud service providers.
---

# megaport_partner (Data Source)

Partner Port Data Source. Returns the interfaces Megaport has with cloud service providers.

## Example Usage

```terraform
data "megaport_partner" "aws_port" {
  connect_type = "AWS"
  company_name = "AWS"
  product_name = "Asia Pacific (Sydney) (ap-southeast-2)"
  location_id  = 3
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `company_name` (String) The name of the company that owns the partner port.
- `company_uid` (String) The unique identifier of the company that owns the partner port.
- `connect_type` (String) The type of connection for the partner port. Filters the locations based on the cloud providers, such as AWS (for Hosted VIF), AWSHC (for Hosted Connection), AZURE, GOOGLE, ORACLE, OUTSCALE, and IBM. Use TRANSIT fto display Ports that support a Megaport Internet connection. Use FRANCEIX to display France-IX Ports that you can connect to.
- `diversity_zone` (String) The diversity zone of the partner port.
- `location_id` (Number) The unique identifier of the location of the partner port.
- `product_name` (String) The name of the partner port.

### Read-Only

- `product_uid` (String) The unique identifier of the partner port.
- `rank` (Number) The rank of the partner port.
- `speed` (Number) The speed of the partner port.
- `vxc_permitted` (Boolean) Whether VXCs are permitted on the partner port. If false, you can not create a VXC on this port. If true, you can create a VXC on this port.
