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
	"github.com/megaport/megaportgo/authentication"
	"github.com/megaport/megaportgo/types"
	"github.com/megaport/terraform-provider-megaport/data_megaport"
	"github.com/megaport/terraform-provider-megaport/resource_megaport"
	"github.com/megaport/terraform-provider-megaport/terraform_utility"
	"github.com/engi-fyi/go-credentials/credential"
	"github.com/engi-fyi/go-credentials/factory"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"strconv"
)

const ERR_USER_NOT_ACCEPT_TOS = "sorry, you haven't accepted the Megaport terms of service"

func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"environment": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "staging",
			},
			"accept_purchase_terms": {
				Type:     schema.TypeBool,
				Required: true,
			},
			"username": {
				Type:     schema.TypeString,
				Required: true,
			},
			"password": {
				Type:     schema.TypeString,
				Required: true,
			},
			"mfa_otp_key": {
				Type:     schema.TypeString,
				Optional: true,
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
		},
		ConfigureFunc: providerConfigure,
		DataSourcesMap: map[string]*schema.Resource{
			"megaport_port":           data_megaport.MegaportPort(),
			"megaport_location":       data_megaport.MegaportLocation(),
			"megaport_vxc":            data_megaport.MegaportVXC(),
			"megaport_partner_port":   data_megaport.MegaportPartnerPort(),
			"megaport_aws_connection": data_megaport.MegaportAWSConnection(),
			"megaport_gcp_connection":   data_megaport.MegaportGcpConnection(),
			"megaport_azure_connection": data_megaport.MegaportAzureConnection(),
			"megaport_mcr":            data_megaport.MegaportMCR(),
		},
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	if acceptTerms, ato := d.GetOk("accept_purchase_terms"); ato {
		if acceptTerms.(bool) {
			username, password, otpKey := checkAuthVariables(d)
			megaportUrl := getEnvironmentUrl(d)
			deletePorts := shouldDeletePorts(d)

			megaportClient := terraform_utility.MegaportClient{}
			credFactory, factErr := factory.New(types.APPLICATION_SHORT_NAME)

			if factErr != nil {
				return nil, factErr
			}

			myCredentials, credErr := credential.New(credFactory, username, password)

			if credErr != nil {
				return nil, credErr
			}

			if otpKey != "" {
				setErr := myCredentials.Section("otp").SetAttribute("key", otpKey)

				if setErr != nil {
					return nil, setErr
				}
			}

			myCredentials.SetAttribute("megaport_url", megaportUrl)
			myCredentials.Section("options").SetAttribute("delete_ports", strconv.FormatBool(deletePorts))
			myCredentials.Save()
			var authErr error
			megaportClient.Credentials, authErr = authentication.Login(true)
			megaportClient.Url = megaportUrl
			megaportClient.DeletePorts = deletePorts

			if authErr != nil {
				return nil, authErr
			}

			return megaportClient, nil
		}
	}

	return nil, errors.New(ERR_USER_NOT_ACCEPT_TOS)
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
