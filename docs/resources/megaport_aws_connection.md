# Resource Megaport AWS Connection
Adds a connection to AWS on either a Port or a MCR. This connection can be one of the following:

 - **Hosted VIF** - A single VIF that is added to the target AWS account.
 - **Hosted Connection** - A full Direct Connect which has a VIF added to your account.
 
Before you can provision an AWS connection, you need to do a megaport_partner_port lookup to get the product identifier of the AWS ports from the Megaport Marketplace.
 
## Example Usage
### AWS VIF Connection

```
data megaport_location glb_switch_sydney {
  name = "Global Switch Sydney West"
}

data megaport_partner_port aws_sydney {
  connect_type = "AWS"
  company_name = "AWS"
  product_name = "Asia Pacific (Sydney) (ap-southeast-2)"
  location_id = data.megaport_location.glb_switch_sydney.id
}

resource megaport_port my_port {
    port_name       = "My Example Port"
    port_speed      = 1000
    location_id     = data.megaport_location.glb_switch_sydney.id
}

resource megaport_aws_connection test {
  vxc_name = "My Example Hosted VIF"
  rate_limit = 1000

  a_end {
    requested_vlan = 191
  }

  csp_settings {
    attached_to = megaport_port.my_port.id
    requested_product_id = data.megaport_partner_port.aws_sydney.id
    requested_asn = 64550
    amazon_asn = 64551
    amazon_account = "123456789012"
  }
}
```

### AWS Hosted Connection with BGP and BFD enabled

```
data megaport_location sydney {
  name = "Equinix SY3"
}

data megaport_location ndc_b1 {
  name    = "NextDC B1"
  has_mcr = true
}

resource megaport_mcr test {
  mcr_name    = "Terraform Test - MCR"
  location_id = data.megaport_location.ndc_b1.id

  router {
    port_speed    = 2500
    requested_asn = 64555
  }

}

data megaport_partner_port aws_hc {
  connect_type = "AWSHC"
  company_name = "AWS"
  product_name = "Asia Pacific (Sydney) (ap-southeast-2) [DZ-BLUE]"
  location_id  = data.megaport_location.sydney.id
}

resource megaport_aws_connection test {
  vxc_name = "My Example Hosted VIF"
  rate_limit = 1000

    a_end {

    requested_vlan = 0
    
    partner_configuration {

      ip_addresses = [ "11.192.0.25/29" ,"12.192.0.25/29" ]

      bfd_configuration {
        tx_internal = 500
        rx_internal = 400
        multiplier = 5
      }

      bgp_connection {
        peer_asn = 62512
        local_ip_address = "12.192.0.25"
        peer_ip_address = "12.192.0.26"
        password = "updated-secure-password-2"
        shutdown = true
        description = "BGP with MED and BFD enabled"
        med_in = 100
        med_out = 100
        bfd_enabled = false
      }

    }

  }

  csp_settings {
    attached_to = megaport_port.my_port.id
    requested_product_id = data.megaport_partner_port.aws_sydney.id
    requested_asn = 64550
    amazon_asn = 64551
    amazon_account = "123456789012"
  }
}
```

## Argument Reference
- `vxc_name` - (Required) The name of your VXC.
- `rate_limit` - (Required) The speed of your VXC in Mbps.
- `a_end`
    - `requested_vlan` - (Required) the vLAN you want assigned to the A-End Port.
    - `partner_configuration` - (Optional) allows you to customise AWS connection-specific details
        - `ip_addresses` - (Optional) List of IP address and associated subnet mask to be configured on this interface.
        - `bgp_connection` - (Optional) BGP peering relationships for this interface - maximum of five. Requires an Interface IP Address to be created.
            - `peer_asn` - (Required) The ASN of the remote BGP peer.
            - `local_ip_address` - (Required) The IPv4 or IPv6 address on this interface to use for communication with the BGP peer.
            - `peer_ip_address` - (Required) The IP address of the BGP peer.
            - `password` - (Optional) A shared key used to authenticate the BGP peer, up to 25 characters.
            - `shutdown` - (Optional) By default, BGP connections are enabled and will actively attempt to connect to the peer. Select shutdown to temporarily disable the BGP session without removing it. This may be useful for troubleshooting or testing fail over scenarios.
            - `description` - (Optional) A description for the BGP connection, up to 100 characters.
            - `med_in` - (Optional) The MED will be applied to all routes received on this BGP connection. Leave blank to use the value received from the BGP peer. The route with the lowest value will be preferred.
            - `med_out` - (Optional) The MED will be applied to all routes transmitted on this BGP connection. The neighbouring autonomous system may prefer the lowest value at their discretion.
            - `bfd_enabled` - (Optional) Must be true for BFD configuration to be honoured - default is false
        - `bfd_configuration` -  (Optional) Bidirectional Forwarding Detection. These settings will be used for all BGP connections on this interface where BFD is enabled.
            - `tx_interval` -  (Optional) The minimum time between sending BFD packets to the neighbour. The supported range is 300ms to 9000ms.
            - `rx_interval` -  (Optional) The minimum time between BFD packets that a neighbour should send. The supported range is 300ms to 9000ms.
            - `multiplier` -  (Optional) The BGP session will be torn down if this many consecutive BFD packets are not received from the neighbour.
- `csp_settings`:
    - `attached_to` - (Required) The identifier of the product (Port/MCR) to attach the connection to.
    - `requested_product_id` - (Required) The partner port on-ramp you want to connect to.
    - `visbility` - (Optional) The Direct Connect interface type.
    - `requested_asn` - (Required) The ASN for the AWS connection.
    - `amazon_asn` - (Required) The ASN set on the Direct Connect gateway in the Amazon account.
    - `amazon_account` - (Required) The Amazon account number.
    - `auth_key` - (Optional) The BGP auth key.
    - `prefixes` - (Optional) The IP prefixes for your connection.
    - `customer_ip` - (Optional) The internal tunnel IP for the Megaport end.
    - `amazon_ip` - (Optional) The internal tunnel IP for the Amazon end.
    - `hosted_connection` - (Optional) If set to true, an AWS Hosted Connection will be created with a dedicated Direct Connect. Otherwise, a Hosted VIF will be created.

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
    - `owner_uid` - The owner id of the A-End Port for this connection.
    - `name` - The name of the A-End Port.
    - `location` - The name of the location for the Port.
    - `assigned_vlan` - The VLAN assigned by Megaport to the A-End Port.
- `b_end`:
    - `owner_uid` - The owner id of the B-End port.
    - `name` - The name of the B-End port.
    - `location` - The location name for the B-End Port.
    - `assigned_vlan` - The VLAN assigned by Megaport to the B-End port.
- `csp_settings`:
    - `assigned_asn` - The ASN assigned by Megaport for this connection.
