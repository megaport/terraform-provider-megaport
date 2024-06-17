// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/stretchr/testify/suite"
)

var providerConfig = fmt.Sprintf(`
provider "megaport" {
  environment = "staging"
  access_key = "%s"
  secret_key     = "%s"
  accept_purchase_terms = true
}
`, os.Getenv("MEGAPORT_ACCESS_KEY"), os.Getenv("MEGAPORT_SECRET_KEY"))

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"scaffolding": providerserver.NewProtocol6WithError(New("test")()),
	"megaport":    providerserver.NewProtocol6WithError(New("test")()),
}

type ProviderTestSuite struct {
	suite.Suite
}
