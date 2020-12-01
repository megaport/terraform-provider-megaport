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
		"a_end": ResourceVxcEndConfiguration(),
		"b_end": ResourceVxcEndConfiguration(),
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
			Type:	  schema.TypeString,
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
					Required: true,
				},
				"assigned_vlan": {
					Type:     schema.TypeInt,
					Computed: true,
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
		"a_end": DataVxcEndConfiguration(),
		"b_end": DataVxcEndConfiguration(),
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

func ResourcePartnerConnectionEndConfiguration() *schema.Schema {
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
