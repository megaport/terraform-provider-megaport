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

package data_megaport

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/megaport/terraform-provider-megaport/resource_megaport"
	"github.com/megaport/terraform-provider-megaport/schema_megaport"
	"github.com/megaport/terraform-provider-megaport/terraform_utility"
)

func MegaportMVE() *schema.Resource {
	return &schema.Resource{
		Read:   dataMegaportMVERead,
		Schema: schema_megaport.DataMegaportMVESchema(),
	}
}

func dataMegaportMVERead(d *schema.ResourceData, m interface{}) error {
	mve := m.(*terraform_utility.MegaportClient).Mve

	mveUid := d.Get("mve_id").(string)
	d.SetId(mveUid)

	details, err := resource_megaport.FetchMVEDetails(mve, d)
	if err != nil {
		return err
	}

	resource_megaport.MVEPopulateBaseResourceData(details, d)

	return nil
}
