---
page_title: "Upgrading to v2"
description: |-
  Guide to upgrading from v1.x to v2.0 of the Megaport Terraform provider
---

# Upgrading to v2

Version 2.0 of the Megaport Terraform provider removes deprecated read-only attributes from the `megaport_port`, `megaport_lag_port`, `megaport_mcr`, `megaport_mve`, and `megaport_vxc` resources, and replaces the inline MCR `prefix_filter_lists` attribute with the standalone `megaport_mcr_prefix_filter_list` resource.

State is migrated automatically. When you run your first plan with v2, the provider silently drops the removed attributes from state. No `terraform state` commands or manual state edits are needed.

## Before you upgrade

If your MCR configurations use the inline `prefix_filter_lists` attribute, leave it in place until after the upgrade. In particular, do **not** set `prefix_filter_lists = []` while still on v1: on v1 an explicit empty list deletes every prefix filter list on the MCR via the API.

## Upgrade steps

1. Update the provider version constraint and re-initialize:

   ```hcl
   terraform {
     required_providers {
       megaport = {
         source  = "megaport/megaport"
         version = "~> 2.0"
       }
     }
   }
   ```

   ```shell
   terraform init -upgrade
   ```

2. Run `terraform plan`. Any reference to a removed attribute (in an output, local, or expression) fails validation with an `Unsupported attribute` error. Remove those references.

3. If you use MCR prefix filter lists, follow the section below.

## Removed read-only attributes

The following attributes were computed-only, so they never appeared as arguments in configuration. If your configuration references one (for example `megaport_mcr.example.provisioning_status` in an output), remove the reference. The data is still available from the Megaport API and Portal.

- **`megaport_port` and `megaport_lag_port`:** `last_updated`, `product_id`, `provisioning_status`, `create_date`, `created_by`, `terminate_date`, `live_date`, `market`, `usage_algorithm`, `contract_start_date`, `contract_end_date`, `vxc_permitted`, `vxc_auto_approval`, `virtual`, `locked`, `cancelable`
- **`megaport_mcr`:** all of the above plus `product_type`, `secondary_name`, `lag_primary`, `lag_id`, `aggregation_id`, `company_name`, `buyout_port`, `admin_locked`
- **`megaport_mve`:** the `megaport_port` set plus `product_type`, `secondary_name`, `company_name`, `buyout_port`, `admin_locked`, and the `vlan` attribute on `vnics`
- **`megaport_vxc`:** `last_updated`, `product_id`, `product_type`, `provisioning_status`, `secondary_name`, `usage_algorithm`, `created_by`, `live_date`, `create_date`, `contract_start_date`, `contract_end_date`, `company_name`, `locked`, `admin_locked`, `cancelable`

## MCR prefix filter lists

The inline `prefix_filter_lists` attribute on `megaport_mcr` has been removed. Prefix filter lists are now managed exclusively with the standalone `megaport_mcr_prefix_filter_list` resource.

Upgrading does not touch your existing prefix filter lists. They remain configured on the MCR but are unmanaged by Terraform until you import them:

1. Upgrade to v2 and remove `prefix_filter_lists` from your `megaport_mcr` configuration.
2. Add a `megaport_mcr_prefix_filter_list` resource and an `import` block (Terraform 1.5 or later) for each existing list. The import ID format is `mcr_uid:prefix_list_id`:

   ```hcl
   import {
     to = megaport_mcr_prefix_filter_list.example
     id = "11111111-1111-1111-1111-111111111111:123"
   }

   resource "megaport_mcr_prefix_filter_list" "example" {
     mcr_id         = megaport_mcr.example.product_uid
     description    = "Allow private IPv4 networks"
     address_family = "IPv4"

     entries = [
       {
         action = "permit"
         prefix = "10.0.0.0/8"
       },
     ]
   }
   ```

3. Run `terraform plan` and review the planned imports. Match each resource's arguments to the existing list so the plan shows imports with no accompanying updates, then apply. The `import` blocks can be removed once the apply succeeds.

On Terraform versions without `import` block support, use the CLI instead:

```shell
terraform import megaport_mcr_prefix_filter_list.example "11111111-1111-1111-1111-111111111111:123"
```

Importing preserves the prefix filter list IDs, so any BGP connection filters on your VXCs that reference those IDs keep working. See the `megaport_mcr_prefix_filter_list` resource documentation for details on the import format and finding list IDs.

If you prefer the lists never to be unmanaged, you can run the same import while still on v1: the resource and import ID format are identical there. Add `lifecycle { ignore_changes = [prefix_filter_lists] }` to the `megaport_mcr` resource first so the inline attribute does not fight the standalone resources, import, then upgrade and remove the inline attribute and the `ignore_changes` entry together (v2 rejects `ignore_changes` entries for attributes that no longer exist).
