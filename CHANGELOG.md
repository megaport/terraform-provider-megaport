# Changelog

All notable changes to the Megaport Terraform Provider will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased] — V2

### Breaking Changes

- Module path changed to `github.com/megaport/terraform-provider-megaport/v2`
- Removed read-only metadata fields from all resources (`product_id`, `provisioning_status`, `usage_algorithm`, `virtual`, `locked`, `cancelable`, `vxc_permitted`, `vxc_auto_approval`, `live_date`, `create_date`, `created_by`, `market`, `terminate_date`, `contract_start_date`, `contract_end_date`)
- VXC partner configs moved inside `a_end_config`/`b_end_config` blocks
- MVE `vendor_config` replaced with per-vendor blocks (`aruba_config`, `cisco_config`, etc.)
- MCR inline `prefix_filter_lists` removed — use `megaport_mcr_prefix_filter_list` resource
- Removed `last_updated` field from all resources
- Removed `ordered_vlan` from VXC — use `vlan` only
- Date formats standardized to RFC 3339
- VXC end config UIDs renamed: `requested_product_uid` → `product_uid`, `current_product_uid` → `assigned_product_uid`

### Added

- `ResourceWithMoveState` for automatic V1 → V2 state migration
- Per-resource configurable timeouts via `timeouts` block
- Data source: `megaport_vxc_csp_connection`
- Shared retry/backoff utilities with exponential backoff and jitter
- Enriched API error messages with HTTP status and trace ID
- Unit tests for `fromAPI` mapping functions across all resources
- `GNUmakefile` with standard build targets

### Fixed

- IX resource now respects provider-level `wait_time` setting
- MCR rate limiter goroutine leak in prefix filter list operations
- Silent diagnostic swallowing in `fromAPI` methods
- Global `waitForTime` variable thread-safety issue (moved to per-resource field)

### Changed

- Retry strategies standardized across all resources (exponential backoff + jitter)
- Shared `configureMegaportResource` helper extracted for all resources
- Port resource helpers extracted into `port_resource_utils.go`

---

## Release History

### [v1.6.0] — 2026-04-03

- fix: remove unused data source filter utilities
- fix: simplify MCR data source to only filter by product_uid
- fix: remove redundant uids attribute from MCR data source
- fix: remove product_id, product_type, and virtual from MCR data source
- fix: have data source test suites for mcr run in parallel
- fix: test logic
- fix: make much more robust data source
- fix: add data source test coverage
- feat: add megaport_mcrs data source
- fix: fix transit VXC update assigning partner config to wrong end and correct error message

### [v1.5.2] — 2026-04-01

- fix: detect partner config changes when state is null during VXC update

### [v1.5.1] — 2026-03-23

- fix: replace volatile string-based partner port filters with stable connect_type + location_id lookups and use separate GCP pairing keys per test
- fix: change safe delete test location again
- fix: change safe delete test to use new diversity zone
- fix; mve locations
- fix: redefine diversity zones for mves that need it
- fix: vnic index handling, location splitting for mve
- fix: location IDs, mve image ID, and gcp/azure keys for testing
- fix: replace noisy CSP port rotation warnings with debug-level tflog and fix B-End comparison logic
- docs: update terraform mcp server documentation
- fix: Retry prefix filter list delete on 409 to handle async VXC deprovision race
- fix: add depends_on to both test VXC resources so Terraform destroys the VXC (and its BGP connection) before attempting to delete the prefix filter lists
- fix: MCR Read/Update to skip prefix filter list API fetch when using standalone resource

### [v1.5.0] — 2026-03-02

