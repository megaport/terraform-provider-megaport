module github.com/megaport/terraform-provider-megaport

go 1.13

require (
	github.com/engi-fyi/go-credentials v1.2.1
	github.com/hashicorp/terraform-plugin-sdk v1.9.1
	github.com/megaport/megaportgo v0.1.2-beta
)

// Will remove this and reference updated version once https://github.com/megaport/megaportgo/pull/2 is merged
replace github.com/megaport/megaportgo => github.com/kdw174/megaportgo v0.1.3-beta
