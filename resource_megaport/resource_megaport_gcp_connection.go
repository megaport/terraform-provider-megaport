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
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	vxc_service "github.com/megaport/megaportgo/service/vxc"
	"github.com/megaport/megaportgo/types"
	"github.com/megaport/terraform-provider-megaport/schema_megaport"
	"github.com/megaport/terraform-provider-megaport/terraform_utility"
)

func MegaportGcpConnection() *schema.Resource {
	return &schema.Resource{
		Create: resourceMegaportGcpConnectionCreate,
		Read:   resourceMegaportGcpConnectionRead,
		Update: resourceMegaportGcpConnectionUpdate,
		Delete: resourceMegaportGcpConnectionDelete,
		Schema: schema_megaport.ResourceGcpConnectionVXCSchema(),
	}
}

func resourceMegaportGcpConnectionCreate(d *schema.ResourceData, m interface{}) error {
	var buyErr error
	vxc := m.(*terraform_utility.MegaportClient).Vxc

	// assemble a end
	aEndConfiguration, aEndPortId, _ := ResourceMegaportVXCCreate_generate_AEnd(d, m)

	// csp settings
	cspSettings := d.Get("csp_settings").(*schema.Set).List()[0].(map[string]interface{})
	rateLimit := d.Get("rate_limit").(int)
	pairingKey := cspSettings["pairing_key"].(string)
	requestedProductID := cspSettings["requested_product_id"].(string)

	// get partner port
	partnerPortId, partnerLookupErr := vxc.LookupPartnerPorts(pairingKey, rateLimit, vxc_service.PARTNER_GOOGLE, requestedProductID)
	if partnerLookupErr != nil {
		return partnerLookupErr
	}

	// get partner config
	partnerConfig, partnerConfigErr := vxc.MarshallPartnerConfig(pairingKey, vxc_service.PARTNER_GOOGLE, nil)
	if partnerConfigErr != nil {
		return partnerConfigErr
	}

	// assemble b end
	bEndConfiguration := types.PartnerOrderBEndConfiguration{
		PartnerPortID: partnerPortId,
		PartnerConfig: partnerConfig,
	}

	vxcId, buyErr := vxc.BuyPartnerVXC(
		aEndPortId,
		d.Get("vxc_name").(string),
		rateLimit,
		aEndConfiguration,
		bEndConfiguration,
	)

	if buyErr != nil {
		return buyErr
	}

	d.SetId(vxcId)
	vxc.WaitForVXCProvisioning(vxcId)
	time.Sleep(60 * time.Second) // wait so that the vLANs will be available.
	return resourceMegaportGcpConnectionRead(d, m)
}

func resourceMegaportGcpConnectionRead(d *schema.ResourceData, m interface{}) error {
	return ResourceMegaportVXCRead(d, m)
}

func resourceMegaportGcpConnectionUpdate(d *schema.ResourceData, m interface{}) error {
	vxc := m.(*terraform_utility.MegaportClient).Vxc
	aVlan := 0

	if aEndConfiguration, ok := d.GetOk("a_end"); ok {
		if newVlan, aOk := aEndConfiguration.(*schema.Set).List()[0].(map[string]interface{})["requested_vlan"].(int); aOk {
			aVlan = newVlan
		}
	}

	if d.HasChange("vxc_name") || d.HasChange("rate_limit") || d.HasChange("a_end") {
		_, updateErr := vxc.UpdateVXC(d.Id(), d.Get("vxc_name").(string),
			d.Get("rate_limit").(int),
			aVlan,
			0)

		if updateErr != nil {
			return updateErr
		}

		vxc.WaitForVXCUpdated(d.Id(), d.Get("vxc_name").(string),
			d.Get("rate_limit").(int),
			aVlan,
			0)
	}

	return resourceMegaportGcpConnectionRead(d, m)
}

func resourceMegaportGcpConnectionDelete(d *schema.ResourceData, m interface{}) error {
	return ResourceMegaportVXCDelete(d, m)
}
