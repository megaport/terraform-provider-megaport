# Lifecycle Prevent Destroy Example

This example demonstrates how to use Terraform's `lifecycle` block with `prevent_destroy = true` to protect critical Megaport resources from accidental deletion.

## Overview

The `prevent_destroy` lifecycle meta-argument is a Terraform feature that prevents the accidental destruction of important resources. When applied to a resource, Terraform will refuse to destroy that resource unless the `prevent_destroy` flag is first removed from the configuration.

## When to Use Prevent Destroy

Use `prevent_destroy` for resources that:

- **Carry production traffic** - Ports and VXCs that support live applications
- **Are difficult to recreate** - Resources with complex configurations or long setup times
- **Have compliance requirements** - Resources that must remain operational for regulatory reasons
- **Serve as critical infrastructure** - MCRs and ports that other services depend on
- **Are expensive to replace** - Resources with long-term contracts or high setup costs

## How It Works

When you add a `lifecycle` block with `prevent_destroy = true` to a resource:

```terraform
resource "megaport_port" "production_port" {
  product_name = "Production Port"
  port_speed   = 10000
  # ... other configuration ...

  lifecycle {
    prevent_destroy = true
  }
}
```

Terraform will:

1. ✅ Allow creation and updates to the resource
2. ❌ Refuse to destroy the resource via `terraform destroy`
3. ❌ Refuse to destroy the resource when removing it from configuration
4. ❌ Refuse to destroy the resource via targeted destroy commands

## Example Scenarios

### Scenario 1: Protecting a Production Port

```terraform
resource "megaport_port" "production_port" {
  product_name           = "Production Port - Protected"
  port_speed             = 10000
  location_id            = data.megaport_location.bne_nxt1.id
  contract_term_months   = 12

  lifecycle {
    prevent_destroy = true
  }
}
```

**Use Case**: Your production port connects to critical cloud services. Deleting it would cause a major service outage.

### Scenario 2: Protecting an MCR with Multiple VXCs

```terraform
resource "megaport_mcr" "production_mcr" {
  product_name = "Production MCR - Protected"
  location_id  = data.megaport_location.bne_nxt1.id

  router {
    port_speed    = 5000
    requested_asn = 64512
  }

  lifecycle {
    prevent_destroy = true
  }
}
```

**Use Case**: Your MCR routes traffic between multiple cloud providers. It's a critical piece of your network infrastructure that should never be accidentally deleted.

### Scenario 3: Protecting a Critical VXC

```terraform
resource "megaport_vxc" "production_vxc" {
  product_name = "Production VXC - Protected"
  rate_limit   = 1000

  a_end {
    port_id = megaport_port.production_port.id
  }

  b_end {
    port_id = megaport_mcr.production_mcr.id
  }

  lifecycle {
    prevent_destroy = true
  }
}
```

**Use Case**: This VXC carries your production database traffic. Deleting it would immediately break your application.

## Removing Protection

If you need to destroy a protected resource, follow these steps:

### Step 1: Remove the lifecycle block

```terraform
resource "megaport_port" "production_port" {
  product_name = "Production Port"
  port_speed   = 10000
  # ... other configuration ...

  # lifecycle {
  #   prevent_destroy = true
  # }
}
```

### Step 2: Apply the configuration change

```bash
terraform apply
```

This updates the configuration in Terraform's state without making any changes to the actual resource.

### Step 3: Destroy the resource

```bash
# Destroy all resources
terraform destroy

# Or destroy a specific resource
terraform destroy -target=megaport_port.production_port
```

## Best Practices

1. **Apply to All Production Resources**

   - Add `prevent_destroy` to all production ports, MCRs, and VXCs
   - Document why each resource is protected in comments

2. **Use with Portal Locking**

   - Combine Terraform's `prevent_destroy` with Megaport Portal's [service locking feature](https://docs.megaport.com/portal-admin/locking/)
   - This provides defense-in-depth protection

3. **Version Control**

   - Track all changes to lifecycle blocks in Git
   - Require code review for removing `prevent_destroy` flags

4. **Workspace Separation**

   - Use separate Terraform workspaces for production and non-production
   - Only apply `prevent_destroy` to production workspaces

5. **Document Protection Reasons**

   ```terraform
   resource "megaport_port" "prod_port" {
     # ... configuration ...

     # Protected: This port carries production traffic for our main application.
     # Last incident: Accidental deletion on 2024-03-15 caused 2-hour outage.
     # Approved by: Engineering Lead
     lifecycle {
       prevent_destroy = true
     }
   }
   ```

6. **Test in Non-Production First**

   - Test your destroy operations in dev/staging environments
   - Verify that non-protected resources can be destroyed successfully

7. **Regular Audits**
   - Periodically review which resources have `prevent_destroy`
   - Remove protection from decommissioned resources

## Relationship with Safe Delete

The `prevent_destroy` lifecycle block is a Terraform-side protection, while **Safe Delete** is a Megaport Provider feature that prevents deletion of resources with active dependencies.

| Feature           | Protection Level        | Use Case                                                                         |
| ----------------- | ----------------------- | -------------------------------------------------------------------------------- |
| `prevent_destroy` | Terraform configuration | Protect critical resources from any deletion attempt                             |
| Safe Delete       | API/Provider level      | Automatic protection - prevents deleting ports/MCRs/MVEs that have attached VXCs |

These features complement each other:

- **Safe Delete** is always enabled and prevents accidental deletion of resources with dependencies
- **prevent_destroy** is optional and prevents deletion of specific resources you designate as critical

See the main README for more information about Safe Delete functionality.

## Error Messages

### When you try to destroy a protected resource:

```
Error: Instance cannot be destroyed

  on prevent_destroy.tf line 10:
  10: resource "megaport_port" "production_port" {

Resource megaport_port.production_port has lifecycle.prevent_destroy set,
but the plan calls for this resource to be destroyed. To avoid this error,
either disable lifecycle.prevent_destroy or reduce the scope of this plan using
the -target flag.
```

**Solution**: Remove the `prevent_destroy` flag from the lifecycle block, apply the configuration, then retry the destroy operation.

## Additional Resources

- [Terraform Lifecycle Meta-Arguments](https://developer.hashicorp.com/terraform/language/meta-arguments/lifecycle)
- [Manage Resource Lifecycle](https://developer.hashicorp.com/terraform/tutorials/state/resource-lifecycle)
- [Megaport Documentation - Terraform State Management](https://docs.megaport.com/terraform/terraform-state-management/?h=#understanding-terraform-state-for-megaport-resources)
- [Megaport Documentation - Service Locking](https://docs.megaport.com/portal-admin/locking/)

## Running This Example

1. Update the provider configuration with your credentials
2. Review the resources and their protection settings
3. Run `terraform init` to initialize the provider
4. Run `terraform plan` to see what will be created
5. Run `terraform apply` to create the resources
6. Try `terraform destroy` and observe that protected resources cannot be destroyed
7. Remove the lifecycle blocks from non-production resources and retry destroy
