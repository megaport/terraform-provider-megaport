// Copyright 2020 Megaport Pty Ltd
//
// Licensed under the Mozilla Public License, Version 2.0 (the
// "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//       https://mozilla.org/MPL/2.0/
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/megaport/terraform-provider-megaport/data_megaport"
	"github.com/megaport/terraform-provider-megaport/resource_megaport"
	"github.com/megaport/terraform-provider-megaport/terraform_utility"
)

const ERR_USER_NOT_ACCEPT_TOS = "sorry, you haven't accepted the Megaport terms of service"

func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"environment": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "staging",
				DefaultFunc: schema.EnvDefaultFunc("MEGAPORT_ENVIRONMENT", nil),
			},
			"accept_purchase_terms": {
				Type:     schema.TypeBool,
				Required: true,
			},
			"access_key": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("MEGAPORT_ACCESS_KEY", nil),
			},
			"secret_key": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("MEGAPORT_SECRET_KEY", nil),
			},
			"delete_ports": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"megaport_port":             resource_megaport.MegaportPort(),
			"megaport_mcr":              resource_megaport.MegaportMCR(),
			"megaport_mve":              resource_megaport.MegaportMVE(),
			"megaport_vxc":              resource_megaport.MegaportVXC(),
			"megaport_aws_connection":   resource_megaport.MegaportAWSConnection(),
			"megaport_gcp_connection":   resource_megaport.MegaportGcpConnection(),
			"megaport_azure_connection": resource_megaport.MegaportAzureConnection(),
			"megaport_oci_connection":   resource_megaport.MegaportOciConnection(),
		},
		ConfigureFunc: providerConfigure,
		DataSourcesMap: map[string]*schema.Resource{
			"megaport_location":         data_megaport.MegaportLocation(),
			"megaport_internet":         data_megaport.MegaportInternet(),
			"megaport_port":             data_megaport.MegaportPort(),
			"megaport_mcr":              data_megaport.MegaportMCR(),
			"megaport_mve":              data_megaport.MegaportMVE(),
			"megaport_vxc":              data_megaport.MegaportVXC(),
			"megaport_partner_port":     data_megaport.MegaportPartnerPort(),
			"megaport_aws_connection":   data_megaport.MegaportAWSConnection(),
			"megaport_gcp_connection":   data_megaport.MegaportGcpConnection(),
			"megaport_azure_connection": data_megaport.MegaportAzureConnection(),
			"megaport_oci_connection":   data_megaport.MegaportOciConnection(),
		},
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	acceptTerms, ato := d.GetOk("accept_purchase_terms")

	if !ato || !acceptTerms.(bool) {
		return nil, errors.New(ERR_USER_NOT_ACCEPT_TOS)
	}

	// Grab credentials from config.
	accessKey, ok := d.Get("access_key").(string)
	if !ok {
		return nil, fmt.Errorf("invalid or missing 'access_key'")
	}
	secretKey, ok := d.Get("secret_key").(string)
	if !ok {
		return nil, fmt.Errorf("invalid or missing 'secret_key'")
	}

	megaportUrl := getEnvironmentUrl(d)
	deletePorts := shouldDeletePorts(d)

	megaportClient := terraform_utility.MegaportClient{
		DeletePorts: deletePorts,
		Url:         megaportUrl,
	}

	if err := megaportClient.ConfigureServices(accessKey, secretKey); err != nil {
		return nil, err
	}

	return &megaportClient, nil
}

func getEnvironmentUrl(d *schema.ResourceData) string {
	env, ok := d.GetOk("environment")
	if !ok {
		return "https://api-staging.megaport.com/"
	}

	environment := env.(string)
	if environment == "production" {
		return "https://api.megaport.com/"
	} else if environment != "" {
		return "https://api-" + environment + ".megaport.com/"
	} else {
		return "https://api-staging.megaport.com/"
	}
}

func shouldDeletePorts(d *schema.ResourceData) bool {
	if shouldDelete, ok := d.GetOk("delete_ports"); ok {
		return shouldDelete.(bool)
	} else {
		return false
	}
}
