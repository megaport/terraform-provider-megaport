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

func ResourceVXCSchema() map[string]*schema.Schema {
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
		"b_end":                   ResourceVxcEndConfiguration(),
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
			Default:  "vxc",
		},
	}
}

func ResourceVxcEndConfiguration() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		Required: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"port_id": {
					Type:     schema.TypeString,
					Required: true,
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
				"requested_vlan": {
					Type:     schema.TypeInt,
					Optional: true,
					Default:  0,
				},
				"assigned_vlan": {
					Type:     schema.TypeInt,
					Computed: true,
				},
				"assigned_asn": {
					Type:     schema.TypeInt,
					Computed: true,
				},
			},
		},
	}
}

func ResourceMcrConfigurationSettings() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		Optional: true,
		ForceNew: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"ip_addresses": {
					Type:     schema.TypeList,
					Optional: true,
					Computed: true,
					ForceNew: true,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
				},
				"nat_ip_addresses": {
					Type:     schema.TypeList,
					Optional: true,
					Computed: true,
					ForceNew: true,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
				},
				"bfd_configuration": ResourceBfdConfigSettings(),
				"bgp_connection":    ResourceBgpConnectionSettings(),
			},
		},
	}
}

func ResourceBfdConfigSettings() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		Optional: true,
		Computed: true,
		ForceNew: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"tx_interval": {
					Type:     schema.TypeInt,
					Required: true,
				},
				"rx_interval": {
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

func ResourceBgpConnectionSettings() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Computed: true,
		ForceNew: true,
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

func DataVXCSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"vxc_id": {
			Type:     schema.TypeString,
			Required: true,
		},
		"vxc_name": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"vxc_type": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"rate_limit": {
			Type:     schema.TypeInt,
			Computed: true,
		},
		"provisioning_status": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"a_end":                   DataVxcEndConfiguration(),
		"b_end":                   DataVxcEndConfiguration(),
		"a_end_mcr_configuration": DataMcrConfigurationSettings(),
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
	}
}

func DataVxcEndConfiguration() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
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
				"requested_vlan": {
					Type:     schema.TypeInt,
					Required: true,
				},
				"assigned_vlan": {
					Type:     schema.TypeInt,
					Computed: true,
				},
				"assigned_asn": {
					Type:     schema.TypeInt,
					Computed: true,
				},
			},
		},
	}
}

func DataMcrConfigurationSettings() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"ip_addresses": {
					Type:     schema.TypeList,
					Computed: true,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
				},
				"nat_ip_addresses": {
					Type:     schema.TypeList,
					Computed: true,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
				},
				"bfd_configuration": DataBfdConfigSettings(),
				"bgp_connection":    DataBgpConnectionSettings(),
			},
		},
	}
}

func DataBfdConfigSettings() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"tx_interval": {
					Type:     schema.TypeInt,
					Computed: true,
				},
				"rx_interval": {
					Type:     schema.TypeInt,
					Computed: true,
				},
				"multiplier": {
					Type:     schema.TypeInt,
					Computed: true,
				},
			},
		},
	}
}

func DataBgpConnectionSettings() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"peer_asn": {
					Type:     schema.TypeInt,
					Computed: true,
				},
				"local_ip_address": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"peer_ip_address": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"password": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"shutdown": {
					Type:     schema.TypeBool,
					Computed: true,
				},
				"description": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"med_in": {
					Type:     schema.TypeInt,
					Computed: true,
				},
				"med_out": {
					Type:     schema.TypeInt,
					Computed: true,
				},
				"bfd_enabled": {
					Type:     schema.TypeBool,
					Computed: true,
				},
			},
		},
	}
}
