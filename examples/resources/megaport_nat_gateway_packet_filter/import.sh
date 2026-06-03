# NAT Gateway packet filters are imported using the composite ID
# "<NAT_GATEWAY_PRODUCT_UID>:<PACKET_FILTER_ID>".
#
# Where:
# - NAT_GATEWAY_PRODUCT_UID: UID of the NAT Gateway that owns the packet filter.
# - PACKET_FILTER_ID:        Numeric ID of the packet filter (positive integer).
terraform import megaport_nat_gateway_packet_filter.example "11111111-1111-1111-1111-111111111111:123"