- fix: remove wait logic from create
- fix: poll API for vnic_index propagation after create/update, preserve from state on read
- docs: clarify computed field behavior during import to prevent confusion about configuration drift
- fix: reuse createVrouterPartnerConfig for B-End VXC creation to fix missing ip_mtu and eliminate code duplication (#319)
- fix: apply ip_mtu to B-End vrouter partner config and add MCR-to-MCR acceptance test (#319)
- refactor: use normalizeCIDR in validator to remove duplicated CIDR parsing logic
- fix: reject CIDR prefixes with host bits set and return actionable error with correct network address (#317)
- fix: address PR review - guard null ge/le in exact match check, clarify CIDR normalization docs
- fix: resolve lint errors - remove unnecessary nil check and extra blank line
- fix: preserve user CIDR prefix in state and normalize on API calls to prevent drift with non-canonical prefixes (#317)
- fix: improve timeout handling
- fix: poll API for vnic_index propagation instead of preserving stale values from state
- docs: improve documentation for inner vlan for azure vxcs
- fix: preserve VXC vnic_index across reads and updates to prevent drift, add import and update drift tests
- fix: preserve VXC user-only fields after import to prevent infinite update drift, update gitignore
- fix(mve): fix mve_sizes data source bugs and improve MVE size documentation

### [v1.4.7] — 2026-01-15

- fix: use prefix-based lookup for normalization to handle entry reordering
- fix: add prefix validation before normalizing to guard against entry reordering
- fix: rename test cases to clarify they return raw API values
- fix: update comment to accurately describe import behavior
- fix: gofmt formatting
- fix: normalize exact match prefix filter entries using plan comparison
- fix: normalize exact match prefix filters to prevent inconsistent state errors
- docs: remove embedded prefix_filter_lists from MCR examples to reflect standalone resource
- fix: remove unused time param in waitForProvisioningStatus
- fix: remove unused param for expected status
- fix: add helper to check for provisioning status for resource tests that change contract term months, remove plan modifier in mve for contract end date to avoid drift errors from api changing value, wait for resources in tests for contract term to reach provisioning status of LIVE
- feat: add contract term update tests for all resources
- fix: change location name to id for global switch
- docs: add 6WIND MVE examples and clarify RSA 2048-bit SSH key requirement

### [v1.4.6] — 2025-11-05

- fix: adds CODEOWNERS

### [v1.4.5] — 2025-10-31

- fix: change field to use ID in vxc test because of location rename
- docs/location-id-improvement
- fix(vxc): add null checks for inner_vlan, improve update verification, and enhance error handling
- fix: add staging env, add link in readme file
- fix: use staging environment for prompts
- fix: variables
- fix: change location to nextdc b1
- fix: contract term months and marketplace visibility
- fix: make sure you trigger mcp server
- fix docs output
- fix: field names and newlines
- fix: prompts
- fix: provider details
- docs: formatting
- docs: mcp server tutorial
- fix: add update timeout and verification to account for api delayed propagation
- fix: refine inner vlan state change to only untag on updates

### [v1.4.4] — 2025-10-23

- docs: specify SHA-256 crypt format for Palo Alto MVE admin_password_hash field
- cleanup: Address PR feedback: remove redundant cost_centre, add case-insensitive validator, clarify lifecycle rules, remove deprecation timeline
- fix: Convert location data sources in examples to use IDs instead of names for stability. Remove tests depending on name
- feat: support 400g port
- feat: support 48 and 60 month contract terms on port, mve, mcr, and vxc
- fix: terraform state management link
- fix: docs
- fix: example syntax for fields
- docs: preventDestroy and safeDelete
- fix: remove validation function to resource file for pfl entry
- fix: mcr docs
- feat: mcr prefix filter list standalone resource

### [v1.4.3] — 2025-09-18

- fix: cost centre for vxc update
- fix: index template
- fix: bump megaportgo version to 1.4.2
- fix: vxc cost centre update
- fix: always use planned cost_centre value in Update functions
- fix: remove generation file
- fix: change to use ids for locations
- fix: typo
- docs: fortinet mve transit vxc example

### [v1.4.2] — 2025-09-08

- fix: allow user to manually input product UID for partner port with azure/gcp/oracle vxc, fix location name in data source test
- docs: update docs to use full readme in template
- fix: naming
- fix: transit vxc example
- docs: service key example
- fix: make strings equal fold for product type mve checks
- fix: do not allow vnic index fields to be included for port/mve product in vxc update
- fix: prevent VLAN update attempts for AWS and Transit connections
- fix(vxc): improve VNIC index handling during VXC updates
- fix: only send Update API call if there are changed fields in the update request.
- fix: only add Product UID to Update call if the end connection requested UID is different from that of the state. this prevents unnecessary move attempts in the api.
- fix: only add VNIC Index to Update call if end connection product type is MVE

### [v1.4.1] — 2025-08-12

- fix: bump megaportgo version, make vendor config fields more flexible

### [v1.4.0] — 2025-08-04

- fix: correct lint action version
- fix: yaml syntax
- fix: switch version to 8 for linter
- fix: golangci lint version
- fix: github action for linter
- fix: code cleanup
- fix: linter
- fix: docs accuracy
- docs: generate docs
- fix: remove unused variables
- fix: migrate locations data source to use V3 API methods

### [v1.3.9] — 2025-07-24

- test: add acceptance test for ignore lifecycle change for vendor config
- fix: handle null vendor_config during import with lifecycle.ignore_changes
- fix: change language regarding cancelling status
- docs: clarify Terraform state removal timing during resource cancellation
- cleanup: provider naming
- fix: add mve to providerData
- feat: support cancellation at end of term rather than immediately for port and lag ports
- fix: bump mp tf version in provider file
- docs: add info in README and data source documentation
- docs: fix partner port documentation
- docs: improve location documentation
- fix: add mve label to tests, change test cases to use small mve size
- fix: add provider tf to mcr cloud e2e example tf
- cleanup: remove ipMtu from deprecated vxcPartnerConfigAEnd
- docs: generate docs
- feat: support ipMtu in vrouter/aend partner config
- docs: generate docs
- docs/mcr-cloud-end-to-end-examples
- docs: clarify IP address format for BGP fields in megaport_vxc resource
- cleanup: add windows 386 to ignore
- feat(release): Automate release notes generation from PRs
- fix: add check for b end product uid being provided before warning about service key uid

### [v1.3.8] — 2025-06-24

- fix: warn that the requested B-End Product UID is being overridden by the Service Key lookup if product uids dont match
- fix: use product uid from service key so user does not need to use both for service key vxcs
- fix: check for included vnic index in updates to mve vxcs, otherwise keep vnic index same on end config

### [v1.3.7] — 2025-06-04

- fix: ensure consistent representation of empty MCR prefix filter list
- fix: resolve merge conflict
- fix: remove import state test
- feat: add acceptance test for moving mve to mve vxc and changing vnic index
- fix: add back in mixed provider test suite
- fix: prevent user from auto-assigning inner_vlan as 0, update tests
- fix: inner vlan 0 from api logic
- fix: implement tests for inner vlan changes, update documentation regarding updating to inner vlan 0, add checks in update method to prevent change to 0 autoassign inner vlan
- feat: add acceptance test for inner vlan
- fix: properly handle inner VLAN auto-assignment (0) and untagged (-1) values to prevent inconsistency errors
- fix: disable action built in caching for linter
- fix: explicitly set go version
- fix: remove cache key
- fix: use custom cache key to fix cache issue
- fix: attempt to fix cache issue in github action
- fix: add retry logic to linter, specify version
- fix: change oracle virtual circuit id for test
- fix: oracle csp connection, add acceptance test
- fix: language about open tofu compatibility
- fix: Add acceptance test for safe delete prevention with attached VXCs
- feat: add safeDelete field to port, mcr, mve
- fix: make asn require replacement if modified in mcr resource
- fix: add computed diversity zone field to mcr based on api assignment
- fix: support updating vnic index in vxc
- docs: add statement about open tofu compatability and installation
- fix: add opentofu local plugin directory compatibility to ci workflow
- fix: fix opentofu test by using project-local plugin directory with init -plugin-dir
- fix: fix opentofu ci to use protocol v6 binary and plugin path for local provider
- fix: use opentofu dev_overrides for local provider in ci test
- fix: fix opentofu ci to use correct plugin path and binary naming
- fix: have opentofu ci use local provider build via dev plugin path for accurate compatibility testing
- fix: make sure provider is locally built instead of from registry
- fix: build provider in repo root and copy to test directory
- fix: indentation syntax
- fix: unzip command for installing opentofu
- feat: ci testing for opentofu-compatability

### [v1.3.6] — 2025-04-29

- fix: bump x/net to 0.38
- fix: PRODUCT_MVE
- fix: conditional for mve
- fix: require vnic index for mve vxcs
- fix: change azure and google partner keys, define at top of test file
- fix: add check for aws type if vif
- fix: resolve merge conflict
- fix: fix inconsistent Palo Alto name format in MVE image data source - normalize to api spec of PALO_ALTO
- fix: bump crypto version to 0.35
- cleanup: remove duplicate code conditional
- fix: add computed vlan field for mve vnics
- fix: change validation for aws partner config, move partner config schemas into reusable schema files

### [v1.3.5] — 2025-04-15

- feat: support new mcr port speeds

### [v1.3.4] — 2025-04-10

- fix: schema for cloud init for aviatrix edge, use fake creds
- fix: improve clarity of warning message and documentation regarding requested_product_uid involving partner csp ports
- docs: improved documentation and examples for aviatrix mve with description of base64 cloud init
- cleanup: file organization for readability
- feat: Add Internet Exchange (IX) resource

### [v1.3.3] — 2025-03-12

- fix: properly parse lag count from API response

### [v1.3.2] — 2025-03-06

- feat: Update LAG port resource to handle LagPortUIDs and LagCount correctly

### [v1.3.1] — 2025-02-25

- fix: remove repeated code and bump timeout to 10m
- fix: remove deprecated linters
- fix: update golangci.yml
- fix: add enabled linters go golangci yml in root
- fix: file extension for golangci
- fix: re insert removed comment
- fix: linter ci code
- fix: make localAsn a pointer in megaportgo
- fix: add local asn to bgp connection config in vxc resource

### [v1.3.0] — 2025-02-07

- fix: remove excess muxes and wait groups
- fix: remove select
- fix: remove GetToken logic and just use channel blocking
- fix: change intervals
- fix: remove failure to get token error
- fix: wait rather than returning error for token
- fix: port back in the create logic in update method for new pfls
- fix: tweak rate limiter slightly, add unit testing
- fix: cleanup modify logic with create calls for prefix filter lists there so none are missing on update
- fix: add package level type for rate limiter with factory func that allows you to pass in burst amount and refill speed
- fix: add rate limit logic for reading/updating/deleting mcr prefix filter lists in provider
- feat: add location filtering by ID and update docs/tests to reflect such
- docs: location clarification and API endpoint

### [v1.2.9] — 2025-02-04

- docs: add language about month to month contract term for contract_term_months
- fix: remove vlan check
- fix: correction in tests
- feat: support for resource tagging

### [v1.2.8] — 2025-01-29

- fix: change name for mel-mdc data center location
- fix: change name for mel-mdc data center location
- fix: position of check for equality between partner configs
- fix: detect change in partner config, remove repeated code

### [v1.2.7] — 2025-01-15

- fix: add support for development environment in terraform
- fix: change tests to only ignore user provided end config values
- fix: generation
- feat: multicloud example scenarios
- fix: bring back modify plan logicfor imports

### [v1.2.6] — 2024-12-12

- chore(deps): bump golang.org/x/crypto from 0.21.0 to 0.31.0
- fix: add missing ibm partner config in update calls
- fix: csp connections model
- fix: add support for ibm in update partner config for vxc
- fix: attrs for csp connection
- fix: add csp connection fields
- fix: add ibm to valid partner configs
- feat: ibm partner config
- fix: add test for null vlan
- fix: change state of ordered vlan when updating to prevent unintended null ordered vlan
- fix: make lag count require replace

### [v1.2.5] — 2024-12-02

- fix: set vlan to computed for end config, prevent user from inputting it
- fix: misspelling of cost_centre :)
- fix: add test for untag a-end and b-end vlan
- fix: remove vlan test from basic xc
- fix: update docs and tests
- fix: only untag a-end owned vlan for aws vxc and do untag test for port vxc test
- fix: support untagged VLAN in vxc resource, tests to reflect this

### [v1.2.4] — 2024-11-22

- fix: move csps from partner config update support
- fix: move tests back to old suites
- fix: update docs
- fix: requiresReplace logic for csp vxc partner configs
- fix: import bug and make tests parallel
- fix: add check for product uid match
- docs: add warning about csp partner port uid to diags
- fix: import logic and plan modification if partner config
- fix: import partner logic
- fix: do not change requested product uid upon import
- fix: address import of partner config issue
- fix: plan modifier for ordered vlan
- fix: make check for vendor and size case-insensitive

### [v1.2.3] — 2024-11-20

- fix: examples and passing tests for mve image filtering
- fix: change aws port to loc2 for FullEcosystem example

### [v1.2.2] — 2024-11-20

- docs: generate docs
- cleanup: remove debug code and error handle
- fix: plan modify for destroy case
- fix: support upper case vendor names, provide documentation of currently supported vendors
- fix: remove state vendor config because will be null anyway in this situation
- fix: add checks for the computed values of vendor and size before requiring replace
- docs: generate docs
- fix: support import of MVE
- cleanup: move variable names
- feat: add tests for location name and site code using location data source
- feat: location filter by site code
- fix: updates tests to make it easier to change locations
- fix: add unit tests for filtering mve image
- fix: mod tidy
- fix: bump mpgo version
- fix: attr types
- fix: schema types
- fix: schema for filters
- fix: support filtering of mve images
- fix: make fields optional for azure peering
- fix: bump go version
- fix: add oracle csp connection to vxc resource
- fix: bump megaportgo version
- fix: contract term months in update port, mcr, mve
- fix: change azure service key to 197d927b-90bc-4b1b-bffd-fca17a7ec735

### [v1.2.1] — 2024-11-06

- deps: bumps megaportgo to latest
- fix: try changing loop variables for linter
- fix: require replace if logic, correct structs
- fix: require replace if logic
- fix: support other partner configs for updating vxc
- fix: a-end partner config state on update
- feat: support updating partner config in vxc update
- feat: support diversity zone in MVE

### [v1.2.0] — 2024-09-18

- fix: generate docs for mve updates

### [v1.1.9] — 2024-09-18

- feat: add new MVE image types

### [v1.1.8] — 2024-09-12

- fix: partner end config misname
- fix: variable misname for b-end product uid
- fix: ip routes
- fix: refine warning to list matching partner ports when multiple matching ports found

### [v1.1.7] — 2024-09-05

- fix: make service_key sensitive
- fix: generate docs
- feat: add service key to vxc input

### [v1.1.6] — 2024-09-04

- _No user-facing changes._

### [v1.1.5] — 2024-09-04

- fix: prevent partner port from failing if more than one partner port

### [v1.1.4] — 2024-08-30

- fix: remove adminSSHPublicKey from palo alto config for mve
- fix: sshPublicKey variable
- fix: bump go version
- fix: docs
- fix: ssh public key and go version

### [v1.1.3] — 2024-08-29

- fix: tidies modules
- fix: bumps docs generation version to fix #140
- fix: bumps megaportgo version to handle issue creating MVEs

### [v1.1.2] — 2024-08-14

- fix: improves partner port data source and cleans up code

### [v1.1.1] — 2024-08-12

- fix: typo
- generate: docs
- feat: improves vrouter configuration options
- fix: remove innerVLAN and orderedVLAN changes
- fix: remove new product id
- feat: add validation calls to various resource endpoints
- fix: bump megaportgo version to v1.0.15 (#129)
- fix: bump go version to v1.0.15
- Update issue templates to add feature request
- Update issue templates

### [v1.1.0] — 2024-07-26

- fix: removes planmodifier for requested_product_uid temporarily
- fix: remove comments for mve tests
- fix: make both innerVLANs useStateForUnknown, add inner_vlan to acceptance tests in basic vxc test, move test suites to top of file in vxc and mve
- fix: import_whitelist naming typo
- fix: import_whitelist and templating with docs
- fix: provider docs
- fix: fix issue with ordered_vlan and updates for VXC end configuration, add documentation
- fix: provider template file
- fix: templating for terraform docs
- fix: adds a PlanModifier to VXCs to handle unexpected deletion when connected product is deleted
- fix: additional documentation
- fix: link in provider readme
- deps: bumps to new megaportgo with port dz issue fixed
- fix: diversity_zone now requires replacement
- fix: update move vxc to move from mcr to mcr in the a-end while keeping aws vif on b-end
- docs: additional examples
- fix: azure tests pass with added port_choice attribute
- fix: add terraform to code blocks
- fix: change naming to endpoint
- fix: updates more attributes to UseStateForUnknown()
- fix: rename title of moving vxc end configurations
- feat: moving vxc docs
- docs: regenerates azure examples
- fix: updates examples to include primary/secondary
- docs: regenerates
- feat: adds option to select azure primary or secondary when building VXC
- fix: add testing for diversity zone in single port test
- fix: swapping between files
- feat: vrouter vxc docs
- feat: guides folder, test generation of guide md file
- fix: remove null check for asn
- fix: mcr asn assignment for availability on import
- docs: regenerates for vrouter_config updates
- fix: updates vrouter_config to be optional
- fix: bumps megaportgo version and adds not found checks
- docs: regenerates
- fix: removes resources that are already provided by other parts of the schema / providerr
- fix: bump go version
- docs: clearly notate deprecation
- fix: generation
- fix: fields for bgp, use as path in both deprecated a end and new vrouter
- fix: partner config attrs
- docs: regenerates
- fix: updates attributes with better PlanModifiers, removes redundant fields
- fix: bring back a-end partner config as deprecated
- fix: change partner-end-config to vrouter-config, add support for vrouter config to b-end, add support for as_path_prepend_count to bgp config
- adds: .envrc
- fix: bump go version
- fix: bump megaportgo version
- fix: remove b-end inner vlan for AWS vxc test
- feat: support innerVLAN on bEnd config
- feat: exposure auzre peerings in partner configuration for vxc
- fix: innerVLAN should NOT be passed into update if null/unknown
- fix: updates examples to correctly use AWSHC
- docs: regenerates
- fix: adds MVE
- docs: adds index template and improves index documentation
- fix: casing
- docs: improves readme and adds CLA
- docs: regenerates for service_key update
- fix: updates service_key to be sensitive, closes #28
- fix: adds checks to see if the product has been deleted in the API so it can be recreated

### [v1.0.1] — 2024-06-25

- fix: go module update
- fix: acceptance test and mve resources
- fix: keep cost centre same in first three tests, but change only in fourth test
- fix: add check for innerVLAN after update
- fix: support changing innerVLAN in update
- fix: vxc term update works and code cleanup
- fix: import state verify ignore mve resources, can be updated by api
- fix: remove computed, state for unknown on prefix filter lists
- fix: mcr test coverage, account for empty list for prefix filter lists value
- fix: go version
- fix: go version
- fix: go version
- fix: go version update
- fix: go mod update
- fix: mcr configs and move some locations around in full ecosystem
- fix: remove an unnecessary newline
- fix: aws VXC partner config name field
- fix: update csp connections field description
- fix: schema description tweaks
- fix: transit vxc support
- fix: full ecosystem vxc example
- fix: docs
- fix: support cost centre in MVE, make vnics and vendor config require replace, add update/import mve acceptance tests
- feat: support update and delete mcr prefix filter list, add test coverage for update/delete in mcr acceptance test
- fix: generate docs
- fix: update tests, fix conditional logic for checking prefix filter list entries
- fix: have mcr reads run concurrently, fix tests
- fix: update parsing of prefix filter list
- fix: mcr list endpoint rename
- fix: import typo
- fix: imports
- fix: make location details fully computed in mve
- fix: mve docs
- fix: more docs
- fix: vxc computed fields
- fix: mcr computed fields
- fix: lag port computed
- fix: single port computed vs optional
- fix: make cancelable computed
- fix: locationId
- fix: contract term months
- fix: mcr docs
- fix: more docs
- fix: update docs
- docs: more examples
- docs: more docs
- fix: remove test preCheck
- fix: bump go version
- chore(deps): bump golang.org/x/net from 0.21.0 to 0.23.0
- fix: version bump and docs
- docs: regenerates
- fix: environment variables for provider are now respected, fixes #84
- fix: add check for prefix filter lists in import, but not entries that are only available on creation
- fix: add support for cost centre on VXC buy request
- fix: add sources to provider and gen docs
- docs: generation
- fix: add missing wait time field to provider struct
- fix: bump go version
- docs: generate
- docs: generate
- actions: disables unit tests
- fix: checks out current PR branch
- fix: update examples
- fix: transit vxc support and example with megaport internet
- fix: example for mve vxc aws
- fix: indentations in example test
- fix: a-end inner vlan and vnic index, add test for mve-vxc
- fix: example syntax fix
- fix: tidy
- fix: tidy
- fix: tidy
- fix: mod tidy
- fix: change default value in description to 10
- fix: use package-wide wait time set by provider config for wait time
- fix: mixups between a-end and b-end ordered vlan in schema description
- fix: improve documentation of ordered_vlan in vxc svc, improve validation of ordered_vlan
- Revert "feat: support custom wait time in terraform provider, set default to 20 min"
- feat: support custom wait time in terraform provider, set default to 20 min
- fix: remove vlan from vnics, update examples and acceptance tests to include multiple vnics in create calls
- fix: mcr import state verify
- feat: promo code in resources
- feat: mve images and available sizes as data sources
- fix: check for empty prefix filter list before trying to parse model in create
- fix: make mcr filter list optional/computed
- fix: update examples and docs
- fix: make list of multiple prefix filter lists, get ID from mp api, support read of prefix filter lists, support import of prefix filter lists, update tests
- fix: mcr and mve resource tests, typo in partner port data source
- fix: slight example tweaks
- ci: removes terraform 1.6.* from testing matrix
- fix: updates buy port error message to correctly identify issue
- fix: increases test timeout
- fix: actions should only run once in PRs
- generate: runs docs generation
- fix: removes double equal sign in example
- fix: sets timeout in acceptence tests
- fix: updates name of context check for sloglint
- fix: updates name of govet linter
- fix: param new has same name as predeclared identifier
- fix: removes dead code
- fix: typo in 'information'
- ci: adds integration testing back to action
- fix: dont run all csp vxc tests in parallel because megaport api gets mad
- feat: refactor tests to testify and run all tests in parallel to save time

### [v1.0.0] — 2024-06-03

- fix: improves VXC error messages
- fix: move lag port tests and examples down to single lagCount, update docs
- fix: add full ecosystem tests, update lag port resource to support lagCount of 1
- fix: update go module
- fix: update go module
- fix: test full ecosystem
- fix: tf files
- fix: add diversity zone support to mcr, add location details to products, fix vxc update, fix tests
- fix: add diversity zone to mcr
- fix: adds error check to ModifyPort call
- fix: add requested_product_uid and current_product_uid to vxc, update tests and docs
- fix: marketplace visibility for mcr test
- fix: name of product uid for data.megaport_partner.aws_port.product_uid
- cleanup: rename plan to state in update mcr, remove ordered_product_uid from end configs in vxc, remove port_uid from vxc --- note that the megaport API will break provider on aws vxcs because it sometimes will change the product uid after the user has specified one
- fix: handle error for modify port
- fix: remove marketplace visibility from buy call
- cleanup: uncomment tests
- fix: remove requiresReplaceIf
- cleanup: update dependencies
- fix: update examplesd
- fix: schema updates
- fix: marketplace visibility for port and mcr
- fix: vxc update, cost centre in products/mcr/ports, update marketplace_visibility, fix modifying port bug, make market computed across provider in all resources
- feat: adds user-agent header with useful client info
- fix: terraform lock cleanup and examples
- fix: sets version to allowed actions
- upgrade: uses v1 of the megaportgo library
- deletes: example we don't use
- feat: adds workflows for releasing
- fix: add meraki vendor config, add test for versa vendor config on mve, add fields to mve vendor configs
- fix: update calls
- fix: improvements to tests, add missing fields in single port and lag port resources
- feat: pass in "x-app: terraform" header
- fix: fixes incorrect client name
- fix: tidies modules
- fix: mve updates, mve example, docs
- cleanup: objects in locations data source
- docs: more tf examples
- docs: tf examples
- docs: new docs
- fix: bugs with bgp a-end partner config
- fix: mcr prefix filter calls, a-end partner config
- fix: bug with uid in lag port and import state verify ignore / uid bugs in resource test
- fix: fix bug for partner a-end config
- change to a maintained gpg import github action ref PLAT-253
- docs: update docs
- fix: gcp/azure tests and fix some partner config stuff in vxc resource
- fix: vxc config stuff for end configuration, add testing for mcr-vxc-aws
- cleanup: uncomment tests
- fix: change type of vxc to VROUTER for virtual router csp connection
- fix: lag port resource
- fix: object and list marshaling for mve
- fix: import state ignore fields
- fix: object marshaling for VXC and add import testing to vxc
- fix: dont check for errors in diags until end of call
- fix: diags in vxc
- fix: diags in mve resource
- fix: mcr diags
- cleanup: move dates in mcr helper
- fix: import logic and testing mcr and port import states
- fix: clean up vxc resource, add test
- fix: locations data source, add locations to mcr and port tests
- fix: refactor vxc to use partner config objects and parse in req
- fix: error handling for object and map marshaling
- fix: refactor mve vendor configs
- fix: start mve refactor for objects
- fix: remove temp file
- fix: mcr cleanup, mcr acceptance test
- feat: single port and provider testing, fixes in single port and location
- feat: locations data source
- cleanup: mve fixes
- cleanup: mcr fixes
- docs: docs
- fix: schema for product_uid
- fix: update schema for mve
- docs: slight tweak to description for mcr schema
- docs: change schema description
- cleanup: change port uid to product_uid to match megaport api
- cleanup: change port uid to product_uid
- docs: add description for schema at top level
- fix: update logic for mcr
- fix: mve schema and update logic
- fix: change asn to optional, update docs
- fix: formatting
- fix: formatting
- fix: formatting
- fix: more schema updates for mve
- fix: mve schemas and docs
- fix: schema and add docs
- feat: add port interface model for mcrs and mves
- fix: port interface model
- feat: more mcr stuff and port interface
- feat: more mve structs
- docs: mcr and port
- docs: resource for port
- docs: resources for port and mve
- docs: descriptions for port schema
- fix: descriptions for mve fields
- fix: descriptions for the schema
- fix: updates to the schema
- cleanup: tfsdk field names
- feat: starter code for mve
- fix: naming stuffs
- fix: remove diversity zone from mcr schema
- feat: start mcr boilerplate
- wip: ports

### [v0.4.2] — 2024-04-30

- fix: updates to newest version of megaportgo that fixes #54

### [v0.4.1] — 2023-11-08

- Remove explicit ldflags from goreleaser config

### [v0.4.0] — 2023-11-08

- Update GitHub Action to use Go 1.21
- Remove toolchain statement from go.mod

### [v0.3.0] — 2023-10-05

- Update GitHub Actions

### [v0.2.10] — 2023-07-28

- Fix typo
- DEVOPS-2886 oracle oci support
- Adding Oracle and OCI VXC

### [v0.2.9] — 2023-04-05

- Update go.mod
- Update go.mod

### [v0.2.8] — 2023-02-20

- _No user-facing changes._

### [v0.2.7] — 2022-12-08

- \# 0.2.7-beta (Dec 8, 2022)

### [v0.2.6] — 2022-11-02

- Update go.mod
- Update Makefile
- \# 0.2.6-beta (Nov 2, 2022)

### [v0.2.5] — 2022-05-05

- \# 0.2.5-beta (May 5, 2022)
- Import `MarshallMcrAEndConfig` from upstream megaportgo client.

### [v0.2.4] — 2022-04-06

- Remove tab
- \# 0.2.4-beta (April 6, 2022)
- Add import support for megaport_aws_connection Add a connection_name attribute for AWS connection Remove erroneous Required flags on requested_asn and amazon_asn (not supported on AWSHC)
- Add optional connection_name attribute for AWS connections

### [v0.2.3] — 2022-03-24

- \# 0.2.3-beta (March 24, 2022)

### [v0.2.2] — 2022-03-02

- \# 0.2.2-beta (March 2, 2022)

### [v0.2.1] — 2022-02-22

- \# 0.2.1-beta (February 22, 2022)

### [v0.2.0] — 2022-01-27

- \# 0.2.0-beta (January 27, 2022)

### [v0.1.10] — 2021-11-05

- \# 0.1.10-beta (November 5, 2021)

### [v0.1.9] — 2021-08-19

- \# 0.1.9-beta (August 19, 2021)

### [v0.1.8] — 2021-06-19

- \## 0.1.8-beta (June 19, 2021) Notes
- updated megaportgo reference to latest tag
- added requested_product_id to gcp_connection csp settings to allow selecting the google b end location

### [v0.1.7] — 2021-06-04

- Notes
- Update doco, add in mfa env, chaneg user env
- add provider environment vars

### [v0.1.6] — 2021-02-11

- 0.1.6-beta (February 11, 2021)

### [v0.1.5] — 2021-02-10

- \## 0.1.4-beta (January 12, 2021)

### [v0.1.4] — 2021-01-12

- 0.1.4-beta (January 12, 2021)

### [v0.1.3] — 2020-12-22

- Wiki Documentation update (No functionality changes)
- Wiki Documentation update (No functionality changes)

### [v0.1.2] — 2020-12-09

- Reformat Documentation for Terraform
- Documentation update (no functionality changes).
- Documentation and examples update (no functionality changes).

### [v0.1.1] — 2020-12-01

- 0.1.0-beta (December 1, 2020)
- Add Workflow

