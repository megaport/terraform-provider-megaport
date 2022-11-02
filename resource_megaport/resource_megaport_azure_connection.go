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

func MegaportAzureConnection() *schema.Resource {
	return &schema.Resource{
		Create: resourceMegaportAzureConnectionCreate,
		Read:   resourceMegaportAzureConnectionRead,
		Update: resourceMegaportAzureConnectionUpdate,
		Delete: resourceMegaportAzureConnectionDelete,
		Schema: schema_megaport.ResourceAzureConnectionVXCSchema(),
	}
}

func resourceMegaportAzureConnectionCreate(d *schema.ResourceData, m interface{}) error {
	vxc := m.(*terraform_utility.MegaportClient).Vxc

	// assemble a end
	aEndConfiguration, aEndPortId, _ := ResourceMegaportVXCCreate_generate_AEnd(d, m)

	// csp settings
	cspSettings := d.Get("csp_settings").(*schema.Set).List()[0].(map[string]interface{})
	rateLimit := d.Get("rate_limit").(int)
	serviceKey := cspSettings["service_key"].(string)

	// peerings
	var peerings []types.PartnerOrderAzurePeeringConfig
	if len(cspSettings["private_peering"].(*schema.Set).List()) > 0 {
		private_peering := cspSettings["private_peering"].(*schema.Set).List()[0].(map[string]interface{})
		new_private_peering := types.PartnerOrderAzurePeeringConfig{
			Type:            "private",
			PeerASN:         private_peering["peer_asn"].(string),
			PrimarySubnet:   private_peering["primary_subnet"].(string),
			SecondarySubnet: private_peering["secondary_subnet"].(string),
			SharedKey:       private_peering["shared_key"].(string),
			VLAN:            private_peering["requested_vlan"].(int),
		}
		peerings = append(peerings, new_private_peering)
	} else if cspSettings["auto_create_private_peering"].(bool) {
		new_private_peering := types.PartnerOrderAzurePeeringConfig{
			Type: "private",
		}
		peerings = append(peerings, new_private_peering)
	}
	if len(cspSettings["microsoft_peering"].(*schema.Set).List()) > 0 {
		microsoft_peering := cspSettings["microsoft_peering"].(*schema.Set).List()[0].(map[string]interface{})
		new_microsoft_peering := types.PartnerOrderAzurePeeringConfig{
			Type:            "microsoft",
			PeerASN:         microsoft_peering["peer_asn"].(string),
			PrimarySubnet:   microsoft_peering["primary_subnet"].(string),
			SecondarySubnet: microsoft_peering["secondary_subnet"].(string),
			Prefixes:        microsoft_peering["public_prefixes"].(string),
			SharedKey:       microsoft_peering["shared_key"].(string),
			VLAN:            microsoft_peering["requested_vlan"].(int),
		}
		peerings = append(peerings, new_microsoft_peering)
	} else if cspSettings["auto_create_microsoft_peering"].(bool) {
		new_microsoft_peering := types.PartnerOrderAzurePeeringConfig{
			Type: "microsoft",
		}
		peerings = append(peerings, new_microsoft_peering)
	}

	// get partner port
	partnerPortId, partnerLookupErr := vxc.LookupPartnerPorts(serviceKey, rateLimit, vxc_service.PARTNER_AZURE, "")
	if partnerLookupErr != nil {
		return partnerLookupErr
	}

	// get partner config
	partnerConfig, partnerConfigErr := vxc.MarshallPartnerConfig(serviceKey, vxc_service.PARTNER_AZURE, peerings)
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
	return resourceMegaportAzureConnectionRead(d, m)
}

func resourceMegaportAzureConnectionRead(d *schema.ResourceData, m interface{}) error {
	return ResourceMegaportVXCRead(d, m)
}

func resourceMegaportAzureConnectionUpdate(d *schema.ResourceData, m interface{}) error {
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

	return resourceMegaportAzureConnectionRead(d, m)
}

func resourceMegaportAzureConnectionDelete(d *schema.ResourceData, m interface{}) error {
	return ResourceMegaportVXCDelete(d, m)
}
