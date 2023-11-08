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

func ResourceMegaportMVESchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"mve_name": {
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
		"term": {
			Type:     schema.TypeInt,
			Optional: true,
			ForceNew: true,
			Default:  1,
		},
		"locked": {
			Type:     schema.TypeBool,
			Computed: true,
		},
		"admin_locked": {
			Type:     schema.TypeBool,
			Computed: true,
		},
		"vendor": {
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
		"size": {
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
		"image_id": {
			Type:     schema.TypeInt,
			Required: true,
			ForceNew: true,
		},
		"image_vendor": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"image_product": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"image_version": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"vendor_config": {
			Type:      schema.TypeMap,
			Required:  true,
			Sensitive: true,
		},
		"vnic": {
			Type:     schema.TypeList,
			Optional: true,
			MaxItems: 5,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"description": {
						Type:     schema.TypeString,
						Required: true,
					},
					"index": {
						Type:     schema.TypeInt,
						Computed: true,
					},
					"vlan": {
						Type:     schema.TypeInt,
						Computed: true,
					},
				},
			},
		},
	}
}

func DataMegaportMVESchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"mve_name": {
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
		"term": {
			Type:     schema.TypeInt,
			Optional: true,
			ForceNew: true,
			Default:  1,
		},
		"locked": {
			Type:     schema.TypeBool,
			Computed: true,
		},
		"admin_locked": {
			Type:     schema.TypeBool,
			Computed: true,
		},
		"vendor": {
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
		"size": {
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
		"image_id": {
			Type:     schema.TypeInt,
			Required: true,
			ForceNew: true,
		},
		"image_vendor": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"image_product": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"image_version": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"vnic": {
			Type:     schema.TypeList,
			Optional: true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"description": {
						Type:     schema.TypeString,
						Required: true,
					},
					"index": {
						Type:     schema.TypeInt,
						Computed: true,
					},
					"vlan": {
						Type:     schema.TypeInt,
						Computed: true,
					},
				},
			},
		},
	}
}
