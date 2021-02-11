## 0.1.6-beta (February 11, 2021)

Notes

* Fix: MCR import now works.
* Documentation:
  * Removed import sections from `megaport_aws_connection`.
  * Removed import sections from `megaport_azure_connection`.  
  * Removed import sections from `megaport_gcp_connection`.
  * Removed import sections from `megaport_vxc_connection`.

## 0.1.5-beta (February 10, 2021)

Notes

* Fix: Resources can now be imported.

## 0.1.4-beta (January 12, 2021)

Notes

* Updated `megaport-go` library.
    * Changed the WaitForPortProvisioning function so that it 
      considers "LIVE" or "CONFIGURED" as an active status.
* Fix: Ports will create correctly at the CONFIGURED stage.

## 0.1.3-beta (December 22, 2020)

Notes

* Documentation and example updates (no functionality changes)

## 0.1.2-beta (December 10, 2020)

Notes:

* Documentation and example updates (no functionality changes).

DOCUMENTATION UPDATES:

* Documentation updates (links, etc)
* Reformat Documentation for Terraform Provider Registry
* Added `requested_vlan` into `full_ecosystem` and `two_ports_and_vxc` examples.
* Added `peerings` block into `mcr_and_csp_vxcs` example.

## 0.1.0-beta (December 1, 2020)

Notes:  

* Initial Beta release

FEATURES:

* **New Data-Source:** `megaport_partner_port` Lookup Partner Ports from the Megaport Marketplace
* **New Data-Source:** `megaport_location` Lookup Megaport Location ID's
* **New Data-Source:** `megaport_port` Lookup existing Port details
* **New Data-Source:** `megaport_mcr` Lookup existing MCR (Megaport Cloud Router) details 
* **New Data-Source:** `megaport_vxc` Lookup existing VCX (Virtual Cross Connect) details
* **New Data-Source:** `megaport_aws_connection` Lookup existing AWS CSP connection details
* **New Data-Source:** `megaport_azure_connection` Lookup existing Azure CSP connection details
* **New Data-Source:** `megaport_gcp_connection` Lookup existing GCP CSP connection details
* **New Resource:** `megaport_port` Create a Megaport
* **New Resource:** `megaport_mcr` Create a MCR
* **New Resource:** `megaport_vxc` Create a VCX
* **New Resource:** `megaport_aws_connection` Create an AWS Connection
* **New Resource:** `megaport_azure_connection` Create an Azure Connection
* **New Resource:** `megaport_gcp_connection` Create a GCP Connection
