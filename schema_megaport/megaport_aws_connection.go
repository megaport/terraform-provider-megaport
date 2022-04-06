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

package schema_megaport

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func ResourceAWSConnectionVXCSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"vxc_name": {
			Type:     schema.TypeString,
			Required: true,
		},
		"vxc_type": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"rate_limit": {
			Type:     schema.TypeInt,
			Required: true,
		},
		"provisioning_status": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"a_end":                   ResourceVxcEndConfiguration(),
		"b_end":                   DataVxcEndConfiguration(),
		"csp_settings":            ResourceAwsConnectionCspSettings(),
		"a_end_mcr_configuration": ResourceMcrConfigurationSettings(),
		"created_by": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"live_date": {
			Type:     schema.TypeInt,
			Computed: true,
		},
		"create_date": {
			Type:     schema.TypeInt,
			Computed: true,
		},
		"company_name": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"locked": {
			Type:     schema.TypeBool,
			Computed: true,
		},
		"admin_locked": {
			Type:     schema.TypeBool,
			Computed: true,
		},
		"vxc_internal_type": {
			Type:     schema.TypeString,
			Optional: true,
			Default:  "aws",
		},
		"aws_id": {
			Type:     schema.TypeString,
			Computed: true,
		},
	}
}

func ResourceAwsConnectionCspSettings() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		Required: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"visibility": {
					Type:     schema.TypeString,
					Optional: true,
					Default:  "private",
					ForceNew: true,
				},
				"requested_product_id": {
					Type:     schema.TypeString,
					Required: true,
					ForceNew: true,
				},
				"requested_asn": {
					Type:     schema.TypeInt,
					Required: true,
					ForceNew: true,
				},
				"assigned_asn": {
					Type:     schema.TypeInt,
					Computed: true,
				},
				"amazon_asn": {
					Type:     schema.TypeInt,
					Required: true,
					ForceNew: true,
				},
				"amazon_account": {
					Type:     schema.TypeString,
					Required: true,
					ForceNew: true,
				},
				"auth_key": {
					Type:     schema.TypeString,
					Optional: true,
					ForceNew: true,
					Default:  "",
				},
				"prefixes": {
					Type:     schema.TypeString,
					Optional: true,
					ForceNew: true,
					Default:  "",
				},
				"customer_ip": {
					Type:     schema.TypeString,
					Optional: true,
					ForceNew: true,
					Default:  "",
				},
				"amazon_ip": {
					Type:     schema.TypeString,
					Optional: true,
					ForceNew: true,
					Default:  "",
				},
				"hosted_connection": {
					Type:     schema.TypeBool,
					Optional: true,
					Default:  false,
					ForceNew: true,
				},
				"connection_name": {
					Type:     schema.TypeString,
					Optional: true,
					ForceNew: true,
					Default:  "",
				},
			},
		},
	}
}
