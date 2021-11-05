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
		"rate_limit": {
			Type:     schema.TypeInt,
			Required: true,
		},
		"a_end":        AWSConnectionEndConfiguration(),
		"b_end":        DataVxcEndConfiguration(),
		"csp_settings": ResourceAwsConnectionCspSettings(),
		"vxc_type": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"provisioning_status": {
			Type:     schema.TypeString,
			Computed: true,
		},
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
		"aws_id": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"vxc_internal_type": {
			Type:     schema.TypeString,
			Optional: true,
			Default:  "aws",
		},
	}
}

func AWSConnectionEndConfiguration() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		Optional: true,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"port_id": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"owner_uid": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"name": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"location": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"assigned_vlan": {
					Type:     schema.TypeInt,
					Computed: true,
				},
				"assigned_asn": {
					Type:     schema.TypeInt,
					Computed: true,
				},
				"requested_vlan": {
					Type:     schema.TypeInt,
					Required: true,
				},
				"partner_configuration": ResourceAwsConnectionPartnerConfigurationSettings(),
			},
		},
	}
}

func ResourceAwsConnectionPartnerConfigurationSettings() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		Optional: true,
		Computed: true,
		ForceNew: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"ip_addresses": {
					Type:     schema.TypeList,
					Optional: true,
					Computed: true,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
				},
				"bfd_configuration": ResourceAwsConnectionBfdConfigSettings(),
				"bgp_connection":    ResourceAwsConnectionBgpConnectionSettings(),
			},
		},
	}
}

func ResourceAwsConnectionBfdConfigSettings() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		Optional: true,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"tx_internal": {
					Type:     schema.TypeInt,
					Required: true,
				},
				"rx_internal": {
					Type:     schema.TypeInt,
					Required: true,
				},
				"multiplier": {
					Type:     schema.TypeInt,
					Required: true,
				},
			},
		},
	}
}

func ResourceAwsConnectionBgpConnectionSettings() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"peer_asn": {
					Type:     schema.TypeInt,
					Required: true,
				},
				"local_ip_address": {
					Type:     schema.TypeString,
					Required: true,
				},
				"peer_ip_address": {
					Type:     schema.TypeString,
					Required: true,
				},
				"password": {
					Type:     schema.TypeString,
					Optional: true,
				},
				"shutdown": {
					Type:     schema.TypeBool,
					Optional: true,
					Default:  false,
				},
				"description": {
					Type:     schema.TypeString,
					Optional: true,
					Default:  "",
				},
				"med_in": {
					Type:     schema.TypeInt,
					Optional: true,
					Default:  0,
				},
				"med_out": {
					Type:     schema.TypeInt,
					Optional: true,
					Default:  0,
				},
				"bfd_enabled": {
					Type:     schema.TypeBool,
					Optional: true,
					Default:  false,
				},
			},
		},
	}
}

func ResourceAwsConnectionCspSettings() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		Required: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"attached_to": {
					Type:     schema.TypeString,
					Required: true,
					ForceNew: true,
				},
				"requested_product_id": {
					Type:     schema.TypeString,
					Required: true,
					ForceNew: true,
				},
				"visibility": {
					Type:     schema.TypeString,
					Optional: true,
					Default:  "private",
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
			},
		},
	}
}
