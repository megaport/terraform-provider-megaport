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
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/megaport/megaportgo/types"
	"github.com/megaport/terraform-provider-megaport/schema_megaport"
	"github.com/megaport/terraform-provider-megaport/terraform_utility"
)

func MegaportAWS() *schema.Resource {
	return &schema.Resource{
		Create: resourceMegaportMCRCreate,
		Read:   resourceMegaportMCRRead,
		Update: resourceMegaportMCRUpdate,
		Delete: resourceMegaportMCRDelete,
		Schema: schema_megaport.ResourceMegaportMCRSchema(),
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
	}
}

func resourceMegaportMCRCreate(d *schema.ResourceData, m interface{}) error {
	mcr := m.(*terraform_utility.MegaportClient).Mcr

	routerConfiguration := d.Get("router").(*schema.Set).List()[0].(map[string]interface{})
	locationId := d.Get("location_id").(int)
	mcrName := d.Get("mcr_name").(string)
	term := d.Get("term").(int)
	portSpeed := routerConfiguration["port_speed"].(int)
	mcrAsn := routerConfiguration["requested_asn"].(int)
	mcrId, buyErr := mcr.BuyMCR(locationId, mcrName, term, portSpeed, mcrAsn)

	if buyErr != nil {
		return buyErr
	}

	d.SetId(mcrId)
	mcr.WaitForMcrProvisioning(mcrId)

	for i := 0; i < len(d.Get("prefix_filter_list").(*schema.Set).List()); i++ {
		prefixFilterList := d.Get("prefix_filter_list").(*schema.Set).List()[i].(map[string]interface{})
		var prefixFilterEntries []types.MCRPrefixListEntry
		for x := 0; x < len(prefixFilterList["entry"].(*schema.Set).List()); x++ {
			entries := prefixFilterList["entry"].(*schema.Set).List()[x].(map[string]interface{})
			prefixFilterEntry := types.MCRPrefixListEntry{
				Action: entries["action"].(string),
				Prefix: entries["prefix"].(string),
				Ge:     entries["range_min"].(int),
				Le:     entries["range_max"].(int),
			}
			prefixFilterEntries = append(prefixFilterEntries, prefixFilterEntry)
		}

		validatedPrefixFilterList := types.MCRPrefixFilterList{
			Description:   prefixFilterList["name"].(string),
			AddressFamily: prefixFilterList["address_family"].(string),
			Entries:       prefixFilterEntries,
		}

		mcr.CreatePrefixFilterList(mcrId, validatedPrefixFilterList)
	}
	return resourceMegaportMCRRead(d, m)
}

func resourceMegaportMCRRead(d *schema.ResourceData, m interface{}) error {
	mcr := m.(*terraform_utility.MegaportClient).Mcr
	mcrDetails, retrievalErr := mcr.GetMCRDetails(d.Id())
	isImport := len(d.Get("router").(*schema.Set).List()) == 0
	var myConf map[string]interface{}
	mcrAsn := mcrDetails.Resources.VirtualRouter.ASN

	if !isImport {
		myConf = d.Get("router").(*schema.Set).List()[0].(map[string]interface{})
		mcrAsn = myConf["requested_asn"].(int)
	}

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
	d.Set("term", mcrDetails.ContractTermMonths)
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
	mcr := m.(*terraform_utility.MegaportClient).Mcr

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
	mcr := m.(*terraform_utility.MegaportClient).Mcr

	deleteSuccess, deleteError := mcr.DeleteMCR(d.Id(), true)

	if deleteSuccess {
		return nil
	} else {
		return errors.New(fmt.Sprintf("Error deleting resource %s: %s", d.Id(), deleteError))
	}

}
