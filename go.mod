module github.com/megaport/terraform-provider-megaport

# For local development
#replace github.com/megaport/megaportgo => <megaportgo local directory>

go 1.13

require (
	github.com/aws/aws-sdk-go v1.25.3
	github.com/hashicorp/terraform-plugin-sdk v1.9.1
	github.com/megaport/megaportgo v0.1.15-beta
)
