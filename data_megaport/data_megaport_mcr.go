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
	"github.com/megaport/terraform-provider-megaport/schema_megaport"
	"github.com/megaport/terraform-provider-megaport/terraform_utility"
)

func MegaportMCR() *schema.Resource {
	return &schema.Resource{
		Read:   DataMegaportMCRRead,
		Schema: schema_megaport.DataMegaportMCRSchema(),
	}
}

func DataMegaportMCRRead(d *schema.ResourceData, m interface{}) error {
	mcrId := d.Get("mcr_id").(string)
	d.SetId(mcrId)

	mcr := m.(*terraform_utility.MegaportClient).Mcr

	mcrDetails, retrievalErr := mcr.GetMCRDetails(d.Id())

	if retrievalErr != nil {
		return retrievalErr
	}

	d.Set("uid", mcrDetails.UID)
	d.Set("mcr_name", mcrDetails.Name)
	d.Set("type", mcrDetails.Type)
	d.Set("provisioning_status", mcrDetails.ProvisioningStatus)
	d.Set("create_date", mcrDetails.CreateDate)
	d.Set("created_by", mcrDetails.CreatedBy)
	d.Set("port_speed", mcrDetails.PortSpeed)
	d.Set("live_date", mcrDetails.LiveDate)
	d.Set("market", mcrDetails.Market)
	d.Set("location_id", mcrDetails.LocationID)
	d.Set("marketplace_visibility", mcrDetails.MarketplaceVisibility)
	d.Set("company_name", mcrDetails.CompanyName)
	d.Set("term", mcrDetails.ContractTermMonths)
	d.Set("locked", mcrDetails.Locked)
	d.Set("admin_locked", mcrDetails.AdminLocked)

	virtualRouterConfiguration := []interface{}{map[string]interface{}{
		"assigned_asn": mcrDetails.Resources.VirtualRouter.ASN,
		"port_speed":   mcrDetails.Resources.VirtualRouter.Speed,
	}}

	if routerErr := d.Set("router", virtualRouterConfiguration); routerErr != nil {
		return routerErr
	}

	return nil
}
