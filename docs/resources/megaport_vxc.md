# Resource Megaport VXC
Creates a Virtual Cross Connect (VXC), which is a connection between two Megaport services. Common connection scenarios include:

 - **Port-to-Port** - Create a VXC between two Ports to enable multi-data center connectivity.
 - **Port-to-MCR** - Create a VXC between a Port and an MCR to allow the Port to access services attached to the MCR.
 
## Example Usage
```
data megaport_location nextdc_brisbane_1 {
    name = "NextDC B1"
}

data megaport_location nextdc_brisbane_2 {
    name = "NextDC B2"
}

resource megaport_port port_1 {
    port_name       = "Port 1"
    port_speed      = 10000
    location_id     = data.megaport_location.nextdc_brisbane_1.id
}

resource megaport_port port_2 {
    port_name       = "Port 2"
    port_speed      = 10000
    location_id     = data.megaport_location.nextdc_brisbane_2.id
}

resource megaport_vxc vxc {
    vxc_name        = "Terraform Test VXC"
    rate_limit      = 10000

    a_end {
        port_id     = megaport_port.port_1.id
    }

    b_end {
        port_id     = megaport_port.port_2.id
    }
}
```

## Argument Reference
- `vxc_name` - (Required) The name of your VXC.
- `rate_limit` - (Required) The speed of your VXC in Mbps.
- `a_end`
    - `port_id` - (Required) The Port that the VXC A-End is attached to.
    - `requested_vlan` - (Required) The VLAN to assign to the A-End Port.
- `b_end`
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
- `a_end`
    - `owner_uid` - The identifier of the owner of the A-End Port.
    - `name` - The A-End Port name.
    - `location` - The name of the data center where the Port is located.
    - `assigned_vlan` - The VLAN assigned by Megaport to the A-End Port.
- `b_end`
    - `owner_uid` - The identifier of the owner of the B-End Port.
    - `name` - The B-End Port name.
    - `location` - The name of the data center where the Port is located.
    - `assigned_vlan` - The VLAN assigned by Megaport to the B-End Port.
 