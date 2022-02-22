# Resource Megaport VXC
Creates a Virtual Cross Connect (VXC), which is a connection between two Megaport services. Common connection scenarios include:

 - **Port-to-Port** - Create a VXC between two Ports to enable multi-data center connectivity.
 - **Port-to-MCR** - Create a VXC between an MCR and a Port to allow the Port to access services attached to the MCR.
 
## Example Usage
```
data "megaport_location" "bne_nxt1" {
  name = "NextDC B1"
}

data "megaport_location" "bne_nxt2" {
  name = "NextDC B2"
}

resource "megaport_mcr" "mcr" {
  mcr_name    = "Terraform Example - MCR"
  location_id = data.megaport_location.bne_nxt1.id

  router {
    port_speed    = 5000
    requested_asn = 64555
  }
}

resource "megaport_port" "port" {
  port_name   = "Terraform Example - Port"
  port_speed  = 1000
  location_id = data.megaport_location.bne_nxt2.id
}

resource "megaport_vxc" "vxc" {
  vxc_name   = "Terraform Example - VXC"
  rate_limit = 1000

  a_end {
    port_id = megaport_mcr.mcr.id
  }

  a_end_mcr_configuration {
    ip_addresses = ["10.0.0.1/30"]
    nat_ip_addresses = ["10.0.0.1"]

    bfd_configuration {
      tx_interval = 500
      rx_interval = 400
      multiplier  = 5
    }

    bgp_connection {
      peer_asn         = 64512
      local_ip_address = "10.0.0.1"
      peer_ip_address  = "10.0.0.2"
      password         = "updated-secure-password-2"
      shutdown         = false
      description      = "BGP with MED and BFD enabled"
      med_in           = 100
      med_out          = 100
      bfd_enabled      = true
    }
  }

  b_end {
    port_id = megaport_port.port.id
  }
}
```

## Argument Reference
- `vxc_name` - (Required) The name of your VXC.
- `rate_limit` - (Required) The speed of your VXC in Mbps.
- `a_end`:
    - `port_id` - (Required) The identifier of the product (Port/MCR) to attach the connection to.
    - `requested_vlan` - (Required) The VLAN to assign to the A-End Port.
- `a_end_mcr_configuration` - (Optional) Configuration block for an A-End MCR if you wish to define a BGP Connection.
    - `ip_addresses` - (Optional) List of IP address and associated subnet mask to be configured on this interface.
    - `nat_ip_addresses` - (Optional) List of NAT IP address to be configured on this interface.
    - `bgp_connection` - (Optional) BGP peering relationships for this interface - maximum of five. Requires an Interface IP Address to be created.
        - `peer_asn` - (Required) The ASN of the remote BGP peer.
        - `local_ip_address` - (Required) The IPv4 or IPv6 address on this interface to use for communication with the BGP peer.
        - `peer_ip_address` - (Required) The IP address of the BGP peer.
        - `password` - (Optional) A shared key used to authenticate the BGP peer, up to 25 characters.
        - `shutdown` - (Optional) By default, BGP connections are enabled and will actively attempt to connect to the peer. Select shutdown to temporarily disable the BGP session without removing it. This may be useful for troubleshooting or testing fail over scenarios.
        - `description` - (Optional) A description for the BGP connection, up to 100 characters.
        - `med_in` - (Optional) The MED will be applied to all routes received on this BGP connection. Leave blank to use the value received from the BGP peer. The route with the lowest value will be preferred.
        - `med_out` - (Optional) The MED will be applied to all routes transmitted on this BGP connection. The neighbouring autonomous system may prefer the lowest value at their discretion.
        - `bfd_enabled` - (Optional) Must be true for BFD configuration to be honoured - default is false.
    - `bfd_configuration` - (Optional) Bidirectional Forwarding Detection. These settings will be used for all BGP connections on this interface where BFD is enabled.
        - `tx_interval` - (Optional) The minimum time between sending BFD packets to the neighbour. The supported range is 300ms to 9000ms.
        - `rx_interval` - (Optional) The minimum time between BFD packets that a neighbour should send. The supported range is 300ms to 9000ms.
        - `multiplier` - (Optional) The BGP session will be torn down if this many consecutive BFD packets are not received from the neighbour.
- `b_end`:
    - `port_id` - (Required) The Port that the VXC B-End is attached to.
    - `requested_vlan` - (Required) The VLAN assign to the B-End port.

## Attributes Reference
- `uid` - The Port identifier.
- `vxc_type` - The VXC type.
- `provisioning_status` - The current provisioning status of the VXC (this status does not refer to availability).
- `create_date` - A Unix timestamp representing the time the VXC was created.
- `created_by` - The user who created the VXC.
- `live_date` - A Unix timestamp representing the date the VXC went live.
- `company_name` - The name of the company that owns the account for the VXC.
- `locked` - Indicates whether the resource has been locked by a user.
- `admin_locked` - Indicates whether the resource has been locked by an administrator.
- `vxc_internal_type` - An internal variable used by Terraform to orchestrate CSP VXCs.
- `a_end`:
    - `owner_uid` - The identifier of the owner of the A-End Port.
    - `name` - The A-End Port name.
    - `location` - The name of the data center where the Port is located.
    - `assigned_vlan` - The VLAN assigned by Megaport to the A-End Port.
- `b_end`:
    - `owner_uid` - The identifier of the owner of the B-End Port.
    - `name` - The B-End Port name.
    - `location` - The name of the data center where the Port is located.
    - `assigned_vlan` - The VLAN assigned by Megaport to the B-End Port.
