---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "megaport_ix Resource - terraform-provider-megaport"
subcategory: ""
description: |-
  Manages a Megaport Internet Exchange (IX).
---

# megaport_ix (Resource)

Manages a Megaport Internet Exchange (IX).

## Example Usage

```terraform
resource "megaport_port" "test_port" {
  product_name           = "Test Port for IX"
  location_id            = 67
  port_speed             = 1000
  marketplace_visibility = false
  contract_term_months   = 1
}

resource "megaport_ix" "test_ix" {
  name                  = "Test IX Connection"
  requested_product_uid = megaport_port.test_port.product_uid
  network_service_type  = "Sydney IX"
  asn                   = 65000
  mac_address           = "00:11:22:33:44:55"
  rate_limit            = 500
  vlan                  = 2000
  shutdown              = false
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `mac_address` (String) The MAC address for the IX interface.
- `network_service_type` (String) The type of IX service, e.g., 'Los Angeles IX', 'Sydney IX'.
- `product_name` (String) Name of the IX.
- `rate_limit` (Number) The rate limit in Mbps for the IX connection.
- `requested_product_uid` (String) UID identifier of the product to attach the IX to.
- `vlan` (Number) The VLAN ID for the IX connection.

### Optional

- `asn` (Number) The ASN (Autonomous System Number) for the IX connection.
- `attribute_tags` (Map of String) Attribute tags associated with the IX.
- `cost_centre` (String) Cost centre for invoicing purposes.
- `promo_code` (String) Promo code to apply to the IX.
- `public_graph` (Boolean) Whether the IX usage statistics are publicly viewable.
- `reverse_dns` (String) Custom hostname for your IP address.
- `shutdown` (Boolean) Whether the IX connection is shut down. Default is false.

### Read-Only

- `create_date` (String) The date the IX was created.
- `deploy_date` (String) The date the IX was deployed.
- `ix_peer_macro` (String) IX peer macro configuration.
- `location_id` (Number) The ID of the location where the IX is provisioned.
- `product_id` (Number) Numeric ID of the IX product.
- `product_uid` (String) UID identifier of the IX product.
- `provisioning_status` (String) The provisioning status of the IX.
- `resources` (Attributes) Resources associated with the IX. (see [below for nested schema](#nestedatt--resources))
- `secondary_name` (String) Secondary name for the IX.
- `term` (Number) The term of the IX in months.
- `usage_algorithm` (String) Usage algorithm for the IX.

<a id="nestedatt--resources"></a>
### Nested Schema for `resources`

Read-Only:

- `bgp_connections` (Attributes List) BGP connections for the IX. (see [below for nested schema](#nestedatt--resources--bgp_connections))
- `interface` (Attributes) Interface details for the IX. (see [below for nested schema](#nestedatt--resources--interface))
- `ip_addresses` (Attributes List) IP addresses for the IX. (see [below for nested schema](#nestedatt--resources--ip_addresses))
- `vpls_interface` (Attributes) VPLS interface details for the IX. (see [below for nested schema](#nestedatt--resources--vpls_interface))

<a id="nestedatt--resources--bgp_connections"></a>
### Nested Schema for `resources.bgp_connections`

Read-Only:

- `asn` (Number) ASN for the BGP connection.
- `customer_asn` (Number) Customer ASN for the BGP connection.
- `customer_ip_address` (String) Customer IP address for the BGP connection.
- `isp_asn` (Number) ISP ASN for the BGP connection.
- `isp_ip_address` (String) ISP IP address for the BGP connection.
- `ix_peer_policy` (String) IX peer policy.
- `max_prefixes` (Number) Maximum prefixes.
- `resource_name` (String) Resource name.
- `resource_type` (String) Resource type.


<a id="nestedatt--resources--interface"></a>
### Nested Schema for `resources.interface`

Read-Only:

- `demarcation` (String) Demarcation point for the interface.
- `loa_template` (String) LOA template for the interface.
- `media` (String) Media type for the interface.
- `port_speed` (Number) Port speed in Mbps.
- `resource_name` (String) Resource name.
- `resource_type` (String) Resource type.
- `shutdown` (Boolean) Whether the interface is shut down.
- `up` (Number) Interface up status.


<a id="nestedatt--resources--ip_addresses"></a>
### Nested Schema for `resources.ip_addresses`

Read-Only:

- `address` (String) IP address.
- `resource_name` (String) Resource name.
- `resource_type` (String) Resource type.
- `reverse_dns` (String) Reverse DNS for this IP address.
- `version` (Number) IP version (4 or 6).


<a id="nestedatt--resources--vpls_interface"></a>
### Nested Schema for `resources.vpls_interface`

Read-Only:

- `mac_address` (String) MAC address for the VPLS interface.
- `rate_limit_mbps` (Number) Rate limit in Mbps for the VPLS interface.
- `resource_name` (String) Resource name.
- `resource_type` (String) Resource type.
- `shutdown` (Boolean) Whether the VPLS interface is shut down.
- `vlan` (Number) VLAN ID for the VPLS interface.

## Import

Import is supported using the following syntax:

```shell
# Order can be imported by specifying the Product UID.
terraform import megaport_ix.example "<PRODUCT_UID>"
```
