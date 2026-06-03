# NAT Gateway prefix lists are imported using the composite ID
# "<NAT_GATEWAY_PRODUCT_UID>:<PREFIX_LIST_ID>".
#
# Where:
# - NAT_GATEWAY_PRODUCT_UID: UID of the NAT Gateway that owns the prefix list.
# - PREFIX_LIST_ID:          Numeric ID of the prefix list (positive integer).
terraform import megaport_nat_gateway_prefix_list.example "11111111-1111-1111-1111-111111111111:456"
