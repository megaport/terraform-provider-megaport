#!/bin/bash

# MCR Prefix Filter Lists can be imported using the format: mcr_uid:prefix_list_id
#
# Where:
# - mcr_uid: The UID of the MCR that owns the prefix filter list
# - prefix_list_id: The numeric ID of the prefix filter list (must be positive integer)
#
# You can find these values using the Megaport API or portal:
# 1. MCR UID: Available in MCR details or from terraform state of megaport_mcr resource
# 2. Prefix List ID: Available in MCR prefix filter list details

# Example 1: Import existing prefix filter list
terraform import megaport_mcr_prefix_filter_list.example "11111111-1111-1111-1111-111111111111:123"

# Example 2: Import multiple prefix filter lists from the same MCR
terraform import megaport_mcr_prefix_filter_list.inbound_filter "11111111-1111-1111-1111-111111111111:456"
terraform import megaport_mcr_prefix_filter_list.outbound_filter "11111111-1111-1111-1111-111111111111:789"

# Example 3: Import with descriptive resource names
terraform import megaport_mcr_prefix_filter_list.ipv4_customer_routes "22222222-2222-2222-2222-222222222222:101"
terraform import megaport_mcr_prefix_filter_list.ipv6_customer_routes "22222222-2222-2222-2222-222222222222:102"

# Common troubleshooting:
# - Ensure the MCR UID is correct and the MCR exists
# - Verify the prefix filter list ID exists on the specified MCR
# - Check that the prefix filter list ID is a positive integer
# - Make sure you have appropriate permissions to access the MCR and its prefix filter lists

# To find existing prefix filter lists:
# 1. Use the Megaport Portal to view MCR details
# 2. Use the Megaport API: GET /mcr/{mcrId}/prefixFilterLists
# 3. Check terraform state if the MCR was created with embedded prefix filter lists

echo "Import format: terraform import megaport_mcr_prefix_filter_list.<resource_name> \"<MCR_UID>:<PREFIX_LIST_ID>\""
echo "Replace <resource_name>, <MCR_UID>, and <PREFIX_LIST_ID> with your actual values"