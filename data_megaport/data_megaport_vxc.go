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
	"github.com/megaport/megaportgo/vxc"
	"github.com/megaport/terraform-provider-megaport/schema_megaport"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func MegaportVXC() *schema.Resource {
	return &schema.Resource{
		Read:   DataMegaportVXCRead,
		Schema: schema_megaport.DataVXCSchema(),
	}
}

func DataMegaportVXCRead(d *schema.ResourceData, m interface{}) error {
	vxcId := d.Get("vxc_id").(string)
	d.SetId(vxcId)
	vxcDetails, retrievalErr := vxc.GetVXCDetails(d.Id())

	if retrievalErr != nil {
		return retrievalErr
	}

	d.Set("uid", vxcDetails.UID)
	d.Set("name", vxcDetails.Name)
	d.Set("vxc_type", vxcDetails.Type)
	d.Set("rate_limit", vxcDetails.RateLimit)
	d.Set("provisioning_status", vxcDetails.ProvisioningStatus)
	d.Set("created_by", vxcDetails.CreatedBy)
	d.Set("live_date", vxcDetails.LiveDate)
	d.Set("create_date", vxcDetails.CreateDate)
	d.Set("company_name", vxcDetails.CompanyName)
	d.Set("locked", vxcDetails.Locked)
	d.Set("admin_locked", vxcDetails.AdminLocked)

	aEndConfiguration := []interface{}{map[string]interface{}{
		"port_id":   vxcDetails.AEndConfiguration.UID,
		"owner_uid": vxcDetails.AEndConfiguration.OwnerUID,
		"name":      vxcDetails.AEndConfiguration.Name,
		"location":  vxcDetails.AEndConfiguration.Location,
		"vlan":      vxcDetails.AEndConfiguration.VLAN,
	}}

	if aEndErr := d.Set("a_end", aEndConfiguration); aEndErr != nil {
		return aEndErr
	}

	bEndConfiguration := []interface{}{map[string]interface{}{
		"port_id":   vxcDetails.BEndConfiguration.UID,
		"owner_uid": vxcDetails.BEndConfiguration.OwnerUID,
		"name":      vxcDetails.BEndConfiguration.Name,
		"location":  vxcDetails.BEndConfiguration.Location,
		"vlan":      vxcDetails.BEndConfiguration.VLAN,
	}}

	if bEndErr := d.Set("b_end", bEndConfiguration); bEndErr != nil {
		return bEndErr
	}

	return nil
}
