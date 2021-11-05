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

import "github.com/hashicorp/terraform-plugin-sdk/helper/schema"

func ResourcePortSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"port_name": {
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
		"port_speed": {
			Type:     schema.TypeInt,
			Required: true,
			ForceNew: true,
		},
		"live_date": {
			Type:     schema.TypeInt,
			Computed: true,
		},
		"market_code": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"location_id": {
			Type:     schema.TypeInt,
			Required: true,
		},
		"marketplace_visibility": {
			Type:     schema.TypeBool,
			Optional: true,
			Default:  false,
		},
		"company_name": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"term": {
			Type:     schema.TypeInt,
			Optional: true,
			ForceNew: true,
			Default:  1,
		},
		"lag_primary": {
			Type:     schema.TypeBool,
			Computed: true,
			Optional: true,
		},
		"lag_id": {
			Type:     schema.TypeInt,
			Computed: true,
			Optional: true,
		},
		"locked": {
			Type:     schema.TypeBool,
			Computed: true,
		},
		"admin_locked": {
			Type:     schema.TypeBool,
			Computed: true,
		},
		"lag": {
			Type:     schema.TypeBool,
			Optional: true,
			Default:  false,
			ForceNew: true,
		},
		"lag_port_count": {
			Type:     schema.TypeInt,
			Optional: true,
			Default:  0,
			ForceNew: true,
		},
	}
}

func DataPortSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"port_name": {
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
		"market_code": {
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
		"term": {
			Type:     schema.TypeInt,
			Computed: true,
		},
		"port_id": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"lag_primary": {
			Type:     schema.TypeBool,
			Computed: true,
			Optional: true,
		},
		"lag_id": {
			Type:     schema.TypeInt,
			Computed: true,
			Optional: true,
		},
		"locked": {
			Type:     schema.TypeBool,
			Computed: true,
		},
		"admin_locked": {
			Type:     schema.TypeBool,
			Computed: true,
		},
		"lag": {
			Type:     schema.TypeBool,
			Optional: true,
			Default:  false,
			ForceNew: true,
		},
		"lag_port_count": {
			Type:     schema.TypeInt,
			Optional: true,
			Default:  0,
			ForceNew: true,
		},
	}
}
