# Resource Megaport AWS Connection
Adds a connection to AWS on either a Port or a MCR. This connection can be one of the following:

 - **Hosted VIF** - A single VIF that is added to the target AWS account.
 - **Hosted Connection** - A full Direct Connect which has a VIF added to your account.
 
Before you can provision an AWS connection, you need to do a megaport_partner_port lookup to get the product identifier of the AWS ports from the Megaport Marketplace.
 
## Example Usage
### AWS VIF Connection

```
data "megaport_location" "syd_gs" {
  name = "Global Switch Sydney West"
}

data "megaport_partner_port" "aws_port" {
  connect_type = "AWS"
  company_name = "AWS"
  product_name = "Asia Pacific (Sydney) (ap-southeast-2)"
  location_id  = data.megaport_location.syd_gs.id
}

resource "megaport_port" "port" {
  port_name   = "Terraform Example - Port"
  port_speed  = 1000
  location_id = data.megaport_location.syd_gs.id
}

resource "megaport_aws_connection" "aws_vxc" {
  vxc_name   = "Terraform Example - AWS VXC"
  rate_limit = 1000

  a_end {
    port_id        = megaport_port.port.id
    requested_vlan = 191
  }

  csp_settings {
    requested_product_id = data.megaport_partner_port.aws_port.id
    requested_asn        = 64550
    amazon_asn           = 64551
    amazon_account       = "123456789012"
  }
}
```

## Argument Reference
- `vxc_name` - (Required) The name of your VXC.
- `rate_limit` - (Required) The speed of your VXC in Mbps.
- `destroy_connection` - (Optional, only for Hosted Connection) If set to true the AWS Connection (dxcon) will be deleted at destroy time. An external AWS session is required and the Connection cannot be deleted if it still has Virtual Interfaces attached.
- `a_end` - (Required) ** See VXC Documentation
- `a_end_mcr_configuration` - (Optional) ** See VXC Documentation
- `csp_settings`:
    - `requested_product_id` - (Required) The partner port on-ramp you want to connect to.
    - `visibility` - (Optional, only for VIF) The Direct Connect interface type for Hosted VIF ("public" or "private").
    - `requested_asn` - (Required for VIF) The ASN for the A End connection.
    - `amazon_asn` - (Optional, only for VIF or MCR) The ASN set on the Direct Connect gateway in the Amazon account.
    - `amazon_account` - (Required) The Amazon account number.
    - `auth_key` - (Optional, only for VIF) The BGP auth key.
    - `prefixes` - (Optional, only for VIF) The IP prefixes for your connection.
    - `customer_ip` - (Optional, only for VIF) The internal tunnel IP for the Megaport end.
    - `amazon_ip` - (Optional, only for VIF) The internal tunnel IP for the Amazon end.
    - `hosted_connection` - (Optional) If set to true, an AWS Hosted Connection will be created with a dedicated Direct Connect. Otherwise, a Hosted VIF will be created.
    - `connection_name` - (Optional) The label for the connection in AWS.

## Attributes Reference
- `uid` - The identifier of the Port.
- `vxc_type` - The VXC type.
- `provisioning_status` - The current provisioning status of the VXC (this status does not refer to availability).
- `create_date` - A Unix timestamp representing the time when the VXC was created.
- `created_by` - The user who created the VXC.
- `live_date` - A Unix timestamp representing the date the VXC went live.
- `company_name` - The name of the company that owns the account for the VXC.
- `locked` - Indicates whether the resource has been locked by a user.
- `admin_locked` - Indicates whether the resource has been locked by an admin.
- `vxc_internal_type` - An internal variable used by Terraform to orchestrate CSP VXCs.
- `a_end`:
    - `port_id` - The resource id of the Port (A-End) for the AWS connection.
    - `owner_uid` - The owner id of the A-End Port for this connection.
    - `name` - The name of the A-End Port.
    - `location` - The name of the location for the Port.
    - `assigned_vlan` - The VLAN assigned by Megaport to the A-End Port.
- `b_end`:
    - `port_id` - The resource id of the AWS connection (B-End).
    - `owner_uid` - The owner id of the B-End port.
    - `name` - The name of the B-End port.
    - `location` - The location name for the B-End Port.
    - `assigned_vlan` - The VLAN assigned by Megaport to the B-End port.
- `csp_settings`:
    - `assigned_asn` - The ASN assigned by Megaport for this connection.

## Import
AWS connections can be imported using the `uid`, for example:
```shell script
# terraform import megaport_aws_connection.aws_vxc 43aeaf75-d4e8-46e7-b312-95b931943f55
```

Certain csp_settings attributes are invalid for certain A-/B-end combinations (eg. BGP for Hosted Connections not using MCR). If you import a connection, csp_settings will be populated with only the valid attributes. Check the plan after you import, and remove any invalid attributes from your Terraform configuration if it shows a forced new resource for one of these.
