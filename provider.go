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
			"username": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("MEGAPORT_USERNAME", nil),
			},
			"password": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("MEGAPORT_PASSWORD", nil),
			},
			"mfa_otp_key": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("MEGAPORT_MFA_OTP_KEY", nil),
			},
			"delete_ports": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"megaport_port":             resource_megaport.MegaportPort(),
			"megaport_vxc":              resource_megaport.MegaportVXC(),
			"megaport_aws_connection":   resource_megaport.MegaportAWSConnection(),
			"megaport_mcr":              resource_megaport.MegaportAWS(),
			"megaport_gcp_connection":   resource_megaport.MegaportGcpConnection(),
			"megaport_azure_connection": resource_megaport.MegaportAzureConnection(),
			"megaport_oci_connection":   resource_megaport.MegaportOciConnection(),
		},
		ConfigureFunc: providerConfigure,
		DataSourcesMap: map[string]*schema.Resource{
			"megaport_port":             data_megaport.MegaportPort(),
			"megaport_location":         data_megaport.MegaportLocation(),
			"megaport_vxc":              data_megaport.MegaportVXC(),
			"megaport_partner_port":     data_megaport.MegaportPartnerPort(),
			"megaport_aws_connection":   data_megaport.MegaportAWSConnection(),
			"megaport_gcp_connection":   data_megaport.MegaportGcpConnection(),
			"megaport_azure_connection": data_megaport.MegaportAzureConnection(),
			"megaport_oci_connection": 	 data_megaport.MegaportOciConnection(),
			"megaport_mcr":              data_megaport.MegaportMCR(),
		},
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	acceptTerms, ato := d.GetOk("accept_purchase_terms")

	if !ato || !acceptTerms.(bool) {
		return nil, errors.New(ERR_USER_NOT_ACCEPT_TOS)
	}

	username, password, otpKey := checkAuthVariables(d)
	megaportUrl := getEnvironmentUrl(d)
	deletePorts := shouldDeletePorts(d)

	megaportClient := terraform_utility.MegaportClient{
		DeletePorts: deletePorts,
		Url:         megaportUrl,
	}

	err := megaportClient.ConfigureServices(username, password, otpKey)

	if err != nil {
		return nil, err
	}

	return &megaportClient, nil
}

func checkAuthVariables(d *schema.ResourceData) (string, string, string) {
	username := ""
	password := ""
	oneTimePasswordKey := ""

	if u, ok := d.GetOk("username"); ok {
		username = u.(string)
	}

	if p, ok := d.GetOk("password"); ok {
		password = p.(string)
	}

	if otp, ok := d.GetOk("mfa_otp_key"); ok {
		oneTimePasswordKey = otp.(string)
	}

	return username, password, oneTimePasswordKey
}

func getEnvironmentUrl(d *schema.ResourceData) string {
	if environment, ok := d.GetOk("environment"); ok {
		myEnvironment := environment.(string)

		if myEnvironment == "production" {
			return "https://api.megaport.com/"
		} else if myEnvironment != "" {
			return "https://api-" + myEnvironment + ".megaport.com/"
		} else {
			return "https://api-staging.megaport.com/"
		}
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
