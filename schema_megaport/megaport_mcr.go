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

func ResourceMegaportMCRSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"mcr_name": {
			Type:     schema.TypeString,
			Required: true,
		},
		"type": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"provisioning_status": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"create_date": {
			Type:     schema.TypeInt,
			Computed: true,
		},
		"created_by": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"terminate_date": {
			Type:     schema.TypeInt,
			Computed: true,
		},
		"live_date": {
			Type:     schema.TypeInt,
			Computed: true,
		},
		"market": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"location_id": {
			Type:     schema.TypeInt,
			Required: true,
		},
		"marketplace_visibility": {
			Type:     schema.TypeBool,
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
		"router":             ResourceMcrVirtualRouterConfiguration(),
		"prefix_filter_list": ResourceMcrPrefixFilterList(),
	}
}

func ResourceMcrVirtualRouterConfiguration() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		Required: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"assigned_asn": {
					Type:     schema.TypeInt,
					Computed: true,
				},
				"requested_asn": {
					Type:     schema.TypeInt,
					Optional: true,
					Default:  0,
					ForceNew: true,
				},
				"port_speed": {
					Type:     schema.TypeInt,
					Required: true,
					ForceNew: true,
				},
			},
		},
	}
}

func ResourceMcrPrefixFilterList() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"name": {
					Type:     schema.TypeString,
					Required: true,
					ForceNew: true,
				},
				"address_family": {
					Type:     schema.TypeString,
					Required: true,
					ForceNew: true,
				},
				"entry": ResourceMcrPrefixListEntry(),
			},
		},
	}
}

func ResourceMcrPrefixListEntry() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		Required: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"action": {
					Type:     schema.TypeString,
					Required: true,
					ForceNew: true,
				},
				"prefix": {
					Type:     schema.TypeString,
					Required: true,
					ForceNew: true,
				},
				"range_min": {
					Type:     schema.TypeInt,
					Optional: true,
					ForceNew: true,
				},
				"range_max": {
					Type:     schema.TypeInt,
					Optional: true,
					ForceNew: true,
				},
			},
		},
	}
}

func DataMegaportMCRSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"mcr_id": {
			Type:     schema.TypeString,
			Required: true,
		},
		"mcr_name": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"type": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"provisioning_status": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"create_date": {
			Type:     schema.TypeInt,
			Computed: true,
		},
		"created_by": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"port_speed": {
			Type:     schema.TypeInt,
			Computed: true,
		},
		"live_date": {
			Type:     schema.TypeInt,
			Computed: true,
		},
		"market": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"location_id": {
			Type:     schema.TypeInt,
			Computed: true,
		},
		"marketplace_visibility": {
			Type:     schema.TypeBool,
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
		"router":             DataMcrVirtualRouterConfiguration(),
		"prefix_filter_list": DataMcrPrefixFilterList(),
	}
}

func DataMcrVirtualRouterConfiguration() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"assigned_asn": {
					Type:     schema.TypeInt,
					Computed: true,
				},
				"port_speed": {
					Type:     schema.TypeInt,
					Computed: true,
				},
			},
		},
	}
}

func DataMcrPrefixFilterList() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"name": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"address_family": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"entry": DataMcrPrefixListEntry(),
			},
		},
	}
}

func DataMcrPrefixListEntry() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"action": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"prefix": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"range_min": {
					Type:     schema.TypeInt,
					Computed: true,
				},
				"range_max": {
					Type:     schema.TypeInt,
					Computed: true,
				},
			},
		},
	}
}
