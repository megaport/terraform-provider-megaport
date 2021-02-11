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

package resource_megaport

import (
	"github.com/megaport/megaportgo/vxc"
	"github.com/megaport/terraform-provider-megaport/schema_megaport"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func MegaportVXC() *schema.Resource {
	return &schema.Resource{
		Create: resourceMegaportVXCCreate,
		Read:   ResourceMegaportVXCRead,
		Update: ResourceMegaportVXCUpdate,
		Delete: ResourceMegaportVXCDelete,
		Schema: schema_megaport.ResourceVXCSchema(),
	}
}

func resourceMegaportVXCCreate(d *schema.ResourceData, m interface{}) error {
	vxcName := d.Get("vxc_name").(string)
	rateLimit := d.Get("rate_limit").(int)

	aEndConfiguration := d.Get("a_end").(*schema.Set).List()[0].(map[string]interface{})
	aEndPortId := aEndConfiguration["port_id"].(string)
	aEndVLAN := aEndConfiguration["requested_vlan"].(int)

	bEndConfiguration := d.Get("b_end").(*schema.Set).List()[0].(map[string]interface{})
	bEndPortId := bEndConfiguration["port_id"].(string)
	bEndVLAN := bEndConfiguration["requested_vlan"].(int)

	vxcId, vxcErr := vxc.BuyVXC(aEndPortId, bEndPortId, vxcName, rateLimit, aEndVLAN, bEndVLAN)

	if vxcErr != nil {
		return vxcErr
	}

	d.SetId(vxcId)
	vxc.WaitForVXCProvisioning(vxcId)
	return ResourceMegaportVXCRead(d, m)
}

func ResourceMegaportVXCRead(d *schema.ResourceData, m interface{}) error {
	vxcDetails, retrievalErr := vxc.GetVXCDetails(d.Id())
	aVlan := vxcDetails.AEndConfiguration.VLAN
	bVlan := vxcDetails.BEndConfiguration.VLAN

	if aEndConfiguration, ok := d.GetOk("a_end"); ok {
		aVlan = aEndConfiguration.(*schema.Set).List()[0].(map[string]interface{})["requested_vlan"].(int)
	}

	if bEndConfiguration, ok := d.GetOk("b_end"); ok && d.Get("vxc_internal_type") == "vxc" {
		bVlan = bEndConfiguration.(*schema.Set).List()[0].(map[string]interface{})["requested_vlan"].(int)
	}

	if retrievalErr != nil {
		return retrievalErr
	}

	d.Set("uid", vxcDetails.UID)
	d.Set("service_id", vxcDetails.ServiceID)
	d.Set("vxc_name", vxcDetails.Name)
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
		"port_id":        vxcDetails.AEndConfiguration.UID,
		"owner_uid":      vxcDetails.AEndConfiguration.OwnerUID,
		"name":           vxcDetails.AEndConfiguration.Name,
		"location":       vxcDetails.AEndConfiguration.Location,
		"assigned_vlan":  vxcDetails.AEndConfiguration.VLAN,
		"requested_vlan": aVlan,
	}}

	if aEndErr := d.Set("a_end", aEndConfiguration); aEndErr != nil {
		return aEndErr
	}

	bEndConfiguration := []interface{}{map[string]interface{}{
		"port_id":        vxcDetails.BEndConfiguration.UID,
		"owner_uid":      vxcDetails.BEndConfiguration.OwnerUID,
		"name":           vxcDetails.BEndConfiguration.Name,
		"location":       vxcDetails.BEndConfiguration.Location,
		"assigned_vlan":  vxcDetails.AEndConfiguration.VLAN,
		"requested_vlan": bVlan,
	}}

	if d.Get("vxc_internal_type") == "vxc" {
		bEndConfiguration[0].(map[string]interface{})["requested_vlan"] = bVlan
	}

	if bEndErr := d.Set("b_end", bEndConfiguration); bEndErr != nil {
		return bEndErr
	}

	return nil
}

// TODO: See if we can do a .HasChange on the subitem for vlans.
//       ** This is the expected behaviour of StackSet - the item is hashed and changes are across
//       ** the whole StackSet. I need to think about the structure of data to pick up the modifications better.
func ResourceMegaportVXCUpdate(d *schema.ResourceData, m interface{}) error {
	aVlan := 0
	bVlan := 0

	if aEndConfiguration, ok := d.GetOk("a_end"); ok {
		if newVlan, aOk := aEndConfiguration.(*schema.Set).List()[0].(map[string]interface{})["requested_vlan"].(int); aOk {
			aVlan = newVlan
		}
	}

	if bEndConfiguration, ok := d.GetOk("b_end"); ok {
		if newVlan, bOk := bEndConfiguration.(*schema.Set).List()[0].(map[string]interface{})["requested_vlan"].(int); bOk {
			bVlan = newVlan
		}
	}

	if d.HasChange("vxc_name") || d.HasChange("rate_limit") || d.HasChange("a_end") || d.HasChange("b_end") {
		vxc.UpdateVXC(d.Id(), d.Get("vxc_name").(string),
			d.Get("rate_limit").(int),
			aVlan,
			bVlan)
		vxc.WaitForVXCUpdated(d.Id(), d.Get("vxc_name").(string),
			d.Get("rate_limit").(int),
			aVlan,
			bVlan)
	}

	return ResourceMegaportVXCRead(d, m)
}

func ResourceMegaportVXCDelete(d *schema.ResourceData, m interface{}) error {
	vxc.DeleteVXC(d.Id(), true)
	return nil
}
