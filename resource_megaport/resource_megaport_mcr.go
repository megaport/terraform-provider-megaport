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
	"github.com/megaport/megaportgo/mcr"
	"github.com/megaport/terraform-provider-megaport/schema_megaport"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func MegaportAWS() *schema.Resource {
	return &schema.Resource{
		Create: resourceMegaportMCRCreate,
		Read:   resourceMegaportMCRRead,
		Update: resourceMegaportMCRUpdate,
		Delete: resourceMegaportMCRDelete,
		Schema: schema_megaport.ResourceMegaportMCRSchema(),
	}
}

func resourceMegaportMCRCreate(d *schema.ResourceData, m interface{}) error {
	routerConfiguration := d.Get("router").(*schema.Set).List()[0].(map[string]interface{})
	locationId := d.Get("location_id").(int)
	mcrName := d.Get("mcr_name").(string)
	portSpeed := routerConfiguration["port_speed"].(int)
	mcrAsn := routerConfiguration["requested_asn"].(int)
	mcrId, buyErr := mcr.BuyMCR(locationId, mcrName, portSpeed, mcrAsn)

	if buyErr != nil {
		return buyErr
	}

	d.SetId(mcrId)
	mcr.WaitForMcrProvisioning(mcrId)
	return resourceMegaportMCRRead(d, m)
}

func resourceMegaportMCRRead(d *schema.ResourceData, m interface{}) error {
	mcrDetails, retrievalErr := mcr.GetMCRDetails(d.Id())
	myConf := d.Get("router").(*schema.Set).List()[0].(map[string]interface{})
	mcrAsn := myConf["requested_asn"].(int)

	if retrievalErr != nil {
		return retrievalErr
	}

	d.Set("uid", mcrDetails.UID)
	d.Set("mcr_name", mcrDetails.Name)
	d.Set("type", mcrDetails.Type)
	d.Set("provisioning_status", mcrDetails.ProvisioningStatus)
	d.Set("create_date", mcrDetails.CreateDate)
	d.Set("created_by", mcrDetails.CreatedBy)
	d.Set("live_date", mcrDetails.LiveDate)
	d.Set("market", mcrDetails.Market)
	d.Set("location_id", mcrDetails.LocationID)
	d.Set("marketplace_visibility", mcrDetails.MarketplaceVisibility)
	d.Set("company_name", mcrDetails.CompanyName)
	d.Set("locked", mcrDetails.Locked)
	d.Set("admin_locked", mcrDetails.AdminLocked)

	routerConfiguration := []interface{}{map[string]interface{}{
		"assigned_asn":  mcrDetails.Resources.VirtualRouter.ASN,
		"requested_asn": mcrAsn,
		"port_speed":    mcrDetails.Resources.VirtualRouter.Speed,
	}}

	if routerError := d.Set("router", routerConfiguration); routerError != nil {
		return routerError
	}

	return nil
}

func resourceMegaportMCRUpdate(d *schema.ResourceData, m interface{}) error {
	if d.HasChange("mcr_name") || d.HasChange("marketplace_visibility") {
		_, nameErr := mcr.ModifyMCR(d.Id(),
			d.Get("mcr_name").(string),
			"",
			d.Get("marketplace_visibility").(bool))

		if nameErr != nil {
			return nameErr
		}
	}

	return resourceMegaportMCRRead(d, m)
}

func resourceMegaportMCRDelete(d *schema.ResourceData, m interface{}) error {
	mcr.DeleteMCR(d.Id(), true)
	return nil
}
