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
	"github.com/megaport/megaportgo/port"
	"github.com/megaport/terraform-provider-megaport/schema_megaport"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func MegaportPort() *schema.Resource {
	return &schema.Resource{
		Read:   dataMegaportPortRead,
		Schema: schema_megaport.DataPortSchema(),
	}
}

func dataMegaportPortRead(d *schema.ResourceData, m interface{}) error {
	portId := d.Get("port_id").(string)
	d.SetId(portId)
	portDetails, retrievalErr := port.GetPortDetails(d.Id())

	if retrievalErr != nil {
		return retrievalErr
	}

	d.Set("uid", portDetails.UID)
	d.Set("port_name", portDetails.Name)
	d.Set("type", portDetails.Type)
	d.Set("provisioning_status", portDetails.ProvisioningStatus)
	d.Set("create_date", portDetails.CreateDate)
	d.Set("created_by", portDetails.CreatedBy)
	d.Set("port_speed", portDetails.PortSpeed)
	d.Set("live_date", portDetails.LiveDate)
	d.Set("market_code", portDetails.Market)
	d.Set("location_id", portDetails.LocationID)
	d.Set("marketplace_visibility", portDetails.MarketplaceVisibility)
	d.Set("company_name", portDetails.CompanyName)
	d.Set("term", portDetails.ContractTermMonths)
	d.Set("lag_primary", portDetails.LAGPrimary)
	d.Set("lag_id", portDetails.LAGID)
	d.Set("locked", portDetails.Locked)
	d.Set("admin_locked", portDetails.AdminLocked)

	return nil
}
