terraform {
  required_providers {
    megaport = {
      source  = "megaport/megaport"
      version = "~> 1.3"
    }
  }
}

provider "megaport" {
  environment           = "staging"
  access_key            = "access_key"
  secret_key            = "secret_key"
  accept_purchase_terms = true
}

# Lookup location for port deployment
data "megaport_location" "bne_nxt1" {
  name = "NextDC B1"
}

# ==================================================================================
# Example 1: Production Port with Lifecycle Protection
# ==================================================================================
# This port is protected from accidental deletion using the lifecycle block.
# If you attempt to destroy this resource, Terraform will fail with an error.
# This is ideal for production resources that should never be accidentally deleted.

resource "megaport_port" "production_port" {
  product_name           = "Production Port - Protected"
  port_speed             = 10000
  location_id            = data.megaport_location.bne_nxt1.id
  contract_term_months   = 12
  marketplace_visibility = false
  cost_centre            = "Production Infrastructure"

  # Lifecycle block to prevent accidental destruction
  lifecycle {
    prevent_destroy = true
  }
}

# ==================================================================================
# Example 2: Production MCR with Lifecycle Protection
# ==================================================================================
# Similar to the port above, this MCR is protected from deletion.
# MCRs often serve as critical routing infrastructure and should be protected.

resource "megaport_mcr" "production_mcr" {
  product_name         = "Production MCR - Protected"
  location_id          = data.megaport_location.bne_nxt1.id
  contract_term_months = 12
  port_speed           = 1000
  cost_centre          = "Production Infrastructure"

  # Lifecycle block to prevent accidental destruction
  lifecycle {
    prevent_destroy = true
  }
}

# ==================================================================================
# Example 3: Critical VXC with Lifecycle Protection
# ==================================================================================
# This VXC connects production services and is protected from deletion.
# VXCs that carry production traffic should be protected to prevent service disruption.

resource "megaport_vxc" "production_vxc" {
  product_name         = "Production VXC - Protected"
  rate_limit           = 1000
  contract_term_months = 12
  cost_centre          = "Production Infrastructure"

  a_end {
    requested_product_uid = megaport_port.production_port.product_uid
  }

  b_end {
    requested_product_uid = megaport_mcr.production_mcr.product_uid
  }

  # Lifecycle block to prevent accidental destruction
  lifecycle {
    prevent_destroy = true
  }
}

# ==================================================================================
# Example 4: Non-Production Resources Without Protection
# ==================================================================================
# These resources don't have lifecycle protection and can be destroyed normally.
# Use this pattern for development, testing, or non-critical resources.

resource "megaport_port" "dev_port" {
  product_name           = "Dev Port - Not Protected"
  port_speed             = 1000
  location_id            = data.megaport_location.bne_nxt1.id
  contract_term_months   = 1
  marketplace_visibility = false
  cost_centre            = "Development"

  # No lifecycle block - this resource can be destroyed normally
}

# ==================================================================================
# Removing Lifecycle Protection
# ==================================================================================
# If you need to destroy a protected resource:
#
# Option 1: Remove the lifecycle block from the resource configuration,
#           then run `terraform apply` to update the configuration.
#           After that, you can run `terraform destroy`.
#
# Option 2: Use the -refresh-only flag to update state without changes:
#           terraform plan -refresh-only
#           Then manually remove the lifecycle block and destroy.
#
# Option 3: Use targeted destroy (use with caution):
#           terraform destroy -target=megaport_port.production_port
#           This will still respect the prevent_destroy flag.
#
# Note: You must remove the prevent_destroy flag before destroying.
# ==================================================================================

# ==================================================================================
# Additional Best Practices
# ==================================================================================
# 1. Use prevent_destroy on all production resources
# 2. Document why resources are protected in comments
# 3. Combine with Megaport Portal locking for extra protection
# 4. Use version control to track changes to lifecycle blocks
# 5. Consider using workspaces to separate production from non-production
# 6. Test destroy operations in non-production environments first
# ==================================================================================
