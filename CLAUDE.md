# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Development Commands

```bash
# Build the provider
go build -v .

# Install locally (to ~/go/bin)
go install .

# Run unit tests
go test -v -timeout=30m -cover ./internal/provider/

# Run a single test (by function name)
go test -v -timeout=30m -run TestFunctionName ./internal/provider/

# Run acceptance tests (requires API credentials)
TF_ACC=1 MEGAPORT_ACCESS_KEY=xxx MEGAPORT_SECRET_KEY=xxx go test -v -timeout=30m -cover ./internal/provider/

# Lint (uses golangci-lint v2)
golangci-lint run --timeout=10m

# Generate docs and format examples
go generate ./...
```

## Architecture

This is a **Terraform Plugin Framework** provider (not the older SDK/v2). All provider logic lives in `internal/provider/`.

### Resources (prefixed `megaport_`)
| Resource | File | API Entity |
|---|---|---|
| `megaport_port` | `single_port_resource.go` | Physical port |
| `megaport_lag_port` | `lag_port_resource.go` | LAG port |
| `megaport_mcr` | `mcr_resource.go` | Megaport Cloud Router |
| `megaport_mcr_prefix_filter_list` | `mcr_prefix_filter_list_resource.go` | MCR prefix filter list |
| `megaport_mve` | `mve_resource.go` | Megaport Virtual Edge |
| `megaport_vxc` | `vxc_resource.go` | Virtual Cross Connect |
| `megaport_ix` | `ix_resource.go` | Internet Exchange |

### Data Sources
| Data Source | File |
|---|---|
| `megaport_location` | `location_data_source.go` |
| `megaport_partner_port` | `partner_port_data_source.go` |
| `megaport_mve_images` | `mve_image_data_source.go` |
| `megaport_mve_sizes` | `mve_size_data_source.go` |
| `megaport_mcr_prefix_filter_list` | `mcr_prefix_filter_list_data_source.go` |

### Key Supporting Files
- `vxc_schemas.go` — shared partner config schemas (AWS, Azure, Google, Oracle, etc.)
- `vxc_resource_utils.go` / `mcr_resource_utils.go` — resource-specific utilities
- `mcr_prefix_filter_list_schema.go` — prefix filter list schema definitions
- `tflog_slog_handler.go` — bridges megaportgo slog logging to Terraform's tflog

### External API Client
The provider uses `github.com/megaport/megaportgo` (v1.4.9) as its API client library. The `megaportProviderData` struct (in `provider.go`) holds the configured `*megaport.Client` and is passed to all resources/data sources via `Configure()`.

## Key Patterns

### Resource Implementation
Every resource implements `resource.Resource`, `resource.ResourceWithConfigure`, and `resource.ResourceWithImportState`. The standard methods are: `Metadata`, `Schema`, `Configure`, `Create`, `Read`, `Update`, `Delete`, `ImportState`.

Each resource has a model struct with `tfsdk` tags that maps to the schema. Schema attributes use plan modifiers (e.g., `UseStateForUnknown()`) for computed fields and validators from `terraform-plugin-framework-validators`.

### Provisioning Wait
Resources poll for provisioning completion using a configurable `waitForTime` duration (default 10 minutes, set via provider `wait_time` attribute).

### Error Handling
Uses `diag.Diagnostics` for error reporting. HTTP 404 responses in `Read` should remove the resource from state (not error), since the resource was deleted outside Terraform.

### Resource Tags
All resources support a `resource_tags` map attribute. Use `toResourceTagMap()` helper for conversion.

### Testing
Tests use `testify/suite` with `ProviderTestSuite`. Acceptance tests use `testAccProtoV6ProviderFactories` and the `providerConfig` template from `provider_test.go`. Test names follow `TestAccMegaport{Resource}_Basic`. The `RandomTestName()` helper generates prefixed test names (`tf-acc-test-`).

### VXC Partner Configurations
VXC resources support multiple cloud partner types (AWS, Azure, Google, Oracle, Aruba, Fortinet, Versa, etc.), each with unique nested config schemas defined in `vxc_schemas.go`. Partner port UIDs can change via rotation — handle via `requested_product_uid` vs `current_product_uid`.

### MVE Vendor Configs
MVE resources support multiple vendors (Aruba, Aviatrix, Cisco, Fortinet, Palo Alto, SixWind, Versa, VMware) with vendor-specific configuration blocks and up to 5 network interfaces (vNICs).

## Environment Variables
| Variable | Purpose |
|---|---|
| `MEGAPORT_ACCESS_KEY` | API access key |
| `MEGAPORT_SECRET_KEY` | API secret key |
| `MEGAPORT_ENVIRONMENT` | Override provider environment (production/staging/development) |
| `MEGAPORT_ACCEPT_PURCHASE_TERMS` | Override purchase terms acceptance |

## CI Pipeline

CI runs on PRs to `main` (`.github/workflows/test.yaml`):
1. **Build** + **golangci-lint v2.3.1**
2. **Generate** — verifies `go generate ./...` produces no diff
3. **Unit tests** — `go test` with 30min timeout
4. **OpenTofu 1.6.0 compatibility** test

Acceptance tests are currently disabled in CI.
