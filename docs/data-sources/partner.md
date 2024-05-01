---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "megaport_partner Data Source - terraform-provider-megaport"
subcategory: ""
description: |-
  Partner Port
---

# megaport_partner (Data Source)

Partner Port



<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `company_name` (String) The name of the company that owns the partner port.
- `company_uid` (String) The unique identifier of the company that owns the partner port.
- `connect_type` (String) The type of connection for the partner port.
- `diversity_zone` (String) The diversity zone of the partner port.
- `location_id` (Number) The unique identifier of the location of the partner port.
- `product_name` (String) The name of the partner port.

### Read-Only

- `product_uid` (String) The unique identifier of the partner port.
- `rank` (Number) The rank of the partner port.
- `speed` (Number) The speed of the partner port.
- `vxc_permitted` (Boolean) Whether VXCs are permitted on the partner port.