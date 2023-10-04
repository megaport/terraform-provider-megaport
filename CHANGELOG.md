# 0.3.0-beta (Oct 4, 2023)

## Breaking Changes
  * Provider: `megaport`
    * X-Auth tokens are being deprecated from October 2023, authentication has been updated to use API keys.
    * `username`, `password` and `mfa_otp_key` attributes in provider configuration should be replaced with `access_key` and `secret_key` attributes.
    * Go to Tools > API Key Generator in Megaport Portal to manage Active API Keys.
    * See documentation and example updates for further information.

# 0.2.10-beta (July 28, 2023)

## Changes
  * Feature: Support for MCR contract term.
    * Resource: `megaport_mcr`
      * MCR contract `term` can optionally be set to 1, 12, 24, or 36 months when the MCR is created. Default is month-to-month (1).
  * Feature: Oracle partner support and OCI VXC connectivity. Credit @aszynkow
    * Resource: `megaport_oci_connection`
      * Megaport MCR can now connect to OCI FastConnect via Megaport VXC. See Documentation for examples.
  * Documentation and example updates.

# 0.2.9-beta (Apr 5, 2023)

## Changes
  * Feature: Support for deleting AWS Hosted Connection (dxcon) at destroy time.
    * Resource: `megaport_aws_connection`
      * If `destroy_connection` is set to `true` the Connection will be deleted at destroy time. An external AWS session is required and the Connection cannot be deleted if it still has Virtual Interfaces attached.

# 0.2.8-beta (Feb 20, 2023)

## Changes
  * Feature: Support for filtering Partner Megaports by Diversity Zone.
    * Data Source: `megaport_partner_port`
      * Partner Megaports can now be filtered by `diversity_zone` attribute.
    * Documentation and example updates.

# 0.2.7-beta (Dec 8, 2022)

## Changes
  * Feature: Support for creating Prefix Filter Lists on MCR's.
    * Prefix Filter Lists (multiple allowed) can now be created for MCR resources.
  * Feature: Support for applying BGP Peer Filters and BGP Prefix Filters on VXC's.
    * BGP Peer Filter and BGP Prefix Filter configuration can now be applied within the `a_end_mcr_configuration` `bgp_connection` block for VXC resources.
  * Documentation and example updates.

# 0.2.6-beta (Nov 2, 2022)

## Changes
  * Feature: Support for manually supplying peering values for Azure VXC's.
    * Resource: `megaport_azure_connection`
      * Azure peering values can now be manually supplied via `private_peering` and `microsoft_peering` blocks.
      * Added `auto_create_private_peering` and `auto_create_microsoft_peering` boolean options.
    * Documentation and example updates.

## Breaking Changes
  * Resource: `megaport_azure_connection`
    * Replaced `peerings` block within `csp_settings` with auto-create boolean options.

# 0.2.5-beta (May 5, 2022)

## Changes
  * Refactor: Import MarshallMcrAEndConfig from megaport/megaportgo. Credit @daniel-noland

# 0.2.4-beta (April 6, 2022)

## Changes
  * Feature: Optional connection name attribute for AWS connections. Credit @ngarratt
    * Added optional `connection_name` into `megaport_aws_connection` resource
  * Feature: Import support for pre-existing AWS connections. Credit @ngarratt
    * Documentation:
      * Added import section for `megaport_aws_connection` resource

# 0.2.3-beta (March 24, 2022)

## Changes
  * Feature: Static routes for all VXC Connections with MCR A End
    * `a_end_mcr_configuration` on all VXC resources can now accept `ip_route` configurations
  * Bugfixes:
    * Successfully handle and pass resource delete failures to terraform

## 0.2.2-beta (March 2, 2022)

## Changes
  * Bugfixes:
    * VXC to cloud resources will now properly autoconfigure BGP connection if no manual configuration is supplied.

## 0.2.1-beta (February 22, 2022)

## Changes
  * Feature: NAT support for all VXC Connections with MCR A End
    * Resource: `megaport_vxc_connection`
      * Added optional `nat_ip_addresses` configuration
    * Resource: `megaport_aws_connection`
      * Added optional `nat_ip_addresses` configuration
    * Resource: `megaport_azure_connection`
      * Added optional `nat_ip_addresses` configuration
    * Resource: `megaport_gcp_connection`
      * Added optional `nat_ip_addresses` configuration
    * Documentation and example updates

## 0.2.0-beta (January 27, 2022)

## Changes
  * Feature: BGP Connection support for all VXC Connections with MCR A End
    * Resource: `megaport_vxc_connection`
      * Added optional `a_end_mcr_configuration` configuration
    * Resource: `megaport_aws_connection`
      * Added optional `a_end_mcr_configuration` configuration
    * Resource: `megaport_azure_connection`
      * Added optional `a_end_mcr_configuration` configuration
    * Resource: `megaport_gcp_connection`
      * Added optional `a_end_mcr_configuration` configuration
    * Documentation and example updates

## Breaking Changes
  * Resource: `megaport_aws_connection`
    * Removed `megaport_aws_connection.a_end.partner_configuration` in lieu of the `megaport_aws_connection.a_end_mcr_configuration` block added to all VXC resources.
    * Moved `csp_settings.attached_to` to `a_end.port_id` to bring VXC resources in line with each other.
  * Resource: `megaport_azure_connection`
    * Moved `csp_settings.attached_to` to `a_end.port_id` to bring VXC resources in line with each other.
  * Resource: `megaport_gcp_connection`
    * Moved `csp_settings.attached_to` to `a_end.port_id` to bring VXC resources in line with each other.

## 0.1.10-beta (November 5, 2021)

Notes
  * Feature: BGP Connection support for AWS VXC Connections
    * Added `partner_configuration` into `megaport_aws_connection` resource.
    * Documentation and example updates

## 0.1.9-beta (August 19, 2021)

Notes
 * Fix marshalling issue with VirtualRouter in VXCResource.

## 0.1.8-beta (June 19, 2021)
Notes

* Enable Google Partner port location selection. Credit @kdw174


## 0.1.7-beta (June 4, 2021)
Notes

* Add support for credentials as Environment Variables. Credit @angryninja48

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
