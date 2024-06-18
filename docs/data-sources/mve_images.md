---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "megaport_mve_images Data Source - terraform-provider-megaport"
subcategory: ""
description: |-
  MVE Images
---

# megaport_mve_images (Data Source)

MVE Images



<!-- schema generated by tfplugindocs -->
## Schema

### Read-Only

- `mve_images` (Attributes List) List of MVE Images (see [below for nested schema](#nestedatt--mve_images))

<a id="nestedatt--mve_images"></a>
### Nested Schema for `mve_images`

Read-Only:

- `id` (String) The ID of the MVE Image. The image id returned indicates the software version and key configuration parameters of the image.
- `product` (String) The product of the MVE Image
- `product_code` (String) The product code of the MVE Image
- `release_image` (Boolean) Indicates whether the MVE image is available for selection when ordering an MVE.
- `vendor` (String) The vendor of the MVE Image
- `vendor_description` (String) The vendor description of the MVE Image
- `version` (String) The version of the MVE Image