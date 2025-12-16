# Cloud Port Lookup Data Source Examples

The `megaport_cloud_port_lookup` data source is an enhanced version of the `megaport_partner` data source that returns **all matching ports** instead of just one. This gives you complete visibility and control over port selection.

## Key Advantages

1. **Returns arrays**: See all available options, not just one
2. **OneOf validation**: Proper validation on connect_type field
3. **Secure port support**: Access GCP, Oracle, and Azure secure ports
4. **Better filtering**: More explicit and predictable filtering
5. **Custom selection**: Use Terraform logic to choose the best port

## Basic Usage Patterns

### 1. Simple Port Lookup

```hcl
data "megaport_cloud_port_lookup" "aws_ports" {
  connect_type = "AWS"
  location_id  = 3
}

# Use the first available port
resource "megaport_vxc" "connection" {
  b_end = {
    requested_product_uid = data.megaport_cloud_port_lookup.aws_ports.ports[0].product_uid
  }
  # ... other configuration
}
```

### 2. Advanced Port Selection

```hcl
data "megaport_cloud_port_lookup" "all_aws_ports" {
  connect_type = "AWS"
  location_id  = 3
}

locals {
  # Select port with lowest rank (best performance)
  best_port = [
    for port in data.megaport_cloud_port_lookup.all_aws_ports.ports :
    port if port.rank == min([for p in data.megaport_cloud_port_lookup.all_aws_ports.ports : p.rank]...)
  ][0]

  # Or select by name pattern
  sydney_ap_port = [
    for port in data.megaport_cloud_port_lookup.all_aws_ports.ports :
    port if can(regex("sydney.*ap-southeast-2", lower(port.product_name)))
  ][0]
}
```

### 3. Diversity Zone Selection

```hcl
data "megaport_cloud_port_lookup" "primary_ports" {
  connect_type   = "AWSHC"
  location_id    = 3
  diversity_zone = "red"
}

data "megaport_cloud_port_lookup" "backup_ports" {
  connect_type   = "AWSHC"
  location_id    = 3
  diversity_zone = "blue"
}

# Create redundant connections
resource "megaport_vxc" "primary_connection" {
  b_end = {
    requested_product_uid = data.megaport_cloud_port_lookup.primary_ports.ports[0].product_uid
  }
  # ... configuration
}

resource "megaport_vxc" "backup_connection" {
  b_end = {
    requested_product_uid = data.megaport_cloud_port_lookup.backup_ports.ports[0].product_uid
  }
  # ... configuration
}
```

### 4. Secure Ports (GCP, Oracle, Azure)

```hcl
# For Google Cloud with pairing key
data "megaport_cloud_port_lookup" "gcp_secure" {
  connect_type   = "GOOGLE"
  include_secure = true
  secure_key     = var.gcp_pairing_key
  location_id    = 3
}

# For Oracle with service key
data "megaport_cloud_port_lookup" "oracle_secure" {
  connect_type   = "ORACLE"
  include_secure = true
  secure_key     = var.oracle_service_key
  location_id    = 3
}
```

### 5. Validation and Error Handling

```hcl
data "megaport_cloud_port_lookup" "aws_ports" {
  connect_type = "AWS"
  location_id  = 3
}

# Validate that we have available ports
locals {
  has_ports = length(data.megaport_cloud_port_lookup.aws_ports.ports) > 0

  selected_port = local.has_ports ? data.megaport_cloud_port_lookup.aws_ports.ports[0] : null
}

# Use check blocks for validation (Terraform 1.5+)
check "ports_available" {
  assert {
    condition = local.has_ports
    error_message = "No AWS ports available in the specified location"
  }
}
```

## Migration from megaport_partner

### Old way (single port, warnings):

```hcl
data "megaport_partner" "aws_port" {
  connect_type = "AWS"
  location_id  = 3
  # Would show warning if multiple matches found
}
```

### New way (array of ports, full control):

```hcl
data "megaport_cloud_port_lookup" "aws_ports" {
  connect_type = "AWS"
  location_id  = 3
}

locals {
  # You choose the selection criteria
  selected_port = data.megaport_cloud_port_lookup.aws_ports.ports[0]
}
```

## Best Practices

1. **Always validate port availability** before using them in resources
2. **Use specific filtering** to reduce the number of matching ports
3. **Consider diversity zones** for redundant connections
4. **Test secure port access** in non-production environments first
5. **Use locals for selection logic** to keep resource definitions clean

## Connect Type Reference

- `AWS`: Amazon Web Services (Private VIF)
- `AWSHC`: AWS Hosted Connection
- `AZURE`: Microsoft Azure ExpressRoute
- `GOOGLE`: Google Cloud Partner Interconnect
- `ORACLE`: Oracle FastConnect
- `IBM`: IBM Cloud
- `OUTSCALE`: Outscale
- `TRANSIT`: Megaport Internet
- `FRANCEIX`: France-IX
