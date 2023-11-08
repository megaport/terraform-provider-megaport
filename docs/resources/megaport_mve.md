# Resource Megaport MVE

Creates a Port, which is a physical Port in a Megaport-enabled location. Common connection scenarios include:

 - **MVE-to-Port** - Create a VXC towards a port to connect your overlay network to a physical one.
 - **MVE-to-Cloud** - Configure a cloud onramp by adding a VXC to your MVE.
 - **MVE-to-MVE** - Configure a HA firewall cluster by connecting a pair of MVEs together.

## Example Usage (Simple)
```
data "megaport_location" "datacom" {
  name = "Datacom 6 Orbit"
}

resource "megaport_mve" "mve_datacom" {
  location_id = data.megaport_location.datacom.id
  mve_name    = "Terraform example - MVE"
  image_id    = 32
  vendor      = "PALO_ALTO"
  size        = "SMALL"

  vendor_config = {
    "sshPublicKey"      = var.rsa_pub_key
    "adminPasswordHash" = var.admin_password_hash
  }
}
```

This example results in the creation of a small Palo Alto MVE with a single vNIC located at Datacom in Auckland under a 12 month term.

## Example Usage (Complex)
```
data "megaport_location" "datacom" {
  name = "Datacom 6 Orbit"
}

data "megaport_location" "spark" {
  name = "Spark Mayoral Drive"
}

resource "megaport_mve" "mve_datacom" {
  location_id = data.megaport_location.datacom.id
  mve_name    = "Terraform example - MVE"
  image_id    = 32
  vendor      = "PALO_ALTO"
  size        = "SMALL"

  vendor_config = {
    "sshPublicKey"      = var.rsa_pub_key
    "adminPasswordHash" = var.admin_password_hash
  }

  vnic {
    description = "Internet"
  }
  vnic {
    description = "HA1 - 1/1"
  }
  vnic {
    description = "HA2 - 1/2"
  }
  vnic {
    description = "HA3 - 1/3"
  }
  vnic {
    description = "Data"
  }
}

data "megaport_internet" "akl_blue" {
  metro                    = "Auckland"
  requested_diversity_zone = "blue"
}

resource "megaport_vxc" "transit_blue" {
  vxc_name   = "internet"
  rate_limit = 50

  a_end {
    mve_id         = megaport_mve.mve_datacom.id
    vnic_index     = megaport_mve.mve_datacom.vnic.0.index
    requested_vlan = megaport_mve.mve_datacom.vnic.0.vlan
  }

  b_end {
    port_id = data.megaport_internet.akl_blue.id
  }
}
```

This example creates a small Palo Alto MVE with some additional vNICs ready for a HA scenario. It also configures an Internet VXC on the first vNIC to allow the access for configuration.

## Argument Reference

The following arguments are supported:

 - `mve_name` - (Required) The name of the MVE.
 - `location_id` - (Required) The identifier for the data center where you want to create the MVE.
 - `image_id` - (Required) The image ID to use for the service.
 - `size` - (Required) The name of the image vendor, see the [dev portal](https://dev.megaport.com/#690488c6-0de1-467d-bd75-90dec6cab201) for available sizes.
 - `vendor_config` - (Required) A map containing the image-specific configuration.
 - `term` - (Optional) The contract term for your MVE. The default is month-to-month.
 - `marketplace_visibility` - (Optional) Whether to make this Port public on the [Megaport Marketplace](https://docs.megaport.com/marketplace/).
 - `vnic` - (Optional) A list of vNICs to attach to the MVE.
     - `description` - (Required) The description for the vNIC.
 
 ## Attribute Reference

 In addition to the arguments, the following attributes are exported:
 - `uid` - The Globally Unique Identifier (GUID) for the service.
 - `provisioning_status` - The current provisioning status of the MVE (this status does not refer to availability).
 - `create_date` - A Unix timestamp representing the time the MVE was created.
 - `created_by` - The user who created the MVE.
 - `live_date` - A Unix timestamp representing when the MVE went live.
 - `company_name` - The name of the company that owns the account for the MVE.
 - `locked` - Indicates whether the resource has been locked by a user.
 - `admin_locked` - Indicates whether the resource has been locked by an administrator.
 
## Import

MVEs can be imported using the `uid`, for example:

```
# terraform import megaport_mve.example_mve 03fec669-f2a8-4488-9be9-def6f7998a10
```
