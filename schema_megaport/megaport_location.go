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

// DataLocationSchema is the data schema of a Megaport Location
func DataLocationSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"name": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"country": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"live_date": {
			Type:     schema.TypeInt,
			Computed: true,
		},
		"site_code": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"address": {
			Type:     schema.TypeMap,
			Computed: true,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
		},
		"latitude": {
			Type:     schema.TypeFloat,
			Computed: true,
		},
		"longitude": {
			Type:     schema.TypeFloat,
			Computed: true,
		},
		"market": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"market_code": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"metro": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"mcr_available": {
			Type:     schema.TypeBool,
			Computed: true,
		},
		"has_mcr": {
			Type:     schema.TypeBool,
			Optional: true,
		},
		"id": {
			Type:     schema.TypeInt,
			Computed: true,
		},
		"status": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"match_exact": {
			Type:     schema.TypeBool,
			Optional: true,
			Default:  false,
		},
	}
}
